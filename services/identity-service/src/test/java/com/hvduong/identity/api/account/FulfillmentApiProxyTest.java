package com.hvduong.identity.api.account;

import com.hvduong.identity.api.account.impl.FulfillmentApiProxyController;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
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

class FulfillmentApiProxyTest {
    private final UUID id = UUID.randomUUID();
    private final IdentityPrincipalMapper principals = mock(IdentityPrincipalMapper.class);
    private final FulfillmentApiProxyController controller = new FulfillmentApiProxyController(principals);
    private final Jwt jwt = Jwt.withTokenValue("server-token").header("alg", "RS256")
            .issuer("https://identity.test/realm").subject("staff").claim("user_id", id.toString()).build();
    private void principal(String type, String status) {
        var row = new IdentityPrincipalRow(); row.setPrincipalId(id); row.setPrincipalType(type); row.setProvisioningStatus(status);
        when(principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject())).thenReturn(row);
    }
    @Test void forwardsServerTokenAndCommandHeadersButNotClientIdentity() throws Exception {
        principal("EMPLOYEE", "ACTIVE");
        var received = new AtomicReference<com.sun.net.httpserver.Headers>();
        var body = new AtomicReference<String>();
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v1/admin/fulfillment/tasks/" + id + "/start", exchange -> {
            received.set(exchange.getRequestHeaders()); body.set(new String(exchange.getRequestBody().readAllBytes()));
            exchange.getResponseHeaders().add("Content-Type", "application/json"); exchange.getResponseHeaders().add("ETag", "\"2\"");
            exchange.getResponseHeaders().add("Set-Cookie", "secret=forbidden");
            byte[] bytes = "{}".getBytes(); exchange.sendResponseHeaders(200, bytes.length); exchange.getResponseBody().write(bytes); exchange.close();
        });
        server.start();
        try {
            ReflectionTestUtils.setField(controller, "gatewayUrl", "http://127.0.0.1:" + server.getAddress().getPort());
            var request = new MockHttpServletRequest("POST", "/api/v1/admin/packing-api/tasks/" + id + "/start");
            request.setContent("{}".getBytes()); request.addHeader("If-Match", "\"1\""); request.addHeader("Idempotency-Key", "stable-command-123");
            request.addHeader("Cookie", "OMAMA_SESSION=opaque"); request.addHeader("X-Internal-Token", "forged"); request.addHeader("X-User-ID", "forged");
            var response = controller.forward(jwt, request);
            assertThat(response.getStatusCode().value()).isEqualTo(200); assertThat(body.get()).isEqualTo("{}");
            assertThat(received.get().getFirst("Authorization")).isEqualTo("Bearer server-token");
            assertThat(received.get().getFirst("If-Match")).isEqualTo("\"1\""); assertThat(received.get().getFirst("Idempotency-Key")).isEqualTo("stable-command-123");
            assertThat(received.get().getFirst("Cookie")).isNull(); assertThat(received.get().getFirst("X-Internal-Token")).isNull(); assertThat(received.get().getFirst("X-User-ID")).isNull();
            assertThat(response.getHeaders().getFirst("Set-Cookie")).isNull(); assertThat(response.getHeaders().getFirst("ETag")).isEqualTo("\"2\"");
        } finally { server.stop(0); }
    }
    @Test void deniesCustomerSuspendedAndAnonymousCallers() throws Exception {
        var request = new MockHttpServletRequest("GET", "/api/v1/admin/packing-api/tasks");
        assertThat(controller.forward(null, request).getStatusCode().value()).isEqualTo(401);
        principal("CUSTOMER", "ACTIVE"); assertThat(controller.forward(jwt, request).getStatusCode().value()).isEqualTo(403);
        principal("EMPLOYEE", "SUSPENDED"); assertThat(controller.forward(jwt, request).getStatusCode().value()).isEqualTo(403);
    }
    @Test void refusesTraversalUnsupportedMethodsAndOversizedJson() throws Exception {
        principal("EMPLOYEE", "ACTIVE");
        ReflectionTestUtils.setField(controller, "gatewayUrl", "http://127.0.0.1:1");
        for (String path : new String[]{"/../internal/permit", "/tasks/" + id + "/reopen", "/tasks%2f..%2finternal", "//evil.test/tasks"}) {
            assertThat(controller.forward(jwt, new MockHttpServletRequest("POST", "/api/v1/admin/packing-api" + path)).getStatusCode().value()).isEqualTo(404);
        }
        var request = new MockHttpServletRequest("POST", "/api/v1/admin/packing-api/tasks/" + id + "/start"); request.setContent(new byte[65537]);
        assertThat(controller.forward(jwt, request).getStatusCode().value()).isEqualTo(413);
    }
}
