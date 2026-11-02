package com.hvduong.identity.application.account.impl;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.application.account.AuthenticationService;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.dto.response.AuthUrlResponse;
import com.hvduong.identity.dto.response.BffCallbackResult;
import com.hvduong.identity.dto.response.SessionResponse;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.security.pkce.PkceUtil;
import com.hvduong.identity.infrastructure.session.AuthStateStore;
import com.hvduong.identity.infrastructure.session.BffSession;
import com.hvduong.identity.infrastructure.session.BffSessionStore;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.*;
import org.springframework.stereotype.Service;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.HttpStatusCodeException;
import org.springframework.web.client.RestTemplate;

import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.time.OffsetDateTime;
import java.util.Base64;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

@Slf4j
@Service
@RequiredArgsConstructor
public class AuthenticationServiceImpl implements AuthenticationService {

    private final RestTemplate restTemplate;
    private final IdentityPrincipalMapper principalMapper;
    private final AuditService auditService;
    private final ObjectMapper objectMapper;
    private final AuthStateStore authStateStore;
    private final BffSessionStore bffSessionStore;

    @Value("${keycloak.server-url}")
    private String serverUrl;

    @Value("${keycloak.public-url:${keycloak.server-url}}")
    private String publicUrl;

    @Value("${keycloak.realm}")
    private String realm;

    @Value("${keycloak.public-client-id:frontend-web}")
    private String publicClientId;

    @Value("${keycloak.redirect-uri:https://api.omama.vn/api/v1/public/auth/callback}")
    private String redirectUri;

    @Value("${keycloak.scope:openid}")
    private String authScope = "openid";

    @Value("${app.frontend-url:http://localhost:3000}")
    private String defaultFrontendUrl;

    @Override
    public AuthUrlResponse initiateLogin(String clientRedirectUri, String provider) {
        String finalRedirectUri = (clientRedirectUri != null && !clientRedirectUri.isBlank()) 
                ? clientRedirectUri 
                : getDefaultFrontendUrl();

        String codeVerifier = PkceUtil.generateCodeVerifier();
        String codeChallenge = PkceUtil.generateCodeChallenge(codeVerifier);
        String state = PkceUtil.generateState();

        authStateStore.save(state, codeVerifier, finalRedirectUri);

        String authEndpoint = buildAuthEndpoint();
        StringBuilder urlBuilder = new StringBuilder(authEndpoint);
        urlBuilder.append("?response_type=code")
                .append("&client_id=").append(publicClientId)
                .append("&redirect_uri=").append(URLEncoder.encode(redirectUri, StandardCharsets.UTF_8))
                .append("&scope=").append(URLEncoder.encode(authScope != null && !authScope.isBlank() ? authScope : "openid", StandardCharsets.UTF_8))
                .append("&state=").append(URLEncoder.encode(state, StandardCharsets.UTF_8))
                .append("&code_challenge=").append(URLEncoder.encode(codeChallenge, StandardCharsets.UTF_8))
                .append("&code_challenge_method=S256");

        if (provider != null && !provider.isBlank()) {
            urlBuilder.append("&kc_idp_hint=").append(URLEncoder.encode(provider.toLowerCase(), StandardCharsets.UTF_8));
        }

        return AuthUrlResponse.builder()
                .authorizationUrl(urlBuilder.toString())
                .state(state)
                .build();
    }

