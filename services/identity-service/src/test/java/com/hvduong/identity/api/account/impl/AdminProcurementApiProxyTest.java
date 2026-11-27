package com.hvduong.identity.api.account.impl;

import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.sun.net.httpserver.Headers;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.BeforeEach;
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

class AdminProcurementApiProxyTest {
    private static final String PREFIX = "/api/v1/admin/procurement-api/";
    private final UUID id = UUID.randomUUID();
    private final IdentityPrincipalMapper principals = mock(IdentityPrincipalMapper.class);
    private final AdminProcurementApiProxyController controller = new AdminProcurementApiProxyController(principals);
    private final IdentityPrincipalRow principal = new IdentityPrincipalRow();

    private Jwt token(String... grants) {
        return Jwt.withTokenValue("server-token").header("alg", "RS256").issuer("https://identity.test/realm")
                .subject("staff").claim("user_id", id.toString()).claim("principal_type", "EMPLOYEE")
                .claim("permission_grants", List.of(grants)).build();
    }
    private MockHttpServletRequest request(String method, String path) { return new MockHttpServletRequest(method, PREFIX + path); }
    @BeforeEach void setup() {
        principal.setPrincipalId(id); principal.setPrincipalType("EMPLOYEE"); principal.setProvisioningStatus("ACTIVE");
        when(principals.findByIssuerAndSubject("https://identity.test/realm", "staff")).thenReturn(principal);
        ReflectionTestUtils.setField(controller, "enabled", true);
    }

    @Test void failsClosedAndNeverUsesRoleOrFlatPermissions() throws Exception {
        var request = request("GET", "suppliers");
        ReflectionTestUtils.setField(controller, "enabled", false);
        assertThat(controller.forward(token("PROCUREMENT_VIEW#GLOBAL"), request).getStatusCode().value()).isEqualTo(404);
        ReflectionTestUtils.setField(controller, "enabled", true);
        assertThat(controller.forward(null, request).getStatusCode().value()).isEqualTo(401);
        for (String grant : new String[]{"PROCUREMENT_VIEW", "PROCUREMENT_MANAGE#GLOBAL", "PROCUREMENT_VIEW#WAREHOUSE#WH-1", "PROCUREMENT_VIEW#GLOBAL#extra", "ADMIN#GLOBAL"})
            assertThat(controller.forward(token(grant), request).getStatusCode().value()).isEqualTo(403);
        principal.setProvisioningStatus("SUSPENDED");
        assertThat(controller.forward(token("PROCUREMENT_VIEW#GLOBAL"), request).getStatusCode().value()).isEqualTo(403);
        principal.setProvisioningStatus("ACTIVE"); principal.setPrincipalId(UUID.randomUUID());
        assertThat(controller.forward(token("PROCUREMENT_VIEW#GLOBAL"), request).getStatusCode().value()).isEqualTo(403);
    }

    @Test void allowlistExcludesPlansAndUnapprovedPathsAndMethods() {
        String id = this.id.toString();
        String[][] allowed = {{"GET", "suppliers"}, {"POST", "suppliers"}, {"GET", "suppliers/"+id}, {"PUT", "suppliers/"+id}, {"PATCH", "suppliers/"+id+"/status"},
                {"GET", "materials"}, {"POST", "materials"}, {"GET", "materials/"+id}, {"PUT", "materials/"+id}, {"PATCH", "materials/"+id+"/status"}, {"PUT", "materials/"+id+"/preferred-supplier"},
                {"GET", "materials/"+id+"/offers"}, {"GET", "suppliers/"+id+"/offers"}, {"POST", "suppliers/"+id+"/offers"}, {"PATCH", "suppliers/"+id+"/offers/"+id+"/status"},
                {"GET", "purchase-orders"}, {"POST", "purchase-orders"}, {"GET", "purchase-orders/"+id}, {"PUT", "purchase-orders/"+id}, {"GET", "purchase-orders/"+id+"/revisions"},
                {"POST", "purchase-orders/"+id+"/revisions"}, {"POST", "purchase-orders/"+id+"/submit"}, {"POST", "purchase-orders/"+id+"/approve"}, {"POST", "purchase-orders/"+id+"/reject"},
                {"POST", "purchase-orders/"+id+"/send"}, {"POST", "purchase-orders/"+id+"/supplier-response"}, {"POST", "purchase-orders/"+id+"/cancel"}, {"POST", "purchase-orders/"+id+"/close"},
                {"GET", "suppliers/"+id+"/contracts"}, {"POST", "suppliers/"+id+"/contracts"}, {"GET", "contracts/"+id}, {"POST", "contracts/"+id+"/activate"}, {"POST", "contracts/"+id+"/terminate"},
                {"POST", "goods-receipts"}, {"GET", "goods-receipts/"+id}, {"POST", "goods-receipts/"+id+"/inspect"}, {"POST", "goods-receipts/"+id+"/variances/"+id+"/resolve"}};
        assertThat(allowed.length).isEqualTo(37);
        for (String[] route : allowed) assertThat(AdminProcurementApiProxyController.allowedRoute(route[0], route[1])).as(String.join(" ", route)).isTrue();
        for (String path : new String[]{"supply-plans", "supply-plans/"+id, "suppliers/%2e%2e", "../internal", "suppliers/00000000-0000-0000-0000-000000000000", "suppliers/"+id+"/anything"})
            for (String method : new String[]{"GET", "POST", "PUT", "PATCH"}) assertThat(AdminProcurementApiProxyController.allowedRoute(method, path)).isFalse();
        for (String[] route : allowed) assertThat(AdminProcurementApiProxyController.allowedRoute("DELETE", route[1])).isFalse();
    }

