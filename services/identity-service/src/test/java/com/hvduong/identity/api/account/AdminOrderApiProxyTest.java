package com.hvduong.identity.api.account;

import com.hvduong.identity.api.account.impl.AdminOrderApiProxyController;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.Test;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.test.util.ReflectionTestUtils;
import java.net.InetSocketAddress;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicReference;
import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.*;

class AdminOrderApiProxyTest {
    private final UUID id = UUID.randomUUID();
    private final IdentityPrincipalMapper principals = mock(IdentityPrincipalMapper.class);
    private final AdminOrderApiProxyController controller = new AdminOrderApiProxyController(principals);
    private final Jwt jwt = Jwt.withTokenValue("server-token").header("alg", "RS256")
            .issuer("https://identity.test/realm").subject("staff").claim("user_id", id.toString()).build();
    private void principal(String type, String status) {
        var row = new IdentityPrincipalRow(); row.setPrincipalId(id); row.setPrincipalType(type); row.setProvisioningStatus(status);
        when(principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject())).thenReturn(row);
    }
    @Test void forwardsOnlyTheServerTokenAndAllowlistedResponseHeaders() throws Exception {
        principal("EMPLOYEE", "ACTIVE");
        var received = new AtomicReference<com.sun.net.httpserver.Headers>();
        var query = new AtomicReference<String>();
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v1/admin/orders", exchange -> {
            received.set(exchange.getRequestHeaders()); query.set(exchange.getRequestURI().getRawQuery());
            exchange.getResponseHeaders().add("Content-Type", "application/json");
            exchange.getResponseHeaders().add("Set-Cookie", "secret=forbidden");
            byte[] body = "{}".getBytes(); exchange.sendResponseHeaders(403, body.length); exchange.getResponseBody().write(body); exchange.close();
        }); server.start();
        try {
            ReflectionTestUtils.setField(controller, "gatewayUrl", "http://127.0.0.1:" + server.getAddress().getPort());
            var request = new MockHttpServletRequest("GET", "/api/v1/admin/order-api/orders"); request.setQueryString("limit=10");
            request.addHeader("Cookie", "OMAMA_SESSION=opaque"); request.addHeader("Authorization", "Bearer forged"); request.addHeader("X-User-ID", "forged"); request.addHeader("X-Internal-Token", "forged");
            var response = controller.forward(jwt, request);
            assertThat(response.getStatusCode().value()).isEqualTo(403); assertThat(query.get()).isEqualTo("limit=10");
            assertThat(received.get().getFirst("Authorization")).isEqualTo("Bearer server-token");
            assertThat(received.get().getFirst("Accept")).isEqualTo("application/vnd.omama.admin-order+json");
            for (String name : new String[]{"Cookie", "X-User-ID", "X-Internal-Token"}) assertThat(received.get().getFirst(name)).isNull();
            assertThat(response.getHeaders().getFirst("Set-Cookie")).isNull(); assertThat(response.getHeaders().getFirst("Cache-Control")).isEqualTo("no-store");
        } finally { server.stop(0); }
    }
    @Test void rejectsAnonymousCustomerInactivePrincipalAndUnlistedPaths() throws Exception {
        var request = new MockHttpServletRequest("GET", "/api/v1/admin/order-api/orders");
        assertThat(controller.forward(null, request).getStatusCode().value()).isEqualTo(401);
        principal("CUSTOMER", "ACTIVE"); assertThat(controller.forward(jwt, request).getStatusCode().value()).isEqualTo(403);
        principal("EMPLOYEE", "SUSPENDED"); assertThat(controller.forward(jwt, request).getStatusCode().value()).isEqualTo(403);
        principal("EMPLOYEE", "ACTIVE");
        for (String path : new String[]{"/../internal/permit", "/orders/" + id + "/hold", "/orders%2f..%2finternal", "//evil.test/orders"}) {
            assertThat(controller.forward(jwt, new MockHttpServletRequest("GET", "/api/v1/admin/order-api" + path)).getStatusCode().value()).isEqualTo(404);
        }
        assertThat(controller.forward(jwt, new MockHttpServletRequest("POST", "/api/v1/admin/order-api/orders")).getStatusCode().value()).isEqualTo(404);
    }
    @Test void capsTheUpstreamBodyBeforeRelayingIt() throws Exception {
        principal("EMPLOYEE", "ACTIVE");
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v1/admin/orders", exchange -> {
            byte[] body = new byte[(1 << 20) + 1]; exchange.sendResponseHeaders(200, body.length);
            try { exchange.getResponseBody().write(body); } finally { exchange.close(); }
        }); server.start();
        try {
            ReflectionTestUtils.setField(controller, "gatewayUrl", "http://127.0.0.1:" + server.getAddress().getPort());
            assertThat(controller.forward(jwt, new MockHttpServletRequest("GET", "/api/v1/admin/order-api/orders")).getStatusCode().value()).isEqualTo(502);
        } finally { server.stop(0); }
    }
}
