package com.hvduong.identity.api.account.impl;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import jakarta.servlet.http.HttpServletRequest;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RestController;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.List;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Flow;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

/** Session filter resolves/refreshes the bearer; Shipping owns resource authorization. */
@RestController
@RequiredArgsConstructor
public class AdminShippingApiProxyController {
    private static final String PREFIX = "/api/v1/admin/shipping-api";
    private static final String ID = "[0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}";
    private final IdentityPrincipalMapper principals;
    private final ObjectMapper json;
    @Value("${app.shipping-api.enabled:false}") private boolean enabled;
    @Value("${app.shipping-api.url:}") private String upstreamUrl;
    @Value("${app.shipping-api.allow-insecure-http:false}") private boolean allowHttp;
    private final HttpClient client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(3))
            .followRedirects(HttpClient.Redirect.NEVER).build();

    @RequestMapping(value = PREFIX + "/**", method = {RequestMethod.GET, RequestMethod.POST})
    @PreAuthorize("hasAnyAuthority('SHIPMENT_VIEW#GLOBAL','SHIPMENT_ASSIGN#GLOBAL','SHIPMENT_EXECUTE#GLOBAL')")
    public ResponseEntity<byte[]> forward(@AuthenticationPrincipal Jwt jwt, HttpServletRequest request) throws IOException {
        if (!enabled) return denied(404);
        if (jwt == null || jwt.getIssuer() == null) return denied(401);
        var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
        if (principal == null || !"EMPLOYEE".equals(principal.getPrincipalType())
                || !"EMPLOYEE".equals(jwt.getClaimAsString("principal_type"))
                || !"ACTIVE".equals(principal.getProvisioningStatus())
                || !principal.getPrincipalId().toString().equals(jwt.getClaimAsString("user_id"))) return denied(403);
        if (!request.getRequestURI().startsWith(PREFIX + "/")) return denied(404);
        String path = request.getRequestURI().substring(PREFIX.length());
        boolean get = "GET".equals(request.getMethod());
        boolean post = "POST".equals(request.getMethod());
        String permission;
        boolean list;
        if (get && path.matches("/workbench/shipments(?:/" + ID + ")?")) {
            permission = "SHIPMENT_EXECUTE"; list = path.equals("/workbench/shipments");
        } else if (post && path.matches("/shipments/" + ID + "/(?:accept|progress)")) {
            permission = "SHIPMENT_EXECUTE"; list = false;
        } else if ((get && path.matches("/shipments/" + ID + "/assignment(?:/history)?"))
                || (post && path.matches("/shipments/" + ID + "/assignment"))) {
            permission = "SHIPMENT_ASSIGN"; list = false;
        } else if (get && path.matches("/shipments(?:/" + ID + ")?")) {
            permission = "SHIPMENT_VIEW"; list = path.equals("/shipments");
        } else return denied(404);
        List<String> grants = jwt.getClaimAsStringList("permission_grants");
        if (grants == null || !grants.contains(permission + "#GLOBAL")
                || (post && permission.equals("SHIPMENT_ASSIGN") && !grants.contains("USER_VIEW#GLOBAL"))) return denied(403);

        StringBuilder query = new StringBuilder();
        for (var entry : request.getParameterMap().entrySet()) {
            if (!list || !Set.of("limit", "offset").contains(entry.getKey()) || entry.getValue().length != 1
                    || !entry.getValue()[0].matches("[0-9]{1,5}")) return denied(400);
            if (!query.isEmpty()) query.append('&');
            query.append(entry.getKey()).append('=').append(entry.getValue()[0]);
        }
        byte[] body = new byte[0];
        String key = null;
        if (post) {
            if (request.getContentType() == null || !request.getContentType().split(";", 2)[0].trim().equalsIgnoreCase("application/json")) return denied(415);
            key = request.getHeader("Idempotency-Key");
            if (key == null || !key.matches("[!-~]{16,128}")) return denied(400);
            body = request.getInputStream().readNBytes(4097);
            if (body.length > 4096) return denied(413);
            JsonNode value;
            try { value = json.readTree(body); } catch (IOException ex) { return denied(400); }
            Set<String> fields = path.endsWith("/accept") ? Set.of("expected_version")
                    : path.endsWith("/progress") ? Set.of("assignment_version", "expected_revision", "status", "reason")
                    : Set.of("driver_id", "expected_version", "reason");
            if (value == null || !value.isObject()) return denied(400);
            var names = value.fieldNames();
            while (names.hasNext()) if (!fields.contains(names.next())) return denied(400);
        }
        URI base;
        try {
            base = URI.create(upstreamUrl == null ? "" : upstreamUrl);
            if (base.getHost() == null || base.getUserInfo() != null || base.getQuery() != null || base.getFragment() != null
                    || !(base.getPath().isEmpty() || base.getPath().equals("/"))
                    || !("https".equals(base.getScheme()) || (allowHttp && "http".equals(base.getScheme())))) return denied(503);
        } catch (IllegalArgumentException ex) { return denied(503); }
        URI target = URI.create(base.toString().replaceAll("/$", "") + "/api/v2/admin/shipping" + path
                + (query.isEmpty() ? "" : "?" + query));
        var outbound = HttpRequest.newBuilder(target).timeout(Duration.ofSeconds(8))
                .header("Authorization", "Bearer " + jwt.getTokenValue()).header("Accept", "application/json");
        if (post) outbound.header("Content-Type", "application/json").header("Idempotency-Key", key).POST(HttpRequest.BodyPublishers.ofByteArray(body));
        else outbound.GET();
        var pending = client.sendAsync(outbound.build(), info -> new LimitedBody());
        try {
            var response = pending.get(8, TimeUnit.SECONDS);
            int status = response.statusCode();
            if (status >= 300 && status < 400) return denied(502);
            if (status >= 400) return denied(status <= 599 ? status : 502);
            if (status != 200 || !response.headers().firstValue("content-type").orElse("").contains("application/json")) return denied(502);
            try { if (json.readTree(response.body()) == null) return denied(502); } catch (IOException ex) { return denied(502); }
            return ResponseEntity.ok().header("Cache-Control", "private, no-store").header("Content-Type", "application/json").body(response.body());
        } catch (TimeoutException ex) { pending.cancel(true); return denied(503); }
        catch (ExecutionException ex) { return denied(ex.getCause() instanceof BodyTooLarge ? 502 : 503); }
        catch (InterruptedException ex) { pending.cancel(true); Thread.currentThread().interrupt(); return denied(503); }
    }

    private ResponseEntity<byte[]> denied(int status) {
        String code = switch (status) {
            case 400 -> "INVALID_REQUEST"; case 401 -> "UNAUTHORIZED"; case 403 -> "FORBIDDEN";
            case 404 -> "NOT_FOUND"; case 409 -> "CONFLICT"; case 413 -> "PAYLOAD_TOO_LARGE";
            case 415 -> "UNSUPPORTED_MEDIA_TYPE"; case 422 -> "VALIDATION_FAILED";
            case 502 -> "SHIPPING_INVALID_RESPONSE"; default -> "SHIPPING_UNAVAILABLE";
        };
        return ResponseEntity.status(status).header("Cache-Control", "private, no-store").header("Content-Type", "application/json")
                .body(("{\"code\":\"" + code + "\",\"message\":\"Unable to complete Shipping request.\",\"requestId\":\"" + UUID.randomUUID() + "\"}").getBytes(StandardCharsets.UTF_8));
    }

    private static final class BodyTooLarge extends IOException {}
    private static final class LimitedBody implements HttpResponse.BodySubscriber<byte[]> {
        private final HttpResponse.BodySubscriber<byte[]> delegate = HttpResponse.BodySubscribers.ofByteArray();
        private Flow.Subscription subscription;
        private long bytes;
        private boolean done;
        @Override public CompletionStage<byte[]> getBody() { return delegate.getBody(); }
        @Override public void onSubscribe(Flow.Subscription value) { subscription = value; delegate.onSubscribe(value); }
        @Override public void onNext(List<ByteBuffer> buffers) {
            if (done) return;
            for (var buffer : buffers) bytes += buffer.remaining();
            if (bytes > (1 << 20)) { done = true; subscription.cancel(); delegate.onError(new BodyTooLarge()); }
            else delegate.onNext(buffers);
        }
        @Override public void onError(Throwable error) { if (!done) { done = true; delegate.onError(error); } }
        @Override public void onComplete() { if (!done) { done = true; delegate.onComplete(); } }
    }
}
