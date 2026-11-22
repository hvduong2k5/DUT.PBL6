package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import jakarta.servlet.http.HttpServletRequest;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.Set;
import java.nio.ByteBuffer;
import java.util.List;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.Flow;
import java.util.concurrent.TimeUnit;

/** Read-only cookie-to-bearer adapter; the Order service owns resource authorization. */
@RestController
@RequiredArgsConstructor
public class AdminOrderApiProxyController {
    private final IdentityPrincipalMapper principals;
    @Value("${app.customer-api.gateway-url:}") private String gatewayUrl;
    private final HttpClient client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(3))
            .followRedirects(HttpClient.Redirect.NEVER).build();
    private static final String PREFIX = "/api/v1/admin/order-api";
    private static final String MEDIA_TYPE = "application/vnd.omama.admin-order+json";
    private static final Set<String> RESPONSE_HEADERS = Set.of("content-type", "retry-after", "x-request-id");

    @GetMapping(PREFIX + "/**")
    public ResponseEntity<byte[]> forward(@AuthenticationPrincipal Jwt jwt, HttpServletRequest request) throws Exception {
        if (jwt == null) return denied(401);
        var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
        if (principal == null || !"EMPLOYEE".equals(principal.getPrincipalType())
                || !"ACTIVE".equals(principal.getProvisioningStatus())
                || !principal.getPrincipalId().toString().equals(jwt.getClaimAsString("user_id"))) return denied(403);
        if (!"GET".equals(request.getMethod()) || !request.getRequestURI().startsWith(PREFIX + "/")) return denied(404);
        String path = request.getRequestURI().substring(PREFIX.length());
        if (!path.matches("/orders(?:/queue|/[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12})?")) return denied(404);
        if (gatewayUrl == null || gatewayUrl.isBlank()) return denied(503);
        String query = request.getQueryString();
        URI target = URI.create(gatewayUrl.replaceAll("/$", "") + "/api/v1/admin" + path
                + (query == null ? "" : "?" + query));
        var outbound = HttpRequest.newBuilder(target).timeout(Duration.ofSeconds(8))
                .header("Authorization", "Bearer " + jwt.getTokenValue()).header("Accept", MEDIA_TYPE).GET().build();
        try {
            var pending = client.sendAsync(outbound, info -> new LimitedBody());
            HttpResponse<byte[]> upstream;
            try { upstream = pending.get(8, TimeUnit.SECONDS); }
            catch (java.util.concurrent.TimeoutException ex) { pending.cancel(true); return denied(503); }
            byte[] body = upstream.body();
            var response = ResponseEntity.status(upstream.statusCode()).header("Cache-Control", "no-store");
            for (var name : RESPONSE_HEADERS) upstream.headers().firstValue(name).ifPresent(value -> response.header(name,value));
            return response.body(body);
        } catch (java.util.concurrent.ExecutionException ex) {
            return denied(ex.getCause() instanceof BodyTooLarge ? 502 : 503);
        } catch (InterruptedException ex) {
            Thread.currentThread().interrupt(); return denied(503);
        }
    }

    private static final class BodyTooLarge extends java.io.IOException {}
    /** Cancel before accumulating more than one bounded page of upstream JSON. */
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

    private ResponseEntity<byte[]> denied(int status) {
        String code = status == 503 ? "ADMIN_ORDER_UPSTREAM_UNAVAILABLE" : status == 502 ? "ADMIN_ORDER_INVALID_RESPONSE" : "ADMIN_ORDER_PROXY_DENIED";
        byte[] body = ("{\"code\":\"" + code + "\",\"message\":\"Unable to read orders.\",\"requestId\":\""
                + java.util.UUID.randomUUID() + "\",\"errors\":[]}").getBytes(java.nio.charset.StandardCharsets.UTF_8);
        return ResponseEntity.status(status).header("Cache-Control", "no-store").header("Content-Type", "application/json").body(body);
    }
}