    @Test void forwardsOnlyVerifiedJwtAndCommandKeyWithoutSecretsOrRedirects() throws Exception {
        var headers = new AtomicReference<Headers>(); var receivedBody = new AtomicReference<String>();
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v1/admin/procurement/suppliers", exchange -> {
            headers.set(exchange.getRequestHeaders()); receivedBody.set(new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8));
            exchange.getResponseHeaders().add("Content-Type", "application/json"); exchange.getResponseHeaders().add("Set-Cookie", "secret=forbidden");
            byte[] bytes = "{\"id\":\"created\"}".getBytes(StandardCharsets.UTF_8); exchange.sendResponseHeaders(201, bytes.length); exchange.getResponseBody().write(bytes); exchange.close();
        }); server.start();
        try {
            ReflectionTestUtils.setField(controller, "upstreamUrl", "http://127.0.0.1:"+server.getAddress().getPort());
            var request = request("POST", "suppliers"); request.setContentType("application/json"); request.setContent("{\"name\":\"test\"}".getBytes(StandardCharsets.UTF_8));
            request.addHeader("Cookie", "OMAMA_SESSION=opaque"); request.addHeader("Authorization", "Bearer forged"); request.addHeader("X-Internal-Token", "forged"); request.addHeader("X-User-Role", "ADMIN"); request.addHeader("Idempotency-Key", "procurement-command-0001");
            assertThat(controller.forward(token("PROCUREMENT_MANAGE#GLOBAL"), request).getStatusCode().value()).isEqualTo(503);
            ReflectionTestUtils.setField(controller, "allowInsecureHttp", true);
            var response = controller.forward(token("PROCUREMENT_MANAGE#GLOBAL"), request);
            assertThat(response.getStatusCode().value()).isEqualTo(201);
            assertThat(headers.get().getFirst("Authorization")).isEqualTo("Bearer server-token"); assertThat(headers.get().getFirst("Idempotency-Key")).isEqualTo("procurement-command-0001");
            for (String name : new String[]{"Cookie", "X-Internal-Token", "X-User-Role"}) assertThat(headers.get().getFirst(name)).isNull();
            assertThat(receivedBody.get()).isEqualTo("{\"name\":\"test\"}"); assertThat(response.getHeaders().getFirst("Set-Cookie")).isNull();
            assertThat(response.getHeaders().getFirst("Cache-Control")).isEqualTo("no-store");
        } finally { server.stop(0); }
    }

    @Test void separatesContractApprovalAndReceiptPermissions() throws Exception {
        for (String[] route : new String[][]{{"GET", "contracts/"+id, "PROCUREMENT_VIEW#GLOBAL"}, {"POST", "purchase-orders/"+id+"/approve", "PROCUREMENT_MANAGE#GLOBAL"}, {"POST", "goods-receipts/"+id+"/inspect", "PROCUREMENT_VIEW#GLOBAL"}})
            assertThat(controller.forward(token(route[2]), request(route[0], route[1])).getStatusCode().value()).isEqualTo(403);
        // A warehouse grant may reach Procurement, which decides exact receipt scope.
        assertThat(controller.forward(token("PROCUREMENT_RECEIVE#WAREHOUSE#WH-1"), request("GET", "goods-receipts/"+id)).getStatusCode().value()).isEqualTo(503);
        for (String query : new String[]{"target=http://evil", "limit=1&limit=2", "limit=101", "offset=-1", "%ZZ=1"}) {
            var request = request("GET", "suppliers"); request.setQueryString(query);
            assertThat(controller.forward(token("PROCUREMENT_VIEW#GLOBAL"), request).getStatusCode().value()).isEqualTo(400);
        }
    }

    @Test void cancelsOversizedResponsesAndRejectsRedirects() throws Exception {
        var status = new AtomicReference<>(200);
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v1/admin/procurement/suppliers", exchange -> {
            exchange.getResponseHeaders().add("Content-Type", "application/json"); exchange.getResponseHeaders().add("Location", "https://evil.test");
            byte[] bytes = status.get() == 200 ? new byte[(1 << 20) + 1] : "{}".getBytes(StandardCharsets.UTF_8);
            exchange.sendResponseHeaders(status.get(), bytes.length);
            try { exchange.getResponseBody().write(bytes); } finally { exchange.close(); }
        }); server.start();
        try {
            ReflectionTestUtils.setField(controller, "upstreamUrl", "http://127.0.0.1:"+server.getAddress().getPort()); ReflectionTestUtils.setField(controller, "allowInsecureHttp", true);
            assertThat(controller.forward(token("PROCUREMENT_VIEW#GLOBAL"), request("GET", "suppliers")).getStatusCode().value()).isEqualTo(502);
            status.set(302); assertThat(controller.forward(token("PROCUREMENT_VIEW#GLOBAL"), request("GET", "suppliers")).getStatusCode().value()).isEqualTo(502);
        } finally { server.stop(0); }
    }
}
