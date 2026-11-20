package com.hvduong.identity.api.account;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.api.account.impl.AdminShippingApiProxyController;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.sun.net.httpserver.Headers;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.Test;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.test.util.ReflectionTestUtils;

import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicReference;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class AdminShippingApiProxyTest {
    private final UUID id = UUID.randomUUID();
    private final IdentityPrincipalMapper principals = mock(IdentityPrincipalMapper.class);
    private final AdminShippingApiProxyController controller = new AdminShippingApiProxyController(principals, new ObjectMapper());
    private Jwt token(String... grants) {
        return Jwt.withTokenValue("server-staff-token").header("alg", "RS256").issuer("https://issuer.test/realm")
                .subject("staff").claim("principal_type", "EMPLOYEE").claim("user_id", id.toString())
                .claim("permission_grants", List.of(grants)).build();
    }
    private void employee(String type, String status) {
        var row = new IdentityPrincipalRow(); row.setPrincipalId(id); row.setPrincipalType(type); row.setProvisioningStatus(status);
        when(principals.findByIssuerAndSubject("https://issuer.test/realm", "staff")).thenReturn(row);
        ReflectionTestUtils.setField(controller, "enabled", true);
        ReflectionTestUtils.setField(controller, "allowHttp", true);
    }
    private MockHttpServletRequest request(String method, String path) {
        var request = new MockHttpServletRequest(method, "/api/v1/admin/shipping-api" + path);
        request.addHeader("Cookie", "OMAMA_SESSION=private"); request.addHeader("Authorization", "Bearer forged");
        request.addHeader("X-Internal-Token", "forged"); request.addHeader("X-Actor-ID", "forged");
        request.setContentType("application/json"); request.addHeader("Idempotency-Key", "shipping-action-key-0001");
        request.setContent("{\"expected_version\":1}".getBytes(StandardCharsets.UTF_8));
        return request;
    }
    @Test void forwardsOnlyServerBearerAndActionKeyWithoutCookiesOrOverrides() throws Exception {
        employee("EMPLOYEE", "ACTIVE");
        var received = new AtomicReference<Headers>(); var body = new AtomicReference<String>();
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v2/admin/shipping/shipments/" + id + "/accept", exchange -> {
            received.set(exchange.getRequestHeaders()); body.set(new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8));
            exchange.getResponseHeaders().add("Content-Type", "application/json"); exchange.getResponseHeaders().add("Set-Cookie", "forbidden=secret");
            byte[] payload = "{}".getBytes(StandardCharsets.UTF_8); exchange.sendResponseHeaders(200, payload.length); exchange.getResponseBody().write(payload); exchange.close();
        }); server.start();
        try {
            ReflectionTestUtils.setField(controller, "upstreamUrl", "http://127.0.0.1:" + server.getAddress().getPort());
            var response = controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), request("POST", "/shipments/" + id + "/accept"));
            assertThat(response.getStatusCode().value()).isEqualTo(200);
            assertThat(received.get().getFirst("Authorization")).isEqualTo("Bearer server-staff-token");
            assertThat(received.get().getFirst("Idempotency-Key")).isEqualTo("shipping-action-key-0001");
            assertThat(body.get()).isEqualTo("{\"expected_version\":1}");
            for (String name : List.of("Cookie", "X-Internal-Token", "X-Actor-ID")) assertThat(received.get().getFirst(name)).isNull();
            assertThat(response.getHeaders().getFirst("Set-Cookie")).isNull();
            assertThat(response.getHeaders().getFirst("Cache-Control")).isEqualTo("private, no-store");
        } finally { server.stop(0); }
    }
    @Test void rejectsWrongPrincipalsScopesPathsQueriesAndBodyOverrides() throws Exception {
        var read = request("GET", "/workbench/shipments");
        assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), read).getStatusCode().value()).isEqualTo(404);
        employee("CUSTOMER", "ACTIVE"); assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), read).getStatusCode().value()).isEqualTo(403);
        employee("EMPLOYEE", "SUSPENDED"); assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), read).getStatusCode().value()).isEqualTo(403);
        employee("EMPLOYEE", "ACTIVE");
        assertThat(controller.forward(null, read).getStatusCode().value()).isEqualTo(401);
        for (String grant : List.of("SHIPMENT_VIEW#GLOBAL", "SHIPMENT_EXECUTE#WAREHOUSE#WH-1")) assertThat(controller.forward(token(grant), read).getStatusCode().value()).isEqualTo(403);
        for (String path : List.of("/../internal/shipments", "/shipments/" + id + "/cancel", "/workbench/shipments%2f..", "//evil.test")) assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), request("GET", path)).getStatusCode().value()).isEqualTo(404);
        read.addParameter("driver_id", id.toString()); assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), read).getStatusCode().value()).isEqualTo(400);
        var post = request("POST", "/shipments/" + id + "/accept"); post.setContent("{\"actor_id\":\"spoof\"}".getBytes(StandardCharsets.UTF_8));
        assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), post).getStatusCode().value()).isEqualTo(400);
        post.setContent(new byte[4097]); assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), post).getStatusCode().value()).isEqualTo(413);
        var assign = request("POST", "/shipments/" + id + "/assignment");
        assertThat(controller.forward(token("SHIPMENT_ASSIGN#GLOBAL"), assign).getStatusCode().value()).isEqualTo(403);
    }
    @Test void rejectsRedirectOversizedUpstreamAndUnsafeConfiguration() throws Exception {
        employee("EMPLOYEE", "ACTIVE"); var read = request("GET", "/workbench/shipments");
        for (String url : List.of("", "https://user:password@example.test", "https://example.test/private", "https://example.test?url=evil")) {
            ReflectionTestUtils.setField(controller, "upstreamUrl", url);
            assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), read).getStatusCode().value()).isEqualTo(503);
        }
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v2/admin/shipping/workbench/shipments", exchange -> {
            exchange.getResponseHeaders().add("Location", "https://evil.test"); exchange.sendResponseHeaders(302, -1); exchange.close();
        }); server.start();
        try {
            ReflectionTestUtils.setField(controller, "upstreamUrl", "http://127.0.0.1:" + server.getAddress().getPort());
            assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), read).getStatusCode().value()).isEqualTo(502);
            server.removeContext("/api/v2/admin/shipping/workbench/shipments");
            server.createContext("/api/v2/admin/shipping/workbench/shipments", exchange -> {
                byte[] payload = new byte[(1 << 20) + 1]; exchange.sendResponseHeaders(200, payload.length);
                try { exchange.getResponseBody().write(payload); } finally { exchange.close(); }
            });
            assertThat(controller.forward(token("SHIPMENT_EXECUTE#GLOBAL"), read).getStatusCode().value()).isEqualTo(502);
        } finally { server.stop(0); }
    }
}
