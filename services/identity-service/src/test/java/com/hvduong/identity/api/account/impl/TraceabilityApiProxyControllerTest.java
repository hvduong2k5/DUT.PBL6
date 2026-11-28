package com.hvduong.identity.api.account.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import java.util.UUID;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.atomic.AtomicReference;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.Test;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.security.oauth2.jwt.Jwt;

class TraceabilityApiProxyControllerTest {
    @Test void routesAreMethodBoundAndCannotReachInternalProviders() {
        String id = "11111111-1111-4111-8111-111111111111";
        assertEquals("/api/v2/admin/trace/records", TraceabilityApiProxyController.targetPath("GET", "/api/v1/admin/trace-api/records"));
        assertEquals("/api/v2/admin/trace/records/" + id + "/revisions/1/submit", TraceabilityApiProxyController.targetPath("POST", "/api/v1/admin/trace-api/records/" + id + "/revisions/1/submit"));
        assertEquals("/api/v1/admin/inventory/workbench/receipts", TraceabilityApiProxyController.targetPath("POST", "/api/v1/admin/inventory-api/workbench/receipts"));
        assertNull(TraceabilityApiProxyController.targetPath("DELETE", "/api/v1/admin/trace-api/records/" + id));
        assertNull(TraceabilityApiProxyController.targetPath("GET", "/api/v1/admin/trace-api/../internal/resolve"));
        assertNull(TraceabilityApiProxyController.targetPath("GET", "/api/v1/admin/inventory-api/internal"));
    }
    @Test void staffTransportsRequireKnownActiveEmployee() throws Exception {
        var mapper = mock(IdentityPrincipalMapper.class);
        var controller = new TraceabilityApiProxyController(mapper, "");
        var request = new MockHttpServletRequest("GET", "/api/v1/admin/trace-api/records");
        assertEquals(401, controller.forward(null, request).getStatusCode().value());
        UUID id = UUID.fromString("11111111-1111-4111-8111-111111111111");
        var jwt = Jwt.withTokenValue("staff").header("alg", "RS256").issuer("https://issuer.example")
                .subject("subject").claim("user_id", id.toString()).build();
        var row = new IdentityPrincipalRow(); row.setPrincipalId(id); row.setPrincipalType("CUSTOMER"); row.setProvisioningStatus("ACTIVE");
        when(mapper.findByIssuerAndSubject("https://issuer.example", "subject")).thenReturn(row);
        assertEquals(403, controller.forward(jwt, request).getStatusCode().value());
        row.setPrincipalType("EMPLOYEE"); row.setProvisioningStatus("SUSPENDED");
        assertEquals(403, controller.forward(jwt, request).getStatusCode().value());
        row.setProvisioningStatus("ACTIVE");
        assertEquals(503, controller.forward(jwt, request).getStatusCode().value());
    }
    @Test void forwardsVerifiedCredentialsAndBoundsResponseBytes() throws Exception {
        var observed = new AtomicReference<String>();
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v2/admin/trace/records", exchange -> {
            observed.set(exchange.getRequestHeaders().getFirst("Authorization") + ":"
                    + exchange.getRequestHeaders().getFirst("X-User-Id") + ":"
                    + exchange.getRequestHeaders().getFirst("Idempotency-Key"));
            byte[] body = (exchange.getRequestURI().getQuery() == null ? "{\"code\":\"CONFLICT\"}" : "x".repeat(1048577)).getBytes(StandardCharsets.UTF_8);
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.getResponseHeaders().set("Set-Cookie", "untrusted=value");
            exchange.sendResponseHeaders(409, body.length);
            try (var output = exchange.getResponseBody()) { output.write(body); }
        });
        server.start();
        try {
            var mapper = mock(IdentityPrincipalMapper.class);
            var controller = new TraceabilityApiProxyController(mapper, "http://127.0.0.1:" + server.getAddress().getPort());
            UUID id = UUID.randomUUID();
            var row = new IdentityPrincipalRow(); row.setPrincipalId(id); row.setPrincipalType("EMPLOYEE"); row.setProvisioningStatus("ACTIVE");
            when(mapper.findByIssuerAndSubject("https://issuer.example", "subject")).thenReturn(row);
            var jwt = Jwt.withTokenValue("verified-staff").header("alg", "RS256").issuer("https://issuer.example")
                    .subject("subject").claim("user_id", id.toString()).build();
            var request = new MockHttpServletRequest("POST", "/api/v1/admin/trace-api/records");
            request.setContent("{}".getBytes(StandardCharsets.UTF_8));
            request.addHeader("Authorization", "Bearer forged"); request.addHeader("X-User-Id", "forged"); request.addHeader("Idempotency-Key", "stable-command-key");
            var response = controller.forward(jwt, request);
            assertEquals(409, response.getStatusCode().value());
            assertEquals("Bearer verified-staff:null:stable-command-key", observed.get());
            assertEquals("no-store", response.getHeaders().getFirst("Cache-Control"));
            assertNull(response.getHeaders().getFirst("Set-Cookie"));
            request.setQueryString("oversized=true");
            assertEquals(502, controller.forward(jwt, request).getStatusCode().value());
            request.setQueryString(null); request.setContent(new byte[262145]);
            assertEquals(413, controller.forward(jwt, request).getStatusCode().value());
        } finally { server.stop(0); }
    }
}
