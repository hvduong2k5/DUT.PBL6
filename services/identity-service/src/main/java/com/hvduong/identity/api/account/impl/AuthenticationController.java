package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.api.account.AuthenticationApi;
import com.hvduong.identity.application.account.AuthenticationService;
import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.response.AuthUrlResponse;
import com.hvduong.identity.dto.response.BffCallbackResult;
import com.hvduong.identity.dto.response.SessionResponse;
import com.hvduong.identity.infrastructure.session.BffSession;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseCookie;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;

import java.net.URI;
import java.time.Duration;
import java.util.UUID;

@Slf4j
@RestController
@RequiredArgsConstructor
public class AuthenticationController implements AuthenticationApi {

    private final AuthenticationService authenticationService;

    @Value("${app.session.cookie-secure:false}")
    private boolean cookieSecure = false;

    @Value("${app.session.cookie-domain:}")
    private String cookieDomain = "";

    @Value("${app.session.cookie-same-site:Lax}")
    private String sameSite = "Lax";

    @Override
    public ResponseEntity<ApiResponse<AuthUrlResponse>> initiateLogin(String redirectUri, String provider) {
        AuthUrlResponse response = authenticationService.initiateLogin(redirectUri, provider);
        return ResponseEntity.ok(ApiResponse.success("Khởi tạo đăng nhập thành công", response));
    }

    @Override
    public ResponseEntity<Void> handleCallback(String code, String state, String error, String errorDescription) {
        if (error != null) {
            log.warn("OAuth2 callback received error: {} - {}", error, errorDescription);
            throw new IllegalArgumentException("Đăng nhập thất bại từ phía IdP: " + (errorDescription != null ? errorDescription : error));
        }

        BffCallbackResult result = authenticationService.handleCallback(code, state);

        ResponseCookie cookie = buildSessionCookie(result.getSession().getSessionId().toString(), Duration.ofDays(30));

        return ResponseEntity.status(HttpStatus.FOUND)
                .location(URI.create(result.getRedirectUri()))
                .header(HttpHeaders.SET_COOKIE, cookie.toString())
                .build();
    }

    @Override
    public ResponseEntity<ApiResponse<SessionResponse>> getSession(String sessionCookie) {
        UUID sessionId = parseSessionId(sessionCookie);
        SessionResponse response = authenticationService.getSessionInfo(sessionId);
        return ResponseEntity.ok(ApiResponse.success("Lấy thông tin phiên làm việc thành công", response));
    }

    @Override
    public ResponseEntity<ApiResponse<SessionResponse>> refreshSession(String sessionCookie) {
        UUID sessionId = parseSessionId(sessionCookie);
        if (sessionId == null) {
            throw new IllegalArgumentException("Không tìm thấy phiên làm việc trong Cookie.");
        }

        BffSession session = authenticationService.refreshSession(sessionId);
        SessionResponse response = SessionResponse.builder()
                .sessionId(session.getSessionId())
                .principalId(session.getPrincipalId())
                .email(session.getEmail())
                .isAuthenticated(true)
                .expiresAt(session.getAccessTokenExpiresAt())
                .build();

        return ResponseEntity.ok(ApiResponse.success("Làm mới phiên làm việc thành công", response));
    }

    @Override
    public ResponseEntity<ApiResponse<Void>> logout(String sessionCookie) {
        UUID sessionId = parseSessionId(sessionCookie);
        if (sessionId != null) {
            authenticationService.logout(sessionId);
        }

        ResponseCookie expiredCookie = buildSessionCookie("", Duration.ZERO);

        return ResponseEntity.ok()
                .header(HttpHeaders.SET_COOKIE, expiredCookie.toString())
                .body(ApiResponse.success("Đăng xuất thành công", null));
    }

    private ResponseCookie buildSessionCookie(String value, Duration maxAge) {
        ResponseCookie.ResponseCookieBuilder builder = ResponseCookie.from("OMAMA_SESSION", value)
                .httpOnly(true)
                .secure(cookieSecure)
                .path("/")
                .sameSite(sameSite)
                .maxAge(maxAge);

        if (cookieDomain != null && !cookieDomain.isBlank()) {
            builder.domain(cookieDomain);
        }
        return builder.build();
    }

    private UUID parseSessionId(String sessionCookie) {
        if (sessionCookie == null || sessionCookie.isBlank()) {
            return null;
        }
        try {
            return UUID.fromString(sessionCookie.trim());
        } catch (IllegalArgumentException e) {
            log.debug("Invalid session cookie UUID format: {}", sessionCookie);
            return null;
        }
    }
}
