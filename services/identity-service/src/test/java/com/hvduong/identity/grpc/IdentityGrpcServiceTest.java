package com.hvduong.identity.grpc;

import com.hvduong.identity.application.authorization.SpecializedPermissionService;
import com.omamx.proto.common.v1.AuditContext;
import com.omamx.proto.identity.v1.SpecializedPermissionRequest;
import com.omamx.proto.identity.v1.SpecializedPermissionResponse;
import io.grpc.stub.StreamObserver;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class IdentityGrpcServiceTest {

    @Mock
    private SpecializedPermissionService permissionService;

    @Mock
    private StreamObserver<SpecializedPermissionResponse> permissionObserver;

    private IdentityGrpcService grpcService;

    @BeforeEach
    void setUp() {
        grpcService = new IdentityGrpcService(permissionService);
    }

    @Test
    void shouldHandleCheckSpecializedPermission_Allowed() {
        // Arrange
        when(permissionService.evaluatePermission(eq("user-1"), eq("FINANCE_REFUND_APPROVE"), anyMap(), any(), any()))
                .thenReturn(SpecializedPermissionService.EvaluationResult.builder()
                        .allowed(true)
                        .reason("Approved within threshold")
                        .build());

        AuditContext audit = AuditContext.newBuilder()
                .setTraceparent("trace-123")
                .setClientIp("192.168.1.1")
                .build();

        SpecializedPermissionRequest request = SpecializedPermissionRequest.newBuilder()
                .setUserId("user-1")
                .setAction("FINANCE_REFUND_APPROVE")
                .putContextAttributes("refund_amount", "5000000")
                .setAuditContext(audit)
                .build();

        // Act
        grpcService.checkSpecializedPermission(request, permissionObserver);

        // Assert
        ArgumentCaptor<SpecializedPermissionResponse> captor = ArgumentCaptor.forClass(SpecializedPermissionResponse.class);
        verify(permissionObserver, times(1)).onNext(captor.capture());
        verify(permissionObserver, times(1)).onCompleted();

        SpecializedPermissionResponse response = captor.getValue();
        assertThat(response.getIsAllowed()).isTrue();
        assertThat(response.getDecisionReason()).isEqualTo("Approved within threshold");
    }

    @Test
    void shouldHandleCheckSpecializedPermission_DeniedWithApprovalLevel() {
        // Arrange
        when(permissionService.evaluatePermission(eq("user-2"), eq("FINANCE_REFUND_APPROVE"), anyMap(), any(), any()))
                .thenReturn(SpecializedPermissionService.EvaluationResult.builder()
                        .allowed(false)
                        .reason("Exceeds limit")
                        .requiredApprovalLevel("DIRECTOR_APPROVAL_REQUIRED")
                        .build());

        SpecializedPermissionRequest request = SpecializedPermissionRequest.newBuilder()
                .setUserId("user-2")
                .setAction("FINANCE_REFUND_APPROVE")
                .putContextAttributes("refund_amount", "25000000")
                .build();

        // Act
        grpcService.checkSpecializedPermission(request, permissionObserver);

        // Assert
        ArgumentCaptor<SpecializedPermissionResponse> captor = ArgumentCaptor.forClass(SpecializedPermissionResponse.class);
        verify(permissionObserver, times(1)).onNext(captor.capture());
        verify(permissionObserver, times(1)).onCompleted();

        SpecializedPermissionResponse response = captor.getValue();
        assertThat(response.getIsAllowed()).isFalse();
        assertThat(response.getRequiredApprovalLevel()).isEqualTo("DIRECTOR_APPROVAL_REQUIRED");
    }
}
