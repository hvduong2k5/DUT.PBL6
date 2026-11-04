package com.hvduong.identity.security;

import com.hvduong.identity.application.account.AuthenticationService;
import com.hvduong.identity.infrastructure.session.BffSession;
import com.hvduong.identity.infrastructure.session.BffSessionStore;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpHeaders;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.mock.web.MockHttpServletResponse;

import java.io.IOException;
import java.time.Instant;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class BffSessionAuthFilterTest {

    @Mock
    private BffSessionStore bffSessionStore;

    @Mock
    private AuthenticationService authenticationService;

    @Mock
    private FilterChain filterChain;

    @InjectMocks
    private BffSessionAuthFilter bffSessionAuthFilter;

    private MockHttpServletRequest request;
    private MockHttpServletResponse response;

    @BeforeEach
    void setUp() {
        request = new MockHttpServletRequest();
        request.setMethod("GET");
        response = new MockHttpServletResponse();
    }

    @Test
    void cookieMutationWithoutTrustedOriginIsRejectedBeforeSessionLookup() throws Exception {
        request.setMethod("POST");
        request.setCookies(new Cookie("OMAMA_SESSION", UUID.randomUUID().toString()));
        org.springframework.test.util.ReflectionTestUtils.setField(bffSessionAuthFilter, "frontendUrl", "https://omama.vn");
        bffSessionAuthFilter.doFilterInternal(request, response, filterChain);
        assertThat(response.getStatus()).isEqualTo(403);
        verifyNoInteractions(filterChain, bffSessionStore);
    }

    @Test
    void callbackRequiresStateBoundToTheSameBrowser() throws Exception {
        request.setRequestURI("/api/v1/public/auth/callback");
        request.setParameter("state", "attacker-state");
        request.setCookies(new Cookie("OMAMA_AUTH_STATE", "victim-state"));
        bffSessionAuthFilter.doFilterInternal(request, response, filterChain);
        assertThat(response.getStatus()).isEqualTo(400);
        verifyNoInteractions(filterChain, bffSessionStore);
    }

    @Test
    void doFilterInternal_WithBearerHeader_PassesThroughWithoutCheckingCookie() throws ServletException, IOException {
        request.addHeader(HttpHeaders.AUTHORIZATION, "Bearer raw-jwt-token");

        bffSessionAuthFilter.doFilterInternal(request, response, filterChain);

        verify(filterChain).doFilter(request, response);
        verifyNoInteractions(bffSessionStore);
    }

    @Test
    void doFilterInternal_WithoutCookie_PassesThrough() throws ServletException, IOException {
        bffSessionAuthFilter.doFilterInternal(request, response, filterChain);

        verify(filterChain).doFilter(request, response);
        verifyNoInteractions(bffSessionStore);
    }

    @Test
    void doFilterInternal_WithValidSessionCookie_WrapsRequestWithBearerToken() throws ServletException, IOException {
        UUID sessionId = UUID.randomUUID();
        request.setCookies(new Cookie("OMAMA_SESSION", sessionId.toString()));

        BffSession session = BffSession.builder()
                .sessionId(sessionId)
                .accessToken("mock-bff-jwt-token")
                .accessTokenExpiresAt(Instant.now().plusSeconds(900))
                .refreshTokenExpiresAt(Instant.now().plusSeconds(2592000))
                .build();

        when(bffSessionStore.getSession(sessionId)).thenReturn(Optional.of(session));

        ArgumentCaptor<HttpServletRequest> captor = ArgumentCaptor.forClass(HttpServletRequest.class);

        bffSessionAuthFilter.doFilterInternal(request, response, filterChain);

        verify(filterChain).doFilter(captor.capture(), eq(response));
        HttpServletRequest wrapped = captor.getValue();

        assertThat(wrapped.getHeader(HttpHeaders.AUTHORIZATION)).isEqualTo("Bearer mock-bff-jwt-token");
    }

    @Test
    void doFilterInternal_WithExpiredAccessToken_AutoRefreshesBeforeWrapping() throws ServletException, IOException {
        UUID sessionId = UUID.randomUUID();
        request.setCookies(new Cookie("OMAMA_SESSION", sessionId.toString()));

        BffSession expiredSession = BffSession.builder()
                .sessionId(sessionId)
                .accessToken("expired-at")
                .accessTokenExpiresAt(Instant.now().minusSeconds(10))
                .refreshTokenExpiresAt(Instant.now().plusSeconds(2592000))
                .build();

        BffSession refreshedSession = BffSession.builder()
                .sessionId(sessionId)
                .accessToken("fresh-refreshed-at")
                .accessTokenExpiresAt(Instant.now().plusSeconds(900))
                .refreshTokenExpiresAt(Instant.now().plusSeconds(2592000))
                .build();

        when(bffSessionStore.getSession(sessionId)).thenReturn(Optional.of(expiredSession));
        when(authenticationService.refreshSession(sessionId)).thenReturn(refreshedSession);

        ArgumentCaptor<HttpServletRequest> captor = ArgumentCaptor.forClass(HttpServletRequest.class);

        bffSessionAuthFilter.doFilterInternal(request, response, filterChain);

        verify(authenticationService).refreshSession(sessionId);
        verify(filterChain).doFilter(captor.capture(), eq(response));
        assertThat(captor.getValue().getHeader(HttpHeaders.AUTHORIZATION)).isEqualTo("Bearer fresh-refreshed-at");
    }
}

