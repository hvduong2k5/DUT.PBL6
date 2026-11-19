package com.hvduong.identity.api.account;

import com.hvduong.identity.api.account.impl.AuthenticationController;
import com.hvduong.identity.application.account.AuthenticationService;
import com.hvduong.identity.application.dto.response.AuthUrlResponse;
import com.hvduong.identity.application.dto.response.BffCallbackResult;
import com.hvduong.identity.application.dto.response.SessionResponse;
import com.hvduong.identity.infrastructure.session.BffSession;
import jakarta.servlet.http.Cookie;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import java.time.Instant;
import java.util.UUID;

import static org.hamcrest.Matchers.containsString;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@ExtendWith(MockitoExtension.class)
class AuthenticationControllerTest {

    private MockMvc mockMvc;

    @Mock
    private AuthenticationService authenticationService;

    @InjectMocks
    private AuthenticationController authenticationController;

    @BeforeEach
    void setUp() {
        mockMvc = MockMvcBuilders.standaloneSetup(authenticationController)
                .setControllerAdvice(new com.hvduong.identity.common.handler.GlobalExceptionHandler())
                .build();
    }

    @Test
    void initiateLogin_ReturnsAuthUrl() throws Exception {
        AuthUrlResponse mockResponse = AuthUrlResponse.builder()
                .authorizationUrl("http://keycloak/auth?code_challenge=xyz")
                .state("state-123")
                .build();

        when(authenticationService.initiateLogin(eq("https://omama.vn/checkout"), eq("google")))
                .thenReturn(mockResponse);

        mockMvc.perform(get("/api/v1/public/auth/login")
                        .param("redirect_uri", "https://omama.vn/checkout")
                        .param("provider", "google")
                        .contentType(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.authorizationUrl").value("http://keycloak/auth?code_challenge=xyz"))
                .andExpect(jsonPath("$.data.state").value("state-123"));
    }

    @Test
    void initiateLogin_BrowserRequest_Returns302Redirect() throws Exception {
        AuthUrlResponse mockResponse = AuthUrlResponse.builder()
                .authorizationUrl("http://keycloak/auth?code_challenge=xyz")
                .state("state-123")
                .build();

        when(authenticationService.initiateLogin(eq("https://omama.vn/checkout"), isNull()))
                .thenReturn(mockResponse);

        mockMvc.perform(get("/api/v1/public/auth/login")
                        .param("redirect_uri", "https://omama.vn/checkout")
                        .accept(MediaType.TEXT_HTML))
                .andExpect(status().isFound())
                .andExpect(header().string("Location", "http://keycloak/auth?code_challenge=xyz"));
    }

    @Test
    void handleCallback_SetsCookieAndRedirects() throws Exception {
        UUID sessionId = UUID.randomUUID();
        BffSession session = BffSession.builder()
                .sessionId(sessionId)
                .build();

        BffCallbackResult result = BffCallbackResult.builder()
                .session(session)
                .redirectUri("https://omama.vn/dashboard")
                .build();

        when(authenticationService.handleCallback("auth-code-test", "state-test"))
                .thenReturn(result);

        mockMvc.perform(get("/api/v1/public/auth/callback")
                        .param("code", "auth-code-test")
                        .param("state", "state-test"))
                .andExpect(status().isFound())
                .andExpect(header().string(HttpHeaders.LOCATION, "https://omama.vn/dashboard"))
                .andExpect(header().string(HttpHeaders.SET_COOKIE, containsString("OMAMA_SESSION=" + sessionId)))
                .andExpect(header().string(HttpHeaders.SET_COOKIE, containsString("HttpOnly")))
                .andExpect(header().string(HttpHeaders.SET_COOKIE, containsString("SameSite=Lax")));
    }

    @Test
    void getSession_WithValidCookie_ReturnsSessionData() throws Exception {
        UUID sessionId = UUID.randomUUID();
        UUID principalId = UUID.randomUUID();

        SessionResponse sessionResponse = SessionResponse.builder()
                .sessionId(sessionId)
                .principalId(principalId)
                .email("user@omama.vn")
                .isAuthenticated(true)
                .expiresAt(Instant.now().plusSeconds(900))
                .build();

        when(authenticationService.getSessionInfo(eq(sessionId))).thenReturn(sessionResponse);

        mockMvc.perform(get("/api/v1/public/auth/session")
                        .cookie(new Cookie("OMAMA_SESSION", sessionId.toString())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.authenticated").value(true))
                .andExpect(jsonPath("$.data.email").value("user@omama.vn"));
    }

    @Test
    void refreshSession_Success() throws Exception {
        UUID sessionId = UUID.randomUUID();
        BffSession session = BffSession.builder()
                .sessionId(sessionId)
                .principalId(UUID.randomUUID())
                .email("refresh@omama.vn")
                .accessTokenExpiresAt(Instant.now().plusSeconds(900))
                .build();

        when(authenticationService.refreshSession(eq(sessionId))).thenReturn(session);

        mockMvc.perform(post("/api/v1/public/auth/tokens/refresh")
                        .cookie(new Cookie("OMAMA_SESSION", sessionId.toString())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.authenticated").value(true))
                .andExpect(jsonPath("$.data.email").value("refresh@omama.vn"));
    }

    @Test
    void logout_ClearsCookie() throws Exception {
        UUID sessionId = UUID.randomUUID();

        mockMvc.perform(post("/api/v1/public/auth/logout")
                        .cookie(new Cookie("OMAMA_SESSION", sessionId.toString())))
                .andExpect(status().isOk())
                .andExpect(header().string(HttpHeaders.SET_COOKIE, containsString("OMAMA_SESSION=")))
                .andExpect(header().string(HttpHeaders.SET_COOKIE, containsString("Max-Age=0")));

        verify(authenticationService).logout(eq(sessionId));
    }

    @Test
    void handleCallback_WhenIdpReturnsError_RedirectsToErrorLocation() throws Exception {
        when(authenticationService.resolveErrorRedirectUri(eq("state-err"), eq("access_denied"), eq("User cancelled")))
                .thenReturn("http://localhost:3000/login?error=access_denied&error_description=User+cancelled");

        mockMvc.perform(get("/api/v1/public/auth/callback")
                        .param("state", "state-err")
                        .param("error", "access_denied")
                        .param("error_description", "User cancelled"))
                .andExpect(status().isFound())
                .andExpect(header().string(HttpHeaders.LOCATION, "http://localhost:3000/login?error=access_denied&error_description=User+cancelled"));
    }

    @Test
    void handleCallback_WhenCodeOrStateMissing_ReturnsBadRequest() throws Exception {
        mockMvc.perform(get("/api/v1/public/auth/callback"))
                .andExpect(status().isBadRequest());
    }
}
