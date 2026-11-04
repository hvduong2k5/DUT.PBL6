package com.hvduong.identity.api.account;

import com.hvduong.identity.api.account.impl.CustomerApiProxyController;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.Test;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.test.util.ReflectionTestUtils;

import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicReference;
import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.*;

class CustomerApiProxyTest {
    private final UUID id = UUID.randomUUID();
    private final IdentityPrincipalMapper principals = mock(IdentityPrincipalMapper.class);
    private final CustomerApiProxyController controller = new CustomerApiProxyController(principals);
    private final Jwt jwt = Jwt.withTokenValue("server-only-jwt").header("alg", "RS256")
            .issuer("https://identity.example.test/realms/test").subject("test-subject")
            .claim("user_id", id.toString()).build();

    private void active() {
        var row = new IdentityPrincipalRow();
        row.setPrincipalId(id);
        row.setPrincipalType("CUSTOMER");
        row.setProvisioningStatus("ACTIVE");
        when(principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject())).thenReturn(row);
    }

    @Test
    void forwardsOnlyServerTokenAndAllowedHeadersToPinnedGateway() throws Exception {
        active();
        var received = new AtomicReference<com.sun.net.httpserver.Headers>();
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v1/cart", exchange -> {
            received.set(exchange.getRequestHeaders());
            exchange.getResponseHeaders().add("Content-Type", "application/json");
            exchange.getResponseHeaders().add("Set-Cookie", "upstream-secret=forbidden");
            byte[] body = "{\"revision\":2}".getBytes(StandardCharsets.UTF_8);
            exchange.sendResponseHeaders(200, body.length);
            exchange.getResponseBody().write(body);
            exchange.close();
        });
        server.start();
        try {
            ReflectionTestUtils.setField(controller, "gatewayUrl", "http://127.0.0.1:" + server.getAddress().getPort());
            var request = new MockHttpServletRequest("PUT", "/api/v1/customer-api/cart/items/PRIVATE-A");
            request.addHeader("If-Match", "\"1\"");
            request.addHeader("Cookie", "OMAMA_SESSION=opaque");
            request.addHeader("X-User-ID", "forged-owner");
            request.addHeader("X-Internal-Token", "forged-credential");
            var response = controller.forward(jwt, request);
            assertThat(response.getStatusCode().value()).isEqualTo(200);
            assertThat(received.get().getFirst("Authorization")).isEqualTo("Bearer server-only-jwt");
            assertThat(received.get().getFirst("If-Match")).isEqualTo("\"1\"");
            assertThat(received.get().getFirst("Cookie")).isNull();
            assertThat(received.get().getFirst("X-User-ID")).isNull();
            assertThat(received.get().getFirst("X-Internal-Token")).isNull();
            assertThat(response.getHeaders().getFirst("Set-Cookie")).isNull();
        } finally { server.stop(0); }
    }

    @Test
    void refusesAdminAndTraversalPathsWithoutSendingAnUpstreamRequest() throws Exception {
        active();
        for (String path : new String[]{"/admin/users", "/internal/secrets", "/profile/../admin/users", "//evil.test/cart", "/profile%2f..%2fadmin"}) {
            assertThat(controller.forward(jwt, new MockHttpServletRequest("GET", "/api/v1/customer-api" + path))
                    .getStatusCode().value()).isEqualTo(404);
        }
    }

    @Test
    void refusesUnmappedOrAnonymousPrincipals() throws Exception {
        var request = new MockHttpServletRequest("GET", "/api/v1/customer-api/cart");
        assertThat(controller.forward(null, request).getStatusCode().value()).isEqualTo(401);
        assertThat(controller.forward(jwt, request).getStatusCode().value()).isEqualTo(403);
    }
}
