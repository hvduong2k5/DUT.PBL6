package com.hvduong.catalog.grpc;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.catalog.grpc.v1.CatalogAdminServiceGrpc;
import com.omamx.proto.catalog.v1.CatalogServiceGrpc;
import com.sun.net.httpserver.HttpServer;
import io.grpc.*;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicReference;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class CatalogAccessTest {
    @Test
    @SuppressWarnings({"unchecked", "rawtypes"})
    void adminAuthorityComesFromVerifiedEmployeeAndBindsTheActor() throws Exception {
        String actor = UUID.randomUUID().toString();
        var persona = new AtomicReference<>("CUSTOMER");
        var status = new AtomicReference<>("ACTIVE");
        var server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/api/v1/users/me", exchange -> {
            assertEquals("Bearer fixture-credential", exchange.getRequestHeaders().getFirst("Authorization"));
            byte[] body = ("{\"success\":true,\"data\":{\"principalId\":\""+actor+
                    "\",\"principalType\":\""+persona.get()+"\",\"provisioningStatus\":\""+status.get()+
                    "\",\"roles\":[{\"roleCode\":\"ADMIN\"}]}}").getBytes(StandardCharsets.UTF_8);
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(200, body.length);
            try (var output = exchange.getResponseBody()) { output.write(body); }
        });
        server.start();
        try {
            var access = new CatalogAccess(new ObjectMapper());
            ReflectionTestUtils.setField(access, "identityUrl", "http://127.0.0.1:"+server.getAddress().getPort());
            var headers = new Metadata();
            headers.put(Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER), "Bearer fixture-credential");
            headers.put(Metadata.Key.of("x-actor-principal-id", Metadata.ASCII_STRING_MARSHALLER), UUID.randomUUID().toString());
            var called = new AtomicBoolean();
            ServerCallHandler handler = (call, metadata) -> {
                assertEquals(actor, CatalogAccess.actorId());
                called.set(true);
                return new ServerCall.Listener() {};
            };

            var customer = mock(ServerCall.class);
            when(customer.getMethodDescriptor()).thenReturn(CatalogAdminServiceGrpc.getCreateProductMethod());
            access.interceptCall(customer, headers, handler);
            verify(customer).close(eq(Status.PERMISSION_DENIED), any(Metadata.class));
            assertFalse(called.get(), "Customer with an ADMIN role must not reach admin code");

            persona.set("EMPLOYEE");
            status.set("SUSPENDED");
            var suspended = mock(ServerCall.class);
            when(suspended.getMethodDescriptor()).thenReturn(CatalogAdminServiceGrpc.getCreateProductMethod());
            access.interceptCall(suspended, headers, handler);
            verify(suspended).close(eq(Status.PERMISSION_DENIED), any(Metadata.class));
            assertFalse(called.get());

            status.set("ACTIVE");
            var employee = mock(ServerCall.class);
            when(employee.getMethodDescriptor()).thenReturn(CatalogAdminServiceGrpc.getCreateProductMethod());
            access.interceptCall(employee, headers, handler);
            assertTrue(called.get(), "Only an active verified Employee administrator is admitted");
            verify(employee, never()).close(any(), any());
        } finally { server.stop(0); }
    }

    @Test
    @SuppressWarnings({"unchecked", "rawtypes"})
    void checkoutSnapshotDoesNotAcceptAnUncredentialedServiceCall() {
        var access = new CatalogAccess(new ObjectMapper());
        ReflectionTestUtils.setField(access, "internalToken", "s".repeat(32));
        var call = mock(ServerCall.class);
        when(call.getMethodDescriptor()).thenReturn(CatalogServiceGrpc.getGetCheckoutSnapshotMethod());
        var handler = mock(ServerCallHandler.class);
        access.interceptCall(call, new Metadata(), handler);
        verify(call).close(eq(Status.UNAUTHENTICATED), any(Metadata.class));
        verifyNoInteractions(handler);
    }
}