    @Override
    public BffCallbackResult handleCallback(String code, String state) {
        AuthStateStore.PendingAuthState pendingState = authStateStore.consume(state)
                .orElseThrow(() -> new IllegalArgumentException("Trạng thái xác thực (state) không hợp lệ hoặc đã hết hạn."));

        String tokenUrl = buildTokenEndpoint();
        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

        MultiValueMap<String, String> body = new LinkedMultiValueMap<>();
        body.add("grant_type", "authorization_code");
        body.add("client_id", publicClientId);
        body.add("code", code);
        body.add("redirect_uri", redirectUri);
        body.add("code_verifier", pendingState.getCodeVerifier());

        HttpEntity<MultiValueMap<String, String>> entity = new HttpEntity<>(body, headers);

        try {
            ResponseEntity<String> response = restTemplate.postForEntity(tokenUrl, entity, String.class);
            JsonNode root = objectMapper.readTree(response.getBody());

            String accessToken = root.path("access_token").asText();
            String refreshToken = root.path("refresh_token").asText();
            long expiresIn = root.path("expires_in").asLong(900);
            long refreshExpiresIn = root.path("refresh_expires_in").asLong(2592000);

            String subject = extractClaim(accessToken, "sub");
            String email = extractClaim(accessToken, "email");
            if (email == null || email.isBlank()) {
                email = extractClaim(accessToken, "preferred_username");
            }

            IdentityPrincipalRow principal = principalMapper.findBySubject(subject);
            if (principal != null) {
                if ("SUSPENDED".equalsIgnoreCase(principal.getProvisioningStatus())) {
                    log.warn("Blocked login attempt for suspended user subject={}", subject);
                    auditService.recordAudit("USER", principal.getPrincipalId(), null, "LOGIN_BLOCKED_SUSPENDED",
                            "AUTH", principal.getPrincipalId().toString(), "FAILURE", null, null, null,
                            Map.of("subject", subject, "reason", "ACCOUNT_SUSPENDED"));
                    throw new IllegalStateException("Tài khoản của bạn đã bị tạm khóa. Vui lòng liên hệ quản trị viên.");
                }

                auditService.recordAudit("USER", principal.getPrincipalId(), null, "LOGIN_SUCCESS",
                        "AUTH", subject, "SUCCESS", null, null, null,
                        Map.of("email", email != null ? email : ""));
            } else {
                // JIT Provisioning (Just-In-Time)
                UUID newPrincipalId = UUID.randomUUID();
                OffsetDateTime now = OffsetDateTime.now();
                principal = new IdentityPrincipalRow();
                principal.setPrincipalId(newPrincipalId);
                principal.setIdpIssuer(publicUrl + "/realms/" + realm);
                principal.setIdpSubject(subject);
                principal.setPrincipalType("CUSTOMER");
                principal.setProvisioningStatus("ACTIVE");
                principal.setCreatedAt(now);
                principal.setUpdatedAt(now);
                principal.setVersion(0L);

                principalMapper.insert(principal);
                log.info("JIT provisioned new user principalId={} subject={}", newPrincipalId, subject);

                auditService.recordAudit("USER", newPrincipalId, null, "JIT_PROVISION_SUCCESS",
                        "AUTH", subject, "SUCCESS", null, null, null,
                        Map.of("email", email != null ? email : ""));
            }

            BffSession session = bffSessionStore.createSession(
                    subject,
                    principal.getPrincipalId(),
                    email,
                    accessToken,
                    refreshToken,
                    expiresIn,
                    refreshExpiresIn
            );

            return BffCallbackResult.builder()
                    .session(session)
                    .redirectUri(pendingState.getRedirectUri())
                    .build();

        } catch (HttpStatusCodeException ex) {
            log.warn("Keycloak authorization code exchange failed: status={}, body={}", ex.getStatusCode(), ex.getResponseBodyAsString());
            auditService.recordAudit("USER", null, null, "LOGIN_FAILED", "AUTH", null, "FAILURE",
                    null, null, null, Map.of("statusCode", ex.getStatusCode().toString()));
            throw new IllegalArgumentException("Không thể xác thực mã đăng nhập từ Identity Provider.");
        } catch (IllegalStateException ex) {
            throw ex;
        } catch (Exception ex) {
            log.error("Unexpected error during authorization callback", ex);
            throw new RuntimeException("Lỗi hệ thống trong quá trình xử lý đăng nhập, vui lòng thử lại sau.", ex);
        }
    }

    @Override
    public String resolveErrorRedirectUri(String state, String error, String errorDescription) {
        String baseRedirectUri = getDefaultFrontendUrl() + "/login";
        if (state != null && !state.isBlank()) {
            Optional<AuthStateStore.PendingAuthState> pendingOpt = authStateStore.consume(state);
            if (pendingOpt.isPresent() && pendingOpt.get().getRedirectUri() != null && !pendingOpt.get().getRedirectUri().isBlank()) {
                baseRedirectUri = pendingOpt.get().getRedirectUri();
            }
        }
        StringBuilder sb = new StringBuilder(baseRedirectUri);
        sb.append(baseRedirectUri.contains("?") ? "&" : "?")
          .append("error=").append(URLEncoder.encode(error != null ? error : "auth_failed", StandardCharsets.UTF_8));
        if (errorDescription != null && !errorDescription.isBlank()) {
            sb.append("&error_description=").append(URLEncoder.encode(errorDescription, StandardCharsets.UTF_8));
        }
        return sb.toString();
    }

    @Override
    public SessionResponse getSessionInfo(UUID sessionId) {
        if (sessionId == null) {
            return SessionResponse.builder().isAuthenticated(false).build();
        }

        Optional<BffSession> sessionOpt = bffSessionStore.getSession(sessionId);
        if (sessionOpt.isEmpty()) {
            return SessionResponse.builder().isAuthenticated(false).build();
        }

        BffSession session = sessionOpt.get();
        if (session.isRefreshTokenExpired()) {
            bffSessionStore.removeSession(sessionId);
            return SessionResponse.builder().isAuthenticated(false).build();
        }

        if (session.isAccessTokenExpired()) {
            try {
                session = refreshSession(sessionId);
            } catch (Exception e) {
                log.warn("Auto-refresh session failed for sessionId={}: {}", sessionId, e.getMessage());
                return SessionResponse.builder().isAuthenticated(false).build();
            }
        }

        return SessionResponse.builder()
                .sessionId(session.getSessionId())
                .principalId(session.getPrincipalId())
                .email(session.getEmail())
                .isAuthenticated(true)
                .expiresAt(session.getAccessTokenExpiresAt())
                .accessToken(session.getAccessToken())
                .build();
    }

