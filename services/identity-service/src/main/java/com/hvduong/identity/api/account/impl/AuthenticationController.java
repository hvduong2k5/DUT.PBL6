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
    public ResponseEntity<?> initiateLogin(
            String redirectUri, 
            String provider, 
            String format, 
            String acceptHeader, 
            String contentType
    ) {
        AuthUrlResponse response = authenticationService.initiateLogin(redirectUri, provider);
        String bindingCookie = buildCookie("OMAMA_AUTH_STATE", response.getState(), Duration.ofMinutes(10)).toString();

        boolean wantsJson = "json".equalsIgnoreCase(format)
                || (contentType != null && contentType.contains(org.springframework.http.MediaType.APPLICATION_JSON_VALUE))
                || (acceptHeader != null && acceptHeader.contains(org.springframework.http.MediaType.APPLICATION_JSON_VALUE) 
                    && !acceptHeader.contains(org.springframework.http.MediaType.TEXT_HTML_VALUE));

        if (wantsJson) {
            return ResponseEntity.ok().header(HttpHeaders.SET_COOKIE, bindingCookie)
                    .body(ApiResponse.success("Khởi tạo đăng nhập thành công", response));
        }

        return ResponseEntity.status(HttpStatus.FOUND)
                .header(HttpHeaders.SET_COOKIE, bindingCookie)
                .location(URI.create(response.getAuthorizationUrl()))
                .build();
    }

    @Override
    public ResponseEntity<?> handleCallback(String code, String state, String error, String errorDescription, String format, String acceptHeader, String contentType) {
        if (error != null) {
            log.warn("OAuth2 callback received error from IdP: {} - {}", error, errorDescription);
            String errorRedirect = authenticationService.resolveErrorRedirectUri(state, error, errorDescription);
            return ResponseEntity.status(HttpStatus.FOUND)
                    .location(URI.create(errorRedirect))
                    .build();
        }

        if (code == null || code.isBlank() || state == null || state.isBlank()) {
            log.warn("OAuth2 callback missing required code or state");
            throw new IllegalArgumentException("Thiếu tham số 'code' hoặc 'state' trong yêu cầu xác thực.");
        }

        BffCallbackResult result = authenticationService.handleCallback(code, state);

        ResponseCookie cookie = buildSessionCookie(result.getSession().getSessionId().toString(), Duration.ofDays(30));

        if (result.getSession().isNativeClient()) {
            var session=result.getSession();
            var tokens=com.hvduong.identity.dto.response.AuthTokenResponse.builder().tokenType("Bearer")
                .accessToken(session.getAccessToken()).refreshToken(session.getRefreshToken())
                .sessionId(session.getSessionId()).principalId(session.getPrincipalId()).email(session.getEmail())
                .expiresAt(session.getAccessTokenExpiresAt()).expiresIn(Math.max(0,java.time.Duration.between(java.time.Instant.now(),session.getAccessTokenExpiresAt()).getSeconds())).build();
            return ResponseEntity.ok().header(HttpHeaders.SET_COOKIE,cookie.toString(),buildCookie("OMAMA_AUTH_STATE","",Duration.ZERO).toString()).body(ApiResponse.success("Đăng nhập thành công",tokens));
        }
        return ResponseEntity.status(HttpStatus.FOUND)
                .location(URI.create(result.getRedirectUri()))
                .header(HttpHeaders.SET_COOKIE, cookie.toString(), buildCookie("OMAMA_AUTH_STATE", "", Duration.ZERO).toString())
                .build();
    }

    public ResponseEntity<?> handleCallback(String code,String state,String error,String description) {return handleCallback(code,state,error,description,null,null,null);}

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
                .sessionId(session.isNativeClient()?session.getSessionId():session.getPublicId())
                .accessToken(session.isNativeClient()?session.getAccessToken():null)
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
        return buildCookie("OMAMA_SESSION", value, maxAge);
    }

    private ResponseCookie buildCookie(String name, String value, Duration maxAge) {
        ResponseCookie.ResponseCookieBuilder builder = ResponseCookie.from(name, value)
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
            log.debug("Invalid session cookie UUID format");
            return null;
        }
    }
}
