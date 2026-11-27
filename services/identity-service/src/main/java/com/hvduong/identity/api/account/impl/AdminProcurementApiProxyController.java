package com.hvduong.identity.api.account.impl;

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
import java.net.URLDecoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Flow;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

/** Cookie sessions are resolved by BffSessionAuthFilter. Procurement remains the
 * authority for warehouse ownership, optimistic versions and separation of duties. */
@RestController
@RequiredArgsConstructor
public class AdminProcurementApiProxyController {
    private final IdentityPrincipalMapper principals;
    @Value("${app.procurement-api.enabled:false}") private boolean enabled;
    @Value("${app.procurement-api.url:}") private String upstreamUrl;
    @Value("${app.procurement-api.allow-insecure-http:false}") private boolean allowInsecureHttp;
    private static final String PREFIX = "/api/v1/admin/procurement-api";
    private static final String ID = "(?!00000000-0000-0000-0000-000000000000)[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}";
    private final ObjectMapper json = new ObjectMapper();
    private final HttpClient client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(3))
            .followRedirects(HttpClient.Redirect.NEVER).build();

    @PreAuthorize("hasAnyAuthority('PROCUREMENT_VIEW','PROCUREMENT_MANAGE','PROCUREMENT_APPROVE','PROCUREMENT_RECEIVE','PROCUREMENT_VARIANCE_RESOLVE')")
    @RequestMapping(value = PREFIX + "/**", method = {RequestMethod.GET, RequestMethod.POST, RequestMethod.PUT, RequestMethod.PATCH})
    public ResponseEntity<byte[]> forward(@AuthenticationPrincipal Jwt jwt, HttpServletRequest request) throws IOException {
        if (!enabled) return error(404, "NOT_FOUND");
        if (jwt == null || jwt.getIssuer() == null) return error(401, "UNAUTHENTICATED");
        var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
        if (!"EMPLOYEE".equals(jwt.getClaimAsString("principal_type")) || principal == null
                || !"EMPLOYEE".equals(principal.getPrincipalType()) || !"ACTIVE".equals(principal.getProvisioningStatus())
                || !principal.getPrincipalId().toString().equals(jwt.getClaimAsString("user_id"))) return error(403, "FORBIDDEN");
        String uri = request.getRequestURI();
        if (!uri.startsWith(PREFIX + "/")) return error(404, "NOT_FOUND");
        String path = uri.substring(PREFIX.length() + 1);
        String method = request.getMethod();
        if (!allowedRoute(method, path)) return error(404, "NOT_FOUND");
        if (!permitted(jwt, method, path)) return error(403, "FORBIDDEN");
        String query = request.getQueryString();
        if (!allowedQuery(path, query) || (!method.equals("GET") && query != null && !query.isEmpty())) return error(400, "INVALID_QUERY");
        URI base;
        try {
            base = URI.create(upstreamUrl == null ? "" : upstreamUrl);
            if (base.getHost() == null || base.getRawUserInfo() != null || base.getRawQuery() != null || base.getRawFragment() != null
                    || !(base.getRawPath().isEmpty() || base.getRawPath().equals("/"))
                    || !("https".equals(base.getScheme()) || allowInsecureHttp && "http".equals(base.getScheme()))) throw new IllegalArgumentException();
        } catch (IllegalArgumentException ex) { return error(503, "PROCUREMENT_UPSTREAM_UNAVAILABLE"); }
        byte[] body = new byte[0];
        String key = request.getHeader("Idempotency-Key");
        if (!method.equals("GET")) {
            if (key == null || !key.matches("[!-~]{16,128}")) return error(400, "IDEMPOTENCY_KEY_REQUIRED");
            String contentType = request.getContentType();
            if (contentType == null || !contentType.split(";", 2)[0].trim().equals("application/json")) return error(415, "UNSUPPORTED_MEDIA_TYPE");
            body = request.getInputStream().readNBytes(65537);
            if (body.length > 65536) return error(413, "REQUEST_TOO_LARGE");
            try { if (!json.readTree(body).isObject()) return error(400, "INVALID_ARGUMENT"); }
            catch (IOException | NullPointerException ex) { return error(400, "INVALID_ARGUMENT"); }
        }
        var target = URI.create(base.toString().replaceAll("/$", "") + "/api/v1/admin/procurement/" + path + (query == null ? "" : "?" + query));
        var builder = HttpRequest.newBuilder(target).timeout(Duration.ofSeconds(10))
                .header("Authorization", "Bearer " + jwt.getTokenValue()).header("Accept", "application/json");
        if (!method.equals("GET")) builder.header("Content-Type", "application/json").header("Idempotency-Key", key);
        var pending = client.sendAsync(builder.method(method, HttpRequest.BodyPublishers.ofByteArray(body)).build(), info -> new LimitedBody());
        try {
            var response = pending.get(10, TimeUnit.SECONDS);
            int status = response.statusCode();
            if (status == 401 || status == 403) return error(status, status == 401 ? "UNAUTHENTICATED" : "FORBIDDEN");
            if (status >= 300 && status < 400 || !response.headers().firstValue("content-type").orElse("").split(";", 2)[0].trim().equals("application/json")) return error(502, "PROCUREMENT_INVALID_RESPONSE");
            try { if (!json.readTree(response.body()).isObject()) return error(502, "PROCUREMENT_INVALID_RESPONSE"); }
            catch (IOException | NullPointerException ex) { return error(502, "PROCUREMENT_INVALID_RESPONSE"); }
            var result = ResponseEntity.status(status).header("Cache-Control", "no-store").header("Content-Type", "application/json");
            response.headers().firstValue("retry-after").filter(value -> value.matches("[0-9]{1,6}")).ifPresent(value -> result.header("Retry-After", value));
            return result.body(response.body());
        } catch (TimeoutException ex) {
            pending.cancel(true); return error(503, "PROCUREMENT_UPSTREAM_UNAVAILABLE");
        } catch (ExecutionException ex) { return error(502, "PROCUREMENT_UPSTREAM_UNAVAILABLE"); }
        catch (InterruptedException ex) { pending.cancel(true); Thread.currentThread().interrupt(); return error(503, "PROCUREMENT_UPSTREAM_UNAVAILABLE"); }
    }

    static boolean allowedRoute(String method, String path) {
        return switch (method) {
            case "GET" -> path.matches("(?:suppliers|materials|purchase-orders)(?:/" + ID + ")?")
                    || path.matches("(?:suppliers|materials)/" + ID + "/offers")
                    || path.matches("suppliers/" + ID + "/contracts") || path.matches("contracts/" + ID)
                    || path.matches("purchase-orders/" + ID + "/revisions") || path.matches("goods-receipts/" + ID);
            case "POST" -> path.matches("suppliers|materials|purchase-orders|goods-receipts")
                    || path.matches("suppliers/" + ID + "/(?:offers|contracts)")
                    || path.matches("contracts/" + ID + "/(?:activate|terminate)")
                    || path.matches("purchase-orders/" + ID + "/(?:submit|approve|reject|send|supplier-response|revisions|cancel|close)")
                    || path.matches("goods-receipts/" + ID + "/inspect") || path.matches("goods-receipts/" + ID + "/variances/" + ID + "/resolve");
            case "PUT" -> path.matches("(?:suppliers|materials|purchase-orders)/" + ID) || path.matches("materials/" + ID + "/preferred-supplier");
            case "PATCH" -> path.matches("(?:suppliers|materials)/" + ID + "/status") || path.matches("suppliers/" + ID + "/offers/" + ID + "/status");
            default -> false;
        };
    }

    private boolean permitted(Jwt jwt, String method, String path) {
        String permission = path.contains("contracts") ? "PROCUREMENT_MANAGE" : method.equals("GET") ? "PROCUREMENT_VIEW"
                : path.matches(".*/(?:approve|reject)") ? "PROCUREMENT_APPROVE" : path.contains("/variances/") ? "PROCUREMENT_VARIANCE_RESOLVE"
                : path.startsWith("goods-receipts") ? "PROCUREMENT_RECEIVE" : "PROCUREMENT_MANAGE";
        Object raw = jwt.getClaims().get("permission_grants");
        if (!(raw instanceof List<?> grants)) return false;
        for (Object item : grants) {
            if (!(item instanceof String grant)) continue;
            if (grant.equals(permission + "#GLOBAL")) return true;
            // Exact receipt ownership is checked by Procurement from persisted data, not the URL or browser.
            String scopedPermission = method.equals("GET") && path.startsWith("goods-receipts/") ? "PROCUREMENT_RECEIVE" : permission;
            if (path.startsWith("goods-receipts") && (grant.equals(scopedPermission + "#GLOBAL")
                    || grant.matches(scopedPermission + "#WAREHOUSE#[^#\\s]{1,128}"))) return true;
        }
        return false;
    }

    private boolean allowedQuery(String path, String raw) {
        if (raw == null || raw.isEmpty()) return true;
        if (raw.length() > 1024) return false;
        boolean page = path.matches("suppliers|materials|purchase-orders") || path.endsWith("/contracts") || path.endsWith("/revisions");
        Set<String> names = new HashSet<>();
        try {
            for (String pair : raw.split("&", -1)) {
                String[] parts = pair.split("=", 2);
                if (parts.length != 2) return false;
                String name = URLDecoder.decode(parts[0], StandardCharsets.UTF_8);
                String value = URLDecoder.decode(parts[1], StandardCharsets.UTF_8);
                if (!names.add(name) || value.length() > 128) return false;
                if (page && name.equals("limit") && value.matches("[1-9][0-9]{0,2}") && Integer.parseInt(value) <= 100) continue;
                if (page && name.equals("offset") && value.matches("0|[1-9][0-9]{0,8}")) continue;
                if (path.equals("purchase-orders") && Set.of("supplier_id", "status").contains(name) && value.matches("[A-Za-z0-9_-]{1,128}")) continue;
                return false;
            }
        } catch (IllegalArgumentException ex) { return false; }
        return true;
    }

    private ResponseEntity<byte[]> error(int status, String code) {
        byte[] body = ("{\"code\":\"" + code + "\",\"message\":\"Unable to process procurement request.\",\"requestId\":\"" + UUID.randomUUID() + "\",\"errors\":[]}").getBytes(StandardCharsets.UTF_8);
        return ResponseEntity.status(status).header("Cache-Control", "no-store").header("Content-Type", "application/json").body(body);
    }

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
            if (bytes > (1 << 20)) { done = true; subscription.cancel(); delegate.onError(new IOException("response limit")); }
            else delegate.onNext(buffers);
        }
        @Override public void onError(Throwable error) { if (!done) { done = true; delegate.onError(error); } }
        @Override public void onComplete() { if (!done) { done = true; delegate.onComplete(); } }
    }
}
