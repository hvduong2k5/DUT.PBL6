package com.hvduong.identity.application.authorization;

import com.hvduong.identity.application.authorization.impl.SpecializedPermissionServiceImpl;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.AuthorizationDecisionMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityAuditLogMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.SpecializedPermissionPolicyMapper;
import com.hvduong.identity.infrastructure.persistence.entity.SpecializedPermissionPolicyRow;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.OffsetDateTime;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class SpecializedPermissionServiceTest {

    @Mock
    private SpecializedPermissionPolicyMapper policyMapper;

    @Mock
    private AuthorizationDecisionMapper decisionMapper;

    @Mock
    private IdentityAuditLogMapper auditLogMapper;

    private SpecializedPermissionService permissionService;

    private final ObjectMapper objectMapper = new ObjectMapper();

    @BeforeEach
    void setUp() {
        permissionService = new SpecializedPermissionServiceImpl(
                policyMapper,
                decisionMapper,
                auditLogMapper,
                objectMapper
        );
    }

    @Test
    void shouldDenyWhenNoPolicyFound_FailClosed() {
        // Arrange
        when(policyMapper.findActiveByActionCode("UNKNOWN_ACTION")).thenReturn(Collections.emptyList());

        // Act
        SpecializedPermissionService.EvaluationResult result = permissionService.evaluatePermission(
                UUID.randomUUID().toString(),
                "UNKNOWN_ACTION",
                Map.of(),
                "trace-1",
                "127.0.0.1"
        );

        // Assert
        assertThat(result.isAllowed()).isFalse();
        assertThat(result.getReasonCode()).isEqualTo("POLICY_NOT_FOUND");
        verify(decisionMapper, times(1)).insert(any());
        verify(auditLogMapper, times(1)).insert(any());
    }

    @Test
    void shouldAllowWhenAmountWithinThreshold() {
        // Arrange
        UUID policyId = UUID.randomUUID();
        SpecializedPermissionPolicyRow policy = new SpecializedPermissionPolicyRow();
        policy.setPolicyId(policyId);
        policy.setPolicyCode("POL_REFUND_LIMIT");
        policy.setPolicyVersion(1);
        policy.setActionCode("FINANCE_REFUND_APPROVE");
        policy.setPolicyType("AMOUNT_THRESHOLD");
        policy.setRuleConfig("{\"threshold_amount\": 10000000, \"attribute_key\": \"refund_amount\"}");
        policy.setRequiredApprovalLevel(2);
        policy.setValidFrom(OffsetDateTime.now().minusDays(1));

        when(policyMapper.findActiveByActionCode("FINANCE_REFUND_APPROVE")).thenReturn(List.of(policy));

        // Act: 5,000,000 VND <= 10,000,000 VND
        SpecializedPermissionService.EvaluationResult result = permissionService.evaluatePermission(
                UUID.randomUUID().toString(),
                "FINANCE_REFUND_APPROVE",
                Map.of("refund_amount", "5000000"),
                "trace-2",
                "127.0.0.1"
        );

        // Assert
        assertThat(result.isAllowed()).isTrue();
        assertThat(result.getReasonCode()).isEqualTo("WITHIN_THRESHOLD");
        verify(decisionMapper, times(1)).insert(any());
    }

    @Test
    void shouldRequireApprovalWhenAmountExceedsThreshold() {
        // Arrange
        UUID policyId = UUID.randomUUID();
        SpecializedPermissionPolicyRow policy = new SpecializedPermissionPolicyRow();
        policy.setPolicyId(policyId);
        policy.setPolicyCode("POL_REFUND_LIMIT");
        policy.setPolicyVersion(1);
        policy.setActionCode("FINANCE_REFUND_APPROVE");
        policy.setPolicyType("AMOUNT_THRESHOLD");
        policy.setRuleConfig("{\"threshold_amount\": 10000000, \"attribute_key\": \"refund_amount\"}");
        policy.setRequiredApprovalLevel(2); // Level 2 = Director
        policy.setValidFrom(OffsetDateTime.now().minusDays(1));

        when(policyMapper.findActiveByActionCode("FINANCE_REFUND_APPROVE")).thenReturn(List.of(policy));

        // Act: 15,000,000 VND > 10,000,000 VND
        SpecializedPermissionService.EvaluationResult result = permissionService.evaluatePermission(
                UUID.randomUUID().toString(),
                "FINANCE_REFUND_APPROVE",
                Map.of("refund_amount", "15000000"),
                "trace-3",
                "127.0.0.1"
        );

        // Assert
        assertThat(result.isAllowed()).isFalse();
        assertThat(result.getReasonCode()).isEqualTo("THRESHOLD_EXCEEDED");
        assertThat(result.getRequiredApprovalLevel()).isEqualTo("DIRECTOR_APPROVAL_REQUIRED");
        verify(decisionMapper, times(1)).insert(any());
    }

    @Test
    void shouldAllowWhenDiscountWithinPercentLimit() {
        // Arrange
        UUID policyId = UUID.randomUUID();
        SpecializedPermissionPolicyRow policy = new SpecializedPermissionPolicyRow();
        policy.setPolicyId(policyId);
        policy.setPolicyCode("POL_B2B_DISCOUNT");
        policy.setPolicyVersion(1);
        policy.setActionCode("B2B_DISCOUNT_APPLY");
        policy.setPolicyType("PERCENT_THRESHOLD");
        policy.setRuleConfig("{\"max_discount_percentage\": 25, \"attribute_key\": \"discount_percentage\"}");
        policy.setRequiredApprovalLevel(2);

        when(policyMapper.findActiveByActionCode("B2B_DISCOUNT_APPLY")).thenReturn(List.of(policy));

        // Act: 20% <= 25%
        SpecializedPermissionService.EvaluationResult result = permissionService.evaluatePermission(
                UUID.randomUUID().toString(),
                "B2B_DISCOUNT_APPLY",
                Map.of("discount_percentage", "20"),
                "trace-4",
                "127.0.0.1"
        );

        // Assert
        assertThat(result.isAllowed()).isTrue();
        assertThat(result.getReasonCode()).isEqualTo("WITHIN_PERCENT_LIMIT");
    }

    @Test
    void shouldRequireApprovalWhenDiscountExceedsPercentLimit() {
        // Arrange
        UUID policyId = UUID.randomUUID();
        SpecializedPermissionPolicyRow policy = new SpecializedPermissionPolicyRow();
        policy.setPolicyId(policyId);
        policy.setPolicyCode("POL_B2B_DISCOUNT");
        policy.setPolicyVersion(1);
        policy.setActionCode("B2B_DISCOUNT_APPLY");
        policy.setPolicyType("PERCENT_THRESHOLD");
        policy.setRuleConfig("{\"max_discount_percentage\": 25, \"attribute_key\": \"discount_percentage\"}");
        policy.setRequiredApprovalLevel(2);

        when(policyMapper.findActiveByActionCode("B2B_DISCOUNT_APPLY")).thenReturn(List.of(policy));

        // Act: 30% > 25%
        SpecializedPermissionService.EvaluationResult result = permissionService.evaluatePermission(
                UUID.randomUUID().toString(),
                "B2B_DISCOUNT_APPLY",
                Map.of("discount_percentage", "30"),
                "trace-5",
                "127.0.0.1"
        );

        // Assert
        assertThat(result.isAllowed()).isFalse();
        assertThat(result.getReasonCode()).isEqualTo("PERCENT_EXCEEDED");
        assertThat(result.getRequiredApprovalLevel()).isEqualTo("DIRECTOR_APPROVAL_REQUIRED");
    }
}

