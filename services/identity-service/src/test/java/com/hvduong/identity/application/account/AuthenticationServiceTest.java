package com.hvduong.identity.application.account;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.application.account.impl.AuthenticationServiceImpl;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.dto.response.AuthUrlResponse;
import com.hvduong.identity.dto.response.BffCallbackResult;
import com.hvduong.identity.dto.response.SessionResponse;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.session.AuthStateStore;
import com.hvduong.identity.infrastructure.session.BffSession;
import com.hvduong.identity.infrastructure.session.BffSessionStore;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.Spy;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.test.util.ReflectionTestUtils;
import org.springframework.web.client.RestTemplate;

import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.Base64;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class AuthenticationServiceTest {

    @Mock
    private RestTemplate restTemplate;

    @Mock
    private IdentityPrincipalMapper principalMapper;

    @Mock
    private AuditService auditService;

    @Spy
    private ObjectMapper objectMapper = new ObjectMapper();

    @Spy
    private AuthStateStore authStateStore = new AuthStateStore();

    @Spy
    private BffSessionStore bffSessionStore = new BffSessionStore();

    @InjectMocks
    private AuthenticationServiceImpl authenticationService;

    @BeforeEach
    void setUp() {
        ReflectionTestUtils.setField(authenticationService, "serverUrl", "http://keycloak:8080");
        ReflectionTestUtils.setField(authenticationService, "publicUrl", "http://keycloak:8080");
        ReflectionTestUtils.setField(authenticationService, "realm", "omama");
        ReflectionTestUtils.setField(authenticationService, "publicClientId", "frontend-web");
        ReflectionTestUtils.setField(authenticationService, "redirectUri", "https://api.omama.vn/api/v1/public/auth/callback");
        ReflectionTestUtils.setField(authenticationService, "defaultFrontendUrl", "http://localhost:3000");
        ReflectionTestUtils.setField(authenticationService, "authScope", "openid");
    }

    @Test
    void initiateLogin_DefaultRedirect_GeneratesPkceAndAuthUrl() {
        AuthUrlResponse response = authenticationService.initiateLogin(null, null);

        assertThat(response).isNotNull();
        assertThat(response.getState()).isNotBlank();
        assertThat(response.getAuthorizationUrl())
                .contains("http://keycloak:8080/realms/omama/protocol/openid-connect/auth")
                .contains("response_type=code")
                .contains("client_id=frontend-web")
                .contains("code_challenge=")
                .contains("code_challenge_method=S256")
                .contains("scope=openid")
                .doesNotContain("kc_idp_hint");

        Optional<AuthStateStore.PendingAuthState> saved = authStateStore.consume(response.getState());
        assertThat(saved).isPresent();
        assertThat(saved.get().getRedirectUri()).isEqualTo("http://localhost:3000");
    }

    @Test
    void initiateLogin_WithProvider_AddsIdpHint() {
        AuthUrlResponse response = authenticationService.initiateLogin("https://omama.vn/cart", "google");

        assertThat(response).isNotNull();
        assertThat(response.getAuthorizationUrl())
                .contains("kc_idp_hint=google")
                .contains("client_id=frontend-web");

        Optional<AuthStateStore.PendingAuthState> saved = authStateStore.consume(response.getState());
        assertThat(saved).isPresent();
        assertThat(saved.get().getRedirectUri()).isEqualTo("https://omama.vn/cart");
    }

    @Test
    void handleCallback_InvalidState_ThrowsIllegalArgumentException() {
        assertThatThrownBy(() -> authenticationService.handleCallback("mock-code", "invalid-state"))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("không hợp lệ hoặc đã hết hạn");
    }

    @Test
    void handleCallback_ExistingActiveUser_Success() {
        String state = "test-state-1";
        String verifier = "test-verifier-1";
        authStateStore.save(state, verifier, "https://omama.vn/dashboard");

        String sub = "kc-sub-12345";
        String payloadJson = "{\"sub\":\"" + sub + "\",\"email\":\"user@omama.vn\"}";
        String dummyJwt = "header." + Base64.getUrlEncoder().withoutPadding().encodeToString(payloadJson.getBytes(StandardCharsets.UTF_8)) + ".sig";

        String keycloakResponse = "{"
                + "\"access_token\":\"" + dummyJwt + "\","
                + "\"refresh_token\":\"mock-refresh-token\","
                + "\"expires_in\":900,"
                + "\"refresh_expires_in\":2592000"
                + "}";

        when(restTemplate.postForEntity(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(new ResponseEntity<>(keycloakResponse, HttpStatus.OK));

        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(UUID.randomUUID());
        row.setProvisioningStatus("ACTIVE");
        when(principalMapper.findBySubject(sub)).thenReturn(row);

        BffCallbackResult result = authenticationService.handleCallback("auth-code-123", state);

        assertThat(result).isNotNull();
        assertThat(result.getRedirectUri()).isEqualTo("https://omama.vn/dashboard");
        assertThat(result.getSession()).isNotNull();
        assertThat(result.getSession().getPrincipalId()).isEqualTo(row.getPrincipalId());
        assertThat(result.getSession().getEmail()).isEqualTo("user@omama.vn");
        assertThat(result.getSession().getAccessToken()).isEqualTo(dummyJwt);

        verify(auditService).recordAudit(eq("USER"), eq(row.getPrincipalId()), isNull(), eq("LOGIN_SUCCESS"), any(), eq(sub), eq("SUCCESS"), isNull(), isNull(), isNull(), any());
    }

    @Test
    void handleCallback_SuspendedUser_ThrowsIllegalStateException() {
        String state = "test-state-suspended";
        String verifier = "test-verifier-suspended";
        authStateStore.save(state, verifier, "https://omama.vn");

        String sub = "kc-sub-suspended";
        String payloadJson = "{\"sub\":\"" + sub + "\",\"email\":\"suspended@omama.vn\"}";
        String dummyJwt = "header." + Base64.getUrlEncoder().withoutPadding().encodeToString(payloadJson.getBytes(StandardCharsets.UTF_8)) + ".sig";

        String keycloakResponse = "{\"access_token\":\"" + dummyJwt + "\",\"refresh_token\":\"rt\",\"expires_in\":900}";
        when(restTemplate.postForEntity(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(new ResponseEntity<>(keycloakResponse, HttpStatus.OK));

        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(UUID.randomUUID());
        row.setProvisioningStatus("SUSPENDED");
        when(principalMapper.findBySubject(sub)).thenReturn(row);

        assertThatThrownBy(() -> authenticationService.handleCallback("code-xyz", state))
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("tạm khóa");

        verify(auditService).recordAudit(eq("USER"), eq(row.getPrincipalId()), isNull(), eq("LOGIN_BLOCKED_SUSPENDED"), any(), any(), eq("FAILURE"), isNull(), isNull(), isNull(), any());
    }

    @Test
    void handleCallback_NewUser_TriggersJitProvisioning() {
        String state = "test-state-jit";
        String verifier = "test-verifier-jit";
        authStateStore.save(state, verifier, "https://omama.vn/profile");

        String sub = "kc-new-sub";
        String payloadJson = "{\"sub\":\"" + sub + "\",\"email\":\"newbie@omama.vn\"}";
        String dummyJwt = "header." + Base64.getUrlEncoder().withoutPadding().encodeToString(payloadJson.getBytes(StandardCharsets.UTF_8)) + ".sig";

        String keycloakResponse = "{"
                + "\"access_token\":\"" + dummyJwt + "\","
                + "\"refresh_token\":\"rt-jit\","
                + "\"expires_in\":900,"
                + "\"refresh_expires_in\":2592000"
                + "}";

        when(restTemplate.postForEntity(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(new ResponseEntity<>(keycloakResponse, HttpStatus.OK));
        when(principalMapper.findBySubject(sub)).thenReturn(null);

        BffCallbackResult result = authenticationService.handleCallback("code-jit", state);

        assertThat(result).isNotNull();
        assertThat(result.getRedirectUri()).isEqualTo("https://omama.vn/profile");
        assertThat(result.getSession().getEmail()).isEqualTo("newbie@omama.vn");

        verify(principalMapper).insert(argThat(principal -> 
                principal.getIdpSubject().equals(sub) && 
                "CUSTOMER".equals(principal.getPrincipalType()) &&
                "ACTIVE".equals(principal.getProvisioningStatus())
        ));
        verify(auditService).recordAudit(eq("USER"), any(), isNull(), eq("JIT_PROVISION_SUCCESS"), any(), eq(sub), eq("SUCCESS"), isNull(), isNull(), isNull(), any());
    }

    @Test
    void getSessionInfo_ValidSession_ReturnsAuthenticatedResponse() {
        BffSession session = bffSessionStore.createSession(
                "sub-1", UUID.randomUUID(), "test@omama.vn", "mock-token", "mock-rt", 900, 2592000
        );

        SessionResponse response = authenticationService.getSessionInfo(session.getSessionId());

        assertThat(response).isNotNull();
        assertThat(response.isAuthenticated()).isTrue();
        assertThat(response.getEmail()).isEqualTo("test@omama.vn");
        assertThat(response.getPrincipalId()).isEqualTo(session.getPrincipalId());
    }

    @Test
    void getSessionInfo_NonExistentSession_ReturnsUnauthenticated() {
        SessionResponse response = authenticationService.getSessionInfo(UUID.randomUUID());

        assertThat(response).isNotNull();
        assertThat(response.isAuthenticated()).isFalse();
    }

    @Test
    void refreshSession_Success() {
        BffSession session = bffSessionStore.createSession(
                "sub-2", UUID.randomUUID(), "refresh@omama.vn", "old-at", "old-rt", 900, 2592000
        );

        String keycloakResponse = "{"
                + "\"access_token\":\"new-at\","
                + "\"refresh_token\":\"new-rt\","
                + "\"expires_in\":900,"
                + "\"refresh_expires_in\":2592000"
                + "}";

        when(restTemplate.postForEntity(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(new ResponseEntity<>(keycloakResponse, HttpStatus.OK));

        BffSession refreshed = authenticationService.refreshSession(session.getSessionId());

        assertThat(refreshed).isNotNull();
        assertThat(refreshed.getAccessToken()).isEqualTo("new-at");
        assertThat(refreshed.getRefreshToken()).isEqualTo("new-rt");
    }

    @Test
    void logout_Success() {
        BffSession session = bffSessionStore.createSession(
                "sub-3", UUID.randomUUID(), "logout@omama.vn", "at", "rt", 900, 2592000
        );

        when(restTemplate.postForEntity(anyString(), any(HttpEntity.class), eq(String.class)))
                .thenReturn(new ResponseEntity<>("", HttpStatus.NO_CONTENT));

        authenticationService.logout(session.getSessionId());

        assertThat(bffSessionStore.getSession(session.getSessionId())).isEmpty();
        verify(auditService).recordAudit(eq("USER"), eq(session.getPrincipalId()), isNull(), eq("LOGOUT_SUCCESS"), any(), eq("sub-3"), eq("SUCCESS"), isNull(), isNull(), isNull(), any());
    }

    @Test
    void resolveErrorRedirectUri_WithSavedState_AppendsParamsToClientUri() {
        authStateStore.save("state-err-1", "verifier-1", "http://localhost:3000/login");

        String result = authenticationService.resolveErrorRedirectUri("state-err-1", "access_denied", "User cancelled");

        assertThat(result).isEqualTo("http://localhost:3000/login?error=access_denied&error_description=User+cancelled");
    }

    @Test
    void resolveErrorRedirectUri_WithMissingState_FallsBackToDefaultLogin() {
        String result = authenticationService.resolveErrorRedirectUri(null, "unauthorized_client", "Client not allowed");

        assertThat(result).isEqualTo("http://localhost:3000/login?error=unauthorized_client&error_description=Client+not+allowed");
    }
}
