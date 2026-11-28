package com.hvduong.identity.grpc;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import com.google.rpc.ErrorInfo;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.omamx.proto.identity.v1.CurrentStaffGrantsRequest;
import com.omamx.proto.identity.v1.CurrentStaffGrantsResponse;
import io.grpc.Context;
import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.Status;
import io.grpc.protobuf.StatusProto;
import io.grpc.stub.StreamObserver;
import java.util.Collections;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicBoolean;
import org.junit.jupiter.api.Test;
import org.springframework.security.oauth2.jwt.BadJwtException;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtDecoder;

class TraceabilityStaffGrpcServiceTest {
    private final JwtDecoder decoder = mock(JwtDecoder.class);
    private final IdentityPrincipalMapper principals = mock(IdentityPrincipalMapper.class);
    private final PermissionSyncService permissions = mock(PermissionSyncService.class);
    private final UUID id = UUID.fromString("11111111-1111-4111-8111-111111111111");
    private final String warehouse = "22222222-2222-4222-8222-222222222222";

    private IdentityPrincipalRow employee() {
        var row = new IdentityPrincipalRow();
        row.setPrincipalId(id); row.setIdpSubject("staff-subject"); row.setPrincipalType("EMPLOYEE"); row.setProvisioningStatus("ACTIVE");
        return row;
    }
    private Capture invoke(IdentityPrincipalRow row, List<PermissionSyncService.PermissionGrant> grants) {
        var jwt = Jwt.withTokenValue("staff").header("alg", "RS256").issuer("https://issuer.example")
                .subject("staff-subject").audience(List.of("omama-api")).claim("user_id", id.toString()).build();
        when(decoder.decode("staff")).thenReturn(jwt);
        when(principals.findByIssuerAndSubject("https://issuer.example", "staff-subject")).thenReturn(row);
        when(permissions.resolveEffectivePermissionGrants(id)).thenReturn(grants);
        return call();
    }
    private Capture call() {
        var capture = new Capture();
        Context.current().withValue(TraceabilityStaffGrpcService.STAFF_TOKEN, "staff").run(() ->
                new TraceabilityStaffGrpcService(decoder, principals, permissions, "omama-api")
                        .getCurrentStaffGrants(CurrentStaffGrantsRequest.getDefaultInstance(), capture));
        return capture;
    }
    @Test void currentGrantsPreservePermissionScopeTuples() {
        var result = invoke(employee(), List.of(
                new PermissionSyncService.PermissionGrant("traceability.read_internal", "WAREHOUSE", warehouse),
                new PermissionSyncService.PermissionGrant("traceability.create", "GLOBAL", null),
                new PermissionSyncService.PermissionGrant("CATALOG_VIEW", "GLOBAL", null)));
        assertNull(result.error); assertTrue(result.completed);
        assertEquals(2, result.value.getPermissionGrantsCount());
        assertEquals(warehouse, result.value.getPermissionGrants(0).getScopeId());
        assertEquals("", result.value.getPermissionGrants(1).getScopeId());
        assertEquals(id.toString(), result.value.getPrincipalId());
    }
    @Test void customersAndInactiveEmployeesAreForbidden() throws Exception {
        for (String type : List.of("CUSTOMER", "INACTIVE")) {
            var row = employee();
            if (type.equals("CUSTOMER")) row.setPrincipalType(type); else row.setProvisioningStatus(type);
            var result = invoke(row, List.of());
            assertEquals(Status.Code.PERMISSION_DENIED, Status.fromThrowable(result.error).getCode());
            assertEquals("STAFF_FORBIDDEN", StatusProto.fromThrowable(result.error).getDetails(0).unpack(ErrorInfo.class).getReason());
        }
    }
    @Test void invalidJwtIsDistinctFromDependencyFailure() throws Exception {
        doThrow(new BadJwtException("synthetic invalid token")).when(decoder).decode("staff");
        var result = call();
        assertEquals("STAFF_UNAUTHENTICATED", StatusProto.fromThrowable(result.error).getDetails(0).unpack(ErrorInfo.class).getReason());
        doThrow(new IllegalStateException("synthetic provider outage")).when(decoder).decode("staff");
        assertEquals(Status.Code.UNAVAILABLE, Status.fromThrowable(call().error).getCode());
    }
    @Test void excessiveGrantsAreRejectedWithoutTruncation() {
        var result = invoke(employee(), Collections.nCopies(129,
                new PermissionSyncService.PermissionGrant("traceability.create", "WAREHOUSE", warehouse)));
        assertNull(result.value);
        assertEquals(Status.Code.RESOURCE_EXHAUSTED, Status.fromThrowable(result.error).getCode());
    }
    @Test @SuppressWarnings("unchecked") void interceptorRequiresExactlyOneOfEachCredential() {
        String token = "synthetic-service-credential-at-least-32-bytes";
        for (int mode = 0; mode < 3; mode++) {
            ServerCall<CurrentStaffGrantsRequest, CurrentStaffGrantsResponse> call = mock(ServerCall.class);
            Metadata headers = new Metadata();
            headers.put(Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER), "Bearer " + token);
            headers.put(Metadata.Key.of("x-staff-authorization", Metadata.ASCII_STRING_MARSHALLER), "Bearer staff");
            if (mode == 1) headers.put(Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER), "Bearer " + token);
            if (mode == 2) headers.put(Metadata.Key.of("x-staff-authorization", Metadata.ASCII_STRING_MARSHALLER), "Bearer other");
            var reached = new AtomicBoolean();
            TraceabilityStaffGrpcService.credentials(token).interceptCall(call, headers, (request, metadata) -> {
                reached.set(true); assertEquals("staff", TraceabilityStaffGrpcService.STAFF_TOKEN.get());
                return new ServerCall.Listener<>() {};
            });
            assertEquals(mode == 0, reached.get());
            if (mode != 0) verify(call).close(argThat(status -> status.getCode() == Status.Code.UNAUTHENTICATED), any());
        }
    }
    private static final class Capture implements StreamObserver<CurrentStaffGrantsResponse> {
        CurrentStaffGrantsResponse value; Throwable error; boolean completed;
        @Override public void onNext(CurrentStaffGrantsResponse response) { value = response; }
        @Override public void onError(Throwable response) { error = response; }
        @Override public void onCompleted() { completed = true; }
    }
}
