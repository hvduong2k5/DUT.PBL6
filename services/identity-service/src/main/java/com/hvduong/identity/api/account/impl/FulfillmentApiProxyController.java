package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.common.constant.IdentityConstants;
import com.hvduong.identity.common.enums.PrincipalType;
import com.hvduong.identity.common.enums.ProvisioningStatus;
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

/** Staff cookie-to-bearer boundary. Fulfillment remains the warehouse authority. */
@RestController
@RequiredArgsConstructor
public class FulfillmentApiProxyController {
    private final IdentityPrincipalMapper principals;
    @Value("${app.customer-api.gateway-url:}") private String gatewayUrl;
    private final HttpClient client = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(3)).followRedirects(HttpClient.Redirect.NEVER).build();
    private static final String ID = "[0-9a-f-]{36}";

    @RequestMapping("/api/v1/admin/packing-api/**")
    public ResponseEntity<byte[]> forward(@AuthenticationPrincipal Jwt jwt, HttpServletRequest request) throws Exception {
        if (jwt == null) return ResponseEntity.status(401).build();
        var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
        if (principal == null || !PrincipalType.EMPLOYEE.matches(principal.getPrincipalType())
                || !ProvisioningStatus.ACTIVE.matches(principal.getProvisioningStatus())
                || !principal.getPrincipalId().toString().equals(jwt.getClaimAsString(IdentityConstants.CLAIM_USER_ID))) {
            return ResponseEntity.status(403).build();
        }
        String path = request.getRequestURI().substring("/api/v1/admin/packing-api".length());
        boolean allowed = switch (request.getMethod()) {
            case "GET" -> path.matches("/tasks(?:/" + ID + "(?:/history)?)?|/media/" + ID);
            case "POST" -> path.matches("/tasks/" + ID + "/(?:assignments|start|unassign|seal|media-sessions)|/media/" + ID + "/complete");
            case "PUT" -> path.matches("/tasks/" + ID + "/(?:picks/" + ID + "|checklist/[A-Za-z0-9_-]{1,80})");
            default -> false;
        };
        if (!allowed) return ResponseEntity.notFound().build();
        if (gatewayUrl == null || gatewayUrl.isBlank()) return ResponseEntity.status(503).build();
        byte[] body = request.getInputStream().readNBytes(65537);
        if (body.length > 65536) return ResponseEntity.status(413).build();
        String query = request.getQueryString();
        URI target = URI.create(gatewayUrl.replaceAll("/$", "") + "/api/v1/admin/fulfillment" + path
                + (query == null ? "" : "?" + query));
        var outbound = HttpRequest.newBuilder(target).timeout(Duration.ofSeconds(12))
                .header("Authorization", "Bearer " + jwt.getTokenValue())
                .method(request.getMethod(), body.length == 0 ? HttpRequest.BodyPublishers.noBody() : HttpRequest.BodyPublishers.ofByteArray(body));
        for (String name : Set.of("Accept", "Content-Type", "If-Match", "Idempotency-Key")) {
            String value = request.getHeader(name);
            if (value != null) outbound.header(name, value);
        }
        try {
            var upstream = client.send(outbound.build(), HttpResponse.BodyHandlers.ofByteArray());
            var response = ResponseEntity.status(upstream.statusCode()).header("Cache-Control", "no-store");
            for (String name : Set.of("content-type", "etag", "retry-after", "x-request-id")) {
                upstream.headers().firstValue(name).ifPresent(value -> response.header(name, value));
            }
            return response.body(upstream.body());
        } catch (java.io.IOException ex) {
            return ResponseEntity.status(503).header("Cache-Control", "no-store").build();
        }
    }
}
