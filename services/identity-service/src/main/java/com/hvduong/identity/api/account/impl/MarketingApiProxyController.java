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

/** Staff cookie-to-bearer boundary. Marketing owns campaign authorization. */
@RestController
@RequiredArgsConstructor
public class MarketingApiProxyController {
    private final IdentityPrincipalMapper principals;
    @Value("${app.customer-api.gateway-url:}") private String gatewayUrl;
    private final HttpClient client = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(3)).followRedirects(HttpClient.Redirect.NEVER).build();
    private static final String ID = "[0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}";
    private static final String PREFIX = "/api/v1/admin/marketing-api";
    private static final Set<String> REQUEST_HEADERS = Set.of("Accept", "Content-Type", "Idempotency-Key");
    private static final Set<String> RESPONSE_HEADERS = Set.of("content-type", "retry-after", "x-request-id");

    @RequestMapping(PREFIX + "/**")
    public ResponseEntity<byte[]> forward(@AuthenticationPrincipal Jwt jwt, HttpServletRequest request) throws Exception {
        if (jwt == null) return denied(401, "AUTHENTICATION_REQUIRED");
        var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
        if (principal == null || !PrincipalType.EMPLOYEE.matches(principal.getPrincipalType())
                || !ProvisioningStatus.ACTIVE.matches(principal.getProvisioningStatus())
                || !principal.getPrincipalId().toString().equals(jwt.getClaimAsString(IdentityConstants.CLAIM_USER_ID))) {
            return denied(403, "FORBIDDEN");
        }
        String path = request.getRequestURI().substring(PREFIX.length());
        boolean allowed = switch (request.getMethod()) {
            case "GET" -> path.matches("/campaigns(?:/" + ID + "(?:/revisions/" + ID + "(?:/executions(?:/" + ID + "/history)?)?)?)?");
            case "POST" -> path.matches("/campaigns(?:/" + ID
                    + "/revisions(?:/" + ID + "/(?:submit|withdraw|approve|reject|request-changes|executions(?:/" + ID + "/actions)?))?)?");
            case "PATCH" -> path.matches("/campaigns/" + ID + "/revisions/" + ID);
            default -> false;
        };
        if (!allowed) return denied(404, "RESOURCE_NOT_FOUND");
        if (gatewayUrl == null || gatewayUrl.isBlank()) return denied(503, "SERVICE_UNAVAILABLE");
        byte[] body = request.getInputStream().readNBytes(524289);
        if (body.length > 524288) return denied(400, "BAD_REQUEST");
        String query = request.getQueryString();
        URI target = URI.create(gatewayUrl.replaceAll("/$", "") + "/api/v1/admin/marketing" + path
                + (query == null ? "" : "?" + query));
        var outbound = HttpRequest.newBuilder(target).timeout(Duration.ofSeconds(12))
                .header("Authorization", "Bearer " + jwt.getTokenValue())
                .method(request.getMethod(), body.length == 0 ? HttpRequest.BodyPublishers.noBody() : HttpRequest.BodyPublishers.ofByteArray(body));
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
        } catch (InterruptedException ex) {
            Thread.currentThread().interrupt();
            return denied(503, "SERVICE_UNAVAILABLE");
        } catch (java.io.IOException ex) {
            return denied(503, "SERVICE_UNAVAILABLE");
        }
    }

    private ResponseEntity<byte[]> denied(int status, String code) {
        String body = """
                {"code":"%s","message":"Unable to process the Marketing request.","requestId":"%s"}
                """.formatted(code, java.util.UUID.randomUUID());
        return ResponseEntity.status(status).header("Cache-Control", "no-store")
                .contentType(org.springframework.http.MediaType.APPLICATION_JSON)
                .body(body.getBytes(java.nio.charset.StandardCharsets.UTF_8));
    }
}
