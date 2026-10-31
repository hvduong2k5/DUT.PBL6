package com.hvduong.identity.grpc;

import com.hvduong.identity.application.authorization.SpecializedPermissionService;
import com.omamx.proto.identity.v1.GetJwksPublicKeyRequest;
import com.omamx.proto.identity.v1.GetJwksPublicKeyResponse;
import com.omamx.proto.identity.v1.IdentityServiceGrpc;
import com.omamx.proto.identity.v1.SpecializedPermissionRequest;
import com.omamx.proto.identity.v1.SpecializedPermissionResponse;
import io.grpc.stub.StreamObserver;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.server.service.GrpcService;

@Slf4j
@GrpcService
@RequiredArgsConstructor
public class IdentityGrpcService extends IdentityServiceGrpc.IdentityServiceImplBase {

    private final SpecializedPermissionService specializedPermissionService;

    @Override
    public void checkSpecializedPermission(
            SpecializedPermissionRequest request,
            StreamObserver<SpecializedPermissionResponse> responseObserver
    ) {
        log.info("gRPC call CheckSpecializedPermission: userId={}, action={}",
                request.getUserId(), request.getAction());

        try {
            String correlationId = request.hasAuditContext() ? request.getAuditContext().getTraceparent() : null;
            String clientIp = request.hasAuditContext() ? request.getAuditContext().getClientIp() : null;

            SpecializedPermissionService.EvaluationResult eval = specializedPermissionService.evaluatePermission(
                    request.getUserId(),
                    request.getAction(),
                    request.getContextAttributesMap(),
                    correlationId,
                    clientIp
            );

            SpecializedPermissionResponse.Builder responseBuilder = SpecializedPermissionResponse.newBuilder()
                    .setIsAllowed(eval.isAllowed())
                    .setDecisionReason(eval.getReason() != null ? eval.getReason() : "");

            if (eval.getRequiredApprovalLevel() != null && !eval.getRequiredApprovalLevel().isBlank()) {
                responseBuilder.setRequiredApprovalLevel(eval.getRequiredApprovalLevel());
            }

            responseObserver.onNext(responseBuilder.build());
            responseObserver.onCompleted();

        } catch (Exception ex) {
            log.error("Unhandled error in CheckSpecializedPermission", ex);
            // Fail-closed on server error
            SpecializedPermissionResponse response = SpecializedPermissionResponse.newBuilder()
                    .setIsAllowed(false)
                    .setDecisionReason("Server evaluation error: " + ex.getMessage())
                    .build();
            responseObserver.onNext(response);
            responseObserver.onCompleted();
        }
    }

    @Override
    public void getJwksPublicKey(
            GetJwksPublicKeyRequest request,
            StreamObserver<GetJwksPublicKeyResponse> responseObserver
    ) {
        log.info("gRPC call GetJwksPublicKey: keyId={}", request.getKeyId());

        // Note: As per api-contract-decisions.md, this RPC is deprecated in favor of Gateway fetching
        // JWKS directly from Keycloak OIDC Discovery endpoint. We provide standard active key response.
        GetJwksPublicKeyResponse response = GetJwksPublicKeyResponse.newBuilder()
                .setKeyId(request.hasKeyId() ? request.getKeyId() : "default-active-key")
                .setAlgorithm("RS256")
                .setPublicKeyPem("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...\n-----END PUBLIC KEY-----")
                .build();

        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }
}
