package com.hvduong.identity.grpc;

import com.hvduong.identity.application.authorization.SpecializedPermissionService;
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
}
