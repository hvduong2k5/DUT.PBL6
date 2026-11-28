package com.hvduong.identity.grpc;

import static com.google.rpc.Status.newBuilder;

import com.google.protobuf.Any;
import com.google.rpc.ErrorInfo;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.omamx.proto.identity.v1.CurrentStaffGrantsRequest;
import com.omamx.proto.identity.v1.CurrentStaffGrantsResponse;
import com.omamx.proto.identity.v1.StaffAuthorizationGrpc;
import com.omamx.proto.identity.v1.StaffPermissionGrant;
import io.grpc.Context;
import io.grpc.Contexts;
import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.ServerInterceptor;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.protobuf.StatusProto;
import io.grpc.stub.StreamObserver;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.oauth2.jwt.BadJwtException;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.stereotype.Component;

/** Registered only on the dedicated TLS listener, never on the legacy plaintext server. */
@Component
public class TraceabilityStaffGrpcService extends StaffAuthorizationGrpc.StaffAuthorizationImplBase {
    static final Context.Key<String> STAFF_TOKEN = Context.key("traceability-staff-token");
    private final JwtDecoder decoder;
    private final IdentityPrincipalMapper principals;
    private final PermissionSyncService permissions;
    private final String audience;

    public TraceabilityStaffGrpcService(JwtDecoder decoder, IdentityPrincipalMapper principals,
            PermissionSyncService permissions, @Value("${IDENTITY_OIDC_AUDIENCE:omama-api}") String audience) {
        this.decoder = decoder;
        this.principals = principals;
        this.permissions = permissions;
        this.audience = audience;
    }

    static StatusRuntimeException authError(Status.Code code, String reason) {
        return StatusProto.toStatusRuntimeException(newBuilder()
                .setCode(code.value()).setMessage("Staff authorization failed")
                .addDetails(Any.pack(ErrorInfo.newBuilder().setDomain("identity.omama.internal")
                        .setReason(reason).build())).build());
    }

    private static String one(Metadata headers, String name) {
        var values = headers.getAll(Metadata.Key.of(name, Metadata.ASCII_STRING_MARSHALLER));
        if (values == null) return null;
        var iterator = values.iterator();
        if (!iterator.hasNext()) return null;
        String value = iterator.next();
        return iterator.hasNext() ? null : value;
    }

    static ServerInterceptor credentials(String secret) {
        return new ServerInterceptor() {
            @Override
            public <Q, R> ServerCall.Listener<Q> interceptCall(ServerCall<Q, R> call,
                    Metadata headers, ServerCallHandler<Q, R> next) {
                String service = one(headers, "authorization");
                if (secret == null || secret.length() < 32 || service == null
                        || !MessageDigest.isEqual(service.getBytes(StandardCharsets.UTF_8),
                                ("Bearer " + secret).getBytes(StandardCharsets.UTF_8))) {
                    var error = authError(Status.Code.UNAUTHENTICATED, "SERVICE_AUTHENTICATION_FAILED");
                    call.close(Status.fromThrowable(error), error.getTrailers());
                    return new ServerCall.Listener<>() {};
                }
                String staff = one(headers, "x-staff-authorization");
                if (staff == null || !staff.startsWith("Bearer ") || staff.length() <= 7 || staff.length() > 16384) {
                    var error = authError(Status.Code.UNAUTHENTICATED, "STAFF_UNAUTHENTICATED");
                    call.close(Status.fromThrowable(error), error.getTrailers());
                    return new ServerCall.Listener<>() {};
                }
                return Contexts.interceptCall(Context.current().withValue(STAFF_TOKEN, staff.substring(7)), call, headers, next);
            }
        };
    }

    @Override
    public void getCurrentStaffGrants(CurrentStaffGrantsRequest request,
            StreamObserver<CurrentStaffGrantsResponse> observer) {
        try {
            String raw = STAFF_TOKEN.get();
            if (raw == null) throw authError(Status.Code.UNAUTHENTICATED, "STAFF_UNAUTHENTICATED");
            var jwt = decoder.decode(raw);
            if (jwt.getIssuer() == null || jwt.getSubject() == null || jwt.getAudience() == null || !jwt.getAudience().contains(audience)) {
                throw authError(Status.Code.UNAUTHENTICATED, "STAFF_UNAUTHENTICATED");
            }
            var principal = principals.findByIssuerAndSubject(jwt.getIssuer().toString(), jwt.getSubject());
            if (principal == null || !"EMPLOYEE".equals(principal.getPrincipalType())
                    || !"ACTIVE".equals(principal.getProvisioningStatus())
                    || !principal.getPrincipalId().toString().equals(jwt.getClaimAsString("user_id"))) {
                throw authError(Status.Code.PERMISSION_DENIED, "STAFF_FORBIDDEN");
            }
            var result = CurrentStaffGrantsResponse.newBuilder()
                    .setPrincipalId(principal.getPrincipalId().toString()).setIdpSubject(principal.getIdpSubject())
                    .setPrincipalType(principal.getPrincipalType()).setProvisioningStatus(principal.getProvisioningStatus());
            for (var grant : permissions.resolveEffectivePermissionGrants(principal.getPrincipalId())) {
                if (!grant.permission().startsWith("traceability.")) continue;
                String scopeID = grant.scopeId() == null ? "" : grant.scopeId();
                if ("WAREHOUSE".equals(grant.scopeType()) && !UUID.fromString(scopeID).toString().equals(scopeID)) {
                    throw Status.UNAVAILABLE.withDescription("Invalid grant binding").asRuntimeException();
                }
                result.addPermissionGrants(StaffPermissionGrant.newBuilder().setPermissionCode(grant.permission())
                        .setScopeType(grant.scopeType()).setScopeId(scopeID));
                if (result.getPermissionGrantsCount() > 128) {
                    throw Status.RESOURCE_EXHAUSTED.withDescription("Grant response exceeds limit").asRuntimeException();
                }
            }
            var response = result.build();
            if (response.getSerializedSize() > 32768) {
                throw Status.RESOURCE_EXHAUSTED.withDescription("Grant response exceeds limit").asRuntimeException();
            }
            observer.onNext(response);
            observer.onCompleted();
        } catch (BadJwtException error) {
            observer.onError(authError(Status.Code.UNAUTHENTICATED, "STAFF_UNAUTHENTICATED"));
        } catch (StatusRuntimeException error) {
            observer.onError(error);
        } catch (Exception error) {
            observer.onError(Status.UNAVAILABLE.withDescription("Staff authorization unavailable").asRuntimeException());
        }
    }
}
