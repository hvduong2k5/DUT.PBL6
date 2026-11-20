package com.hvduong.identity.api.account;

import com.hvduong.identity.api.account.impl.AdminShippingApiProxyController;
import com.hvduong.identity.application.account.AuthenticationService;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.session.BffSession;
import com.hvduong.identity.infrastructure.session.BffSessionStore;
import com.hvduong.identity.security.BffSessionAuthFilter;
import com.hvduong.identity.security.PrincipalAccess;
import com.hvduong.identity.security.SecurityConfig;
import com.sun.net.httpserver.HttpServer;
import jakarta.servlet.http.Cookie;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.util.ReflectionTestUtils;
import org.springframework.test.web.servlet.MockMvc;

import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicReference;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;

// Real Spring session/security filters and HTTP relay; JWT decoder and session/DB are explicit fixtures.
@WebMvcTest(properties = {"app.shipping-api.enabled=true", "app.shipping-api.allow-insecure-http=true", "app.frontend-url=https://web.test"})
@ContextConfiguration(classes = {AdminShippingApiProxyController.class, SecurityConfig.class, PrincipalAccess.class, BffSessionAuthFilter.class})
class AdminShippingSessionSecurityTest {
    @Autowired private MockMvc mvc;
    @Autowired private AdminShippingApiProxyController controller;
    @MockBean private IdentityPrincipalMapper principals;
    @MockBean private BffSessionStore sessions;
    @MockBean private AuthenticationService authentication;
    @MockBean private JwtDecoder decoder;
    private final UUID sessionId = UUID.randomUUID();
    private final UUID userId = UUID.randomUUID();
    private final AtomicReference<String> bearer = new AtomicReference<>();
    private HttpServer server;
    private Jwt jwt(String grant) {
        return Jwt.withTokenValue("server-session-jwt").header("alg", "RS256").issuer("https://issuer.test/realm")
                .subject("employee").claim("principal_type", "EMPLOYEE").claim("user_id", userId.toString())
                .claim("aud", List.of("omama-api")).claim("permission_grants", List.of(grant)).build();
    }
    @BeforeEach void setup() throws Exception {
        var row = new IdentityPrincipalRow(); row.setPrincipalId(userId); row.setPrincipalType("EMPLOYEE"); row.setProvisioningStatus("ACTIVE");
        when(principals.findByIssuerAndSubject("https://issuer.test/realm", "employee")).thenReturn(row);
        when(sessions.getSession(sessionId)).thenReturn(Optional.of(BffSession.builder().sessionId(sessionId).accessToken("server-session-jwt")
                .accessTokenExpiresAt(Instant.now().plusSeconds(900)).refreshTokenExpiresAt(Instant.now().plusSeconds(3600)).build()));
        when(decoder.decode("server-session-jwt")).thenReturn(jwt("SHIPMENT_EXECUTE#GLOBAL"));
        server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v2/admin/shipping", exchange -> {
            bearer.set(exchange.getRequestHeaders().getFirst("Authorization"));
            assertThat(exchange.getRequestHeaders().getFirst("Cookie")).isNull();
            byte[] body = "{}".getBytes(StandardCharsets.UTF_8); exchange.getResponseHeaders().add("Content-Type", "application/json");
            exchange.sendResponseHeaders(200, body.length); exchange.getResponseBody().write(body); exchange.close();
        }); server.start();
        ReflectionTestUtils.setField(controller, "upstreamUrl", "http://127.0.0.1:" + server.getAddress().getPort());
    }
    @AfterEach void stop() { server.stop(0); }
    @Test void sessionCookieBecomesStoredJwtAndSupportsSameOriginMutation() throws Exception {
        mvc.perform(get("/api/v1/admin/shipping-api/workbench/shipments").cookie(new Cookie("OMAMA_SESSION", sessionId.toString()))).andExpect(status().isOk());
        assertThat(bearer.get()).isEqualTo("Bearer server-session-jwt");
        mvc.perform(post("/api/v1/admin/shipping-api/shipments/" + userId + "/accept").cookie(new Cookie("OMAMA_SESSION", sessionId.toString()))
                .header("Origin", "https://web.test").header("Idempotency-Key", "accept-session-key-0001").contentType("application/json").content("{\"expected_version\":1}")).andExpect(status().isOk());
    }
    @Test void missingSessionCrossOriginAndWrongScopedGrantNeverReachShipping() throws Exception {
        mvc.perform(get("/api/v1/admin/shipping-api/workbench/shipments")).andExpect(status().isUnauthorized()).andExpect(jsonPath("$.requestId").isNotEmpty());
        mvc.perform(post("/api/v1/admin/shipping-api/shipments/" + userId + "/accept").cookie(new Cookie("OMAMA_SESSION", sessionId.toString()))
                .header("Origin", "https://evil.test").contentType("application/json").content("{}")).andExpect(status().isForbidden());
        when(decoder.decode("server-session-jwt")).thenReturn(jwt("SHIPMENT_EXECUTE#WAREHOUSE#WH-1"));
        mvc.perform(get("/api/v1/admin/shipping-api/workbench/shipments").cookie(new Cookie("OMAMA_SESSION", sessionId.toString()))).andExpect(status().isForbidden());
        assertThat(bearer.get()).isNull();
    }
    @Test void refreshUsesFreshJwtAndRevokedExpiredSessionIsRejected() throws Exception {
        var expired = BffSession.builder().sessionId(sessionId).accessToken("expired-jwt").accessTokenExpiresAt(Instant.now().minusSeconds(60))
                .refreshTokenExpiresAt(Instant.now().plusSeconds(3600)).build();
        var refreshed = BffSession.builder().sessionId(sessionId).accessToken("server-session-jwt").accessTokenExpiresAt(Instant.now().plusSeconds(900))
                .refreshTokenExpiresAt(Instant.now().plusSeconds(3600)).build();
        when(sessions.getSession(sessionId)).thenReturn(Optional.of(expired));
        when(authentication.refreshSession(sessionId)).thenReturn(refreshed);
        mvc.perform(get("/api/v1/admin/shipping-api/workbench/shipments").cookie(new Cookie("OMAMA_SESSION", sessionId.toString()))).andExpect(status().isOk());
        when(authentication.refreshSession(sessionId)).thenThrow(new IllegalArgumentException("revoked fixture"));
        mvc.perform(get("/api/v1/admin/shipping-api/workbench/shipments").cookie(new Cookie("OMAMA_SESSION", sessionId.toString()))).andExpect(status().isUnauthorized());
    }
}