    @Override
    public BffSession refreshSession(UUID sessionId) {
        BffSession session = bffSessionStore.getSession(sessionId)
                .orElseThrow(() -> new IllegalArgumentException("Phiên đăng nhập không tồn tại hoặc đã hết hạn."));

        if (session.isRefreshTokenExpired()) {
            bffSessionStore.removeSession(sessionId);
            throw new IllegalArgumentException("Phiên làm việc đã hết hạn. Vui lòng đăng nhập lại.");
        }

        String tokenUrl = buildTokenEndpoint();
        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

        MultiValueMap<String, String> body = new LinkedMultiValueMap<>();
        body.add("grant_type", "refresh_token");
        body.add("client_id", publicClientId);
        body.add("refresh_token", session.getRefreshToken());

        HttpEntity<MultiValueMap<String, String>> entity = new HttpEntity<>(body, headers);

        try {
            ResponseEntity<String> response = restTemplate.postForEntity(tokenUrl, entity, String.class);
            JsonNode root = objectMapper.readTree(response.getBody());

            String newAccessToken = root.path("access_token").asText();
            String newRefreshToken = root.path("refresh_token").asText(session.getRefreshToken());
            long expiresIn = root.path("expires_in").asLong(900);
            long refreshExpiresIn = root.path("refresh_expires_in").asLong(2592000);

            bffSessionStore.updateTokens(sessionId, newAccessToken, newRefreshToken, expiresIn, refreshExpiresIn);
            return session;

        } catch (HttpStatusCodeException ex) {
            log.warn("Keycloak refresh token failed: status={}", ex.getStatusCode());
            bffSessionStore.removeSession(sessionId);
            throw new IllegalArgumentException("Không thể làm mới phiên đăng nhập. Vui lòng đăng nhập lại.");
        } catch (Exception ex) {
            log.error("Unexpected error during refresh token", ex);
            throw new RuntimeException("Không thể làm mới phiên đăng nhập.", ex);
        }
    }

    @Override
    public void logout(UUID sessionId) {
        if (sessionId == null) return;

        Optional<BffSession> sessionOpt = bffSessionStore.getSession(sessionId);
        if (sessionOpt.isPresent()) {
            BffSession session = sessionOpt.get();
            try {
                String logoutUrl = buildLogoutEndpoint();
                HttpHeaders headers = new HttpHeaders();
                headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

                MultiValueMap<String, String> body = new LinkedMultiValueMap<>();
                body.add("client_id", publicClientId);
                body.add("refresh_token", session.getRefreshToken());

                HttpEntity<MultiValueMap<String, String>> entity = new HttpEntity<>(body, headers);
                restTemplate.postForEntity(logoutUrl, entity, String.class);
                log.info("Revoked Keycloak tokens for sessionId={}", sessionId);
            } catch (Exception ex) {
                log.warn("Error revoking tokens on Keycloak for sessionId={}: {}", sessionId, ex.getMessage());
            }

            bffSessionStore.removeSession(sessionId);
            auditService.recordAudit("USER", session.getPrincipalId(), null, "LOGOUT_SUCCESS",
                    "AUTH", session.getIdpSubject(), "SUCCESS", null, null, null, Map.of("email", session.getEmail() != null ? session.getEmail() : ""));
        }
    }

    /**
     * URL cho TRÌNH DUYỆT mở (trang đăng nhập Keycloak) — ưu tiên dùng public-url, fallback về server-url.
     */
    private String getDefaultFrontendUrl() {
        if (defaultFrontendUrl == null || defaultFrontendUrl.isBlank()) {
            return "http://localhost:3000";
        }
        return defaultFrontendUrl.endsWith("/") 
                ? defaultFrontendUrl.substring(0, defaultFrontendUrl.length() - 1) 
                : defaultFrontendUrl;
    }

    private String buildAuthEndpoint() {
        String url = (publicUrl != null && !publicUrl.isBlank()) ? publicUrl : serverUrl;
        String baseUrl = url.endsWith("/") ? url.substring(0, url.length() - 1) : url;
        return baseUrl + "/realms/" + realm + "/protocol/openid-connect/auth";
    }

    /**
     * URL cho BACKEND gọi nội bộ (đổi code lấy token) — dùng server-url.
     */
    private String buildTokenEndpoint() {
        String baseUrl = serverUrl.endsWith("/") ? serverUrl.substring(0, serverUrl.length() - 1) : serverUrl;
        return baseUrl + "/realms/" + realm + "/protocol/openid-connect/token";
    }

    private String buildLogoutEndpoint() {
        String baseUrl = serverUrl.endsWith("/") ? serverUrl.substring(0, serverUrl.length() - 1) : serverUrl;
        return baseUrl + "/realms/" + realm + "/protocol/openid-connect/logout";
    }

    private String extractClaim(String jwtToken, String claimName) {
        try {
            String[] parts = jwtToken.split("\\.");
            if (parts.length < 2) return null;
            byte[] decodedBytes = Base64.getUrlDecoder().decode(parts[1]);
            JsonNode payloadNode = objectMapper.readTree(new String(decodedBytes, StandardCharsets.UTF_8));
            return payloadNode.path(claimName).asText(null);
        } catch (Exception e) {
            log.debug("Could not decode claim {} from JWT", claimName, e);
            return null;
        }
    }
}
