package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import jakarta.servlet.http.HttpServletRequest;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.Set;

/** Cookie-authenticated resource adapter. Tokens remain in the Identity BFF. */
@RestController
@RequiredArgsConstructor
public class CustomerApiProxyController {
    private final IdentityPrincipalMapper principals;
    @Value("${app.customer-api.gateway-url:}") private String gatewayUrl;
    private final HttpClient client = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(3)).followRedirects(HttpClient.Redirect.NEVER).build();
    private static final Set<String> REQUEST_HEADERS = Set.of(
            "Accept", "Content-Type", "If-Match", "Idempotency-Key", "X-Idempotency-Key");
    private static final Set<String> RESPONSE_HEADERS = Set.of(
            "content-type", "etag", "retry-after", "vary", "location", "link", "x-request-id");

    @RequestMapping("/api/v1/customer-api/**")
    public ResponseEntity<byte[]> forward(@AuthenticationPrincipal Jwt jwt, HttpServletRequest request) throws Exception {
        if (jwt == null) return ResponseEntity.status(401).build();
        var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
        if (principal == null || !"ACTIVE".equals(principal.getProvisioningStatus())
                || !principal.getPrincipalId().toString().equals(jwt.getClaimAsString("user_id"))) {
            return ResponseEntity.status(403).build();
        }
        String path = request.getRequestURI().substring("/api/v1/customer-api".length());
        // Never expose admin/internal/provider routes or accept an arbitrary target.
        if (!path.matches("/(?:users/me|profile(?:/addresses(?:/[0-9a-fA-F-]{36}(?:/default)?)?)?|customer/capabilities|locations/checkout|cart(?:/items(?:/[0-9a-fA-F-]{36})?)?|checkout(?:/quote)?|checkout-operations/[0-9a-fA-F-]{36}|orders(?:/[0-9a-fA-F-]{36}(?:/(?:cancel|tracking|payment-instructions|payment))?)?|payments(?:/[0-9a-fA-F-]{36})?)")
                || !Set.of("GET", "POST", "PUT", "PATCH", "DELETE").contains(request.getMethod())) {
            return ResponseEntity.notFound().build();
        }
        if (gatewayUrl == null || gatewayUrl.isBlank()) return ResponseEntity.status(503).build();
        String query = request.getQueryString();
        URI target = URI.create(gatewayUrl.replaceAll("/$", "") + "/api/v1" + path
                + (query == null ? "" : "?" + query));
        byte[] body = request.getInputStream().readNBytes((1 << 20) + 1);
        if (body.length > (1 << 20)) return ResponseEntity.status(413).build();
        var outbound = HttpRequest.newBuilder(target).timeout(Duration.ofSeconds(8))
                .header("Authorization", "Bearer " + jwt.getTokenValue())
                .method(request.getMethod(), body.length == 0
                        ? HttpRequest.BodyPublishers.noBody() : HttpRequest.BodyPublishers.ofByteArray(body));
        for (String name : REQUEST_HEADERS) {
            String value = request.getHeader(name);
            if (value != null) outbound.header(name, value);
        }
        try {
            var upstream = client.send(outbound.build(), HttpResponse.BodyHandlers.ofByteArray());
            var response = ResponseEntity.status(upstream.statusCode()).header("Cache-Control", "no-store");
            for (String name : RESPONSE_HEADERS) {
                upstream.headers().firstValue(name).ifPresent(value -> response.header(name, value));
            }
            return response.body(upstream.body());
        } catch (java.io.IOException ex) {
            return ResponseEntity.status(503).header("Cache-Control", "no-store").build();
        }
    }
}
