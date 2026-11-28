package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import jakarta.servlet.http.HttpServletRequest;
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
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** Cookie-to-bearer transport only. Traceability/Inventory retain business scope checks. */
@RestController
public class TraceabilityApiProxyController {
    private static final String TRACE = "/api/v1/admin/trace-api";
    private static final String INVENTORY = "/api/v1/admin/inventory-api";
    private static final String ID = "[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}";
    private final IdentityPrincipalMapper principals;
    private final String gateway;
    private final HttpClient client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(3))
            .followRedirects(HttpClient.Redirect.NEVER).build();

    public TraceabilityApiProxyController(IdentityPrincipalMapper principals,
            @Value("${app.customer-api.gateway-url:}") String gateway) {
        this.principals = principals;
        this.gateway = gateway;
    }

    static String targetPath(String method, String uri) {
        if (uri.startsWith(TRACE + "/")) {
            String path = uri.substring(TRACE.length());
            boolean allowed = switch (method) {
                case "GET" -> path.matches("/records(?:/by-batch|/" + ID + ")?|/batches/" + ID + "/context");
                case "POST" -> path.matches("/records|/records/" + ID + "/(?:revisions|suspend|archive)|/records/" + ID + "/revisions/[1-9][0-9]{0,8}/(?:submit|approve|reject|publish)");
                case "PATCH" -> path.matches("/records/" + ID + "/revisions/[1-9][0-9]{0,8}");
                default -> false;
            };
            return allowed ? "/api/v2/admin/trace" + path : null;
        }
        if ("GET".equals(method) && uri.equals(INVENTORY + "/workbench")) return "/api/v1/admin/inventory/workbench";
        if ("POST".equals(method) && uri.equals(INVENTORY + "/workbench/receipts")) return "/api/v1/admin/inventory/workbench/receipts";
        return null;
    }

    @RequestMapping({TRACE + "/**", INVENTORY + "/**"})
    public ResponseEntity<byte[]> forward(@AuthenticationPrincipal Jwt jwt, HttpServletRequest request) throws IOException {
        String requestId = UUID.randomUUID().toString();
        if (jwt == null) return fail(401, "UNAUTHENTICATED", requestId);
        if (jwt.getIssuer() == null) return fail(403, "PERMISSION_DENIED", requestId);
        var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
        if (principal == null || !"EMPLOYEE".equals(principal.getPrincipalType())
                || !"ACTIVE".equals(principal.getProvisioningStatus())
                || !principal.getPrincipalId().toString().equals(jwt.getClaimAsString("user_id"))) {
            return fail(403, "PERMISSION_DENIED", requestId);
        }
        String path = targetPath(request.getMethod(), request.getRequestURI());
        if (path == null) return fail(404, "NOT_FOUND", requestId);
        if (gateway == null || gateway.isBlank()) return fail(503, "UPSTREAM_UNAVAILABLE", requestId);
        byte[] body = request.getInputStream().readNBytes(262145);
        if (body.length > 262144) return fail(413, "PAYLOAD_TOO_LARGE", requestId);
        String query = request.getQueryString();
        if (query != null && query.length() > 4096) return fail(400, "INVALID_ARGUMENT", requestId);
        URI target;
        try { target = URI.create(gateway.replaceAll("/$", "") + path + (query == null ? "" : "?" + query)); }
        catch (IllegalArgumentException error) { return fail(400, "INVALID_ARGUMENT", requestId); }
        var outbound = HttpRequest.newBuilder(target).timeout(Duration.ofSeconds(8))
                .header("Authorization", "Bearer " + jwt.getTokenValue()).header("X-Request-Id", requestId)
                .method(request.getMethod(), body.length == 0 ? HttpRequest.BodyPublishers.noBody() : HttpRequest.BodyPublishers.ofByteArray(body));
        for (String name : Set.of("Accept", "Content-Type", "Idempotency-Key", "If-Match", "Origin")) {
            String value = request.getHeader(name);
            if (value != null) outbound.header(name, value);
        }
        var pending = client.sendAsync(outbound.build(), info -> new LimitedBody());
        try {
            var upstream = pending.get(8, TimeUnit.SECONDS);
            var response = ResponseEntity.status(upstream.statusCode()).header("Cache-Control", "no-store");
            for (String name : Set.of("content-type", "x-request-id", "retry-after", "etag")) {
                upstream.headers().firstValue(name).ifPresent(value -> response.header(name, value));
            }
            return response.body(upstream.body());
        } catch (TimeoutException error) {
            pending.cancel(true);
            return fail(503, "UPSTREAM_UNAVAILABLE", requestId);
        } catch (ExecutionException error) {
            return fail(error.getCause() instanceof BodyTooLarge ? 502 : 503, "UPSTREAM_UNAVAILABLE", requestId);
        } catch (InterruptedException error) {
            pending.cancel(true);
            Thread.currentThread().interrupt();
            return fail(503, "UPSTREAM_UNAVAILABLE", requestId);
        }
    }

    private static ResponseEntity<byte[]> fail(int status, String code, String id) {
        return ResponseEntity.status(status).header("Cache-Control", "no-store").header("X-Request-Id", id)
                .header("Content-Type", "application/json").body(("{\"code\":\"" + code
                        + "\",\"message\":\"Unable to process staff request.\",\"requestId\":\"" + id + "\"}").getBytes(StandardCharsets.UTF_8));
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
            if (bytes > 1048576) { done = true; subscription.cancel(); delegate.onError(new BodyTooLarge()); }
            else delegate.onNext(buffers);
        }
        @Override public void onError(Throwable error) { if (!done) { done = true; delegate.onError(error); } }
        @Override public void onComplete() { if (!done) { done = true; delegate.onComplete(); } }
    }
}
