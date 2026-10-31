package com.hvduong.identity.application.authorization.impl;

import com.hvduong.identity.application.authorization.SpecializedPermissionService;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.infrastructure.persistence.entity.AuthorizationDecisionRow;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityAuditLogRow;
import com.hvduong.identity.infrastructure.persistence.entity.SpecializedPermissionPolicyRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.AuthorizationDecisionMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityAuditLogMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.SpecializedPermissionPolicyMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.time.OffsetDateTime;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@Slf4j
@Service
@RequiredArgsConstructor
public class SpecializedPermissionServiceImpl implements SpecializedPermissionService {

    private final SpecializedPermissionPolicyMapper policyMapper;
    private final AuthorizationDecisionMapper decisionMapper;
    private final IdentityAuditLogMapper auditLogMapper;
    private final ObjectMapper objectMapper;

    @Override
    @Transactional
    public EvaluationResult evaluatePermission(
            String userId,
            String action,
            Map<String, String> contextAttributes,
            String correlationId,
            String clientIp
    ) {
        log.info("Evaluating specialized permission for user: {}, action: {}, correlation: {}", userId, action, correlationId);

        UUID actorPrincipalId = null;
        try {
            if (userId != null && !userId.isBlank()) {
                actorPrincipalId = UUID.fromString(userId);
            }
        } catch (IllegalArgumentException ignored) {
            // userId may be external subject or service identifier
        }

        List<SpecializedPermissionPolicyRow> policies = policyMapper.findActiveByActionCode(action);

        EvaluationResult result;
        UUID policyId = null;
        Integer policyVersion = null;

        if (policies.isEmpty()) {
            // Default policy when not configured: Fail-closed (DENY)
            result = EvaluationResult.builder()
                    .allowed(false)
                    .reasonCode("POLICY_NOT_FOUND")
                    .reason("No active policy found for action: " + action + ". Denying by default (fail-closed).")
                    .requiredApprovalLevel("ADMIN_CONFIG_REQUIRED")
                    .build();
        } else {
            SpecializedPermissionPolicyRow policy = policies.get(0);
            policyId = policy.getPolicyId();
            policyVersion = policy.getPolicyVersion();
            result = evaluatePolicyRule(policy, contextAttributes);
        }

        // 1. Record authorization decision
        UUID decisionId = UUID.randomUUID();
        AuthorizationDecisionRow decisionRow = new AuthorizationDecisionRow();
        decisionRow.setDecisionId(decisionId);
        decisionRow.setActorKind(actorPrincipalId != null ? "EMPLOYEE" : "SERVICE");
        decisionRow.setActorPrincipalId(actorPrincipalId);
        decisionRow.setActorClientId(actorPrincipalId == null ? userId : null);
        decisionRow.setActionCode(action);
        decisionRow.setDecision(result.isAllowed() ? "ALLOW" : (result.getRequiredApprovalLevel() != null ? "REQUIRES_APPROVAL" : "DENY"));
        decisionRow.setReasonCode(result.getReasonCode());
        decisionRow.setPolicyId(policyId);
        decisionRow.setPolicyVersion(policyVersion);
        decisionRow.setCorrelationId(correlationId);
        try {
            decisionRow.setContextSummary(objectMapper.writeValueAsString(contextAttributes));
        } catch (Exception e) {
            decisionRow.setContextSummary("{}");
        }
        decisionRow.setEvaluatedAt(OffsetDateTime.now());
        decisionMapper.insert(decisionRow);

        // 2. Append to audit log
        IdentityAuditLogRow auditLogRow = new IdentityAuditLogRow();
        auditLogRow.setAuditId(UUID.randomUUID());
        auditLogRow.setOccurredAt(OffsetDateTime.now());
        auditLogRow.setActorKind(actorPrincipalId != null ? "EMPLOYEE" : "SERVICE");
        auditLogRow.setActorPrincipalId(actorPrincipalId);
        auditLogRow.setActorClientId(actorPrincipalId == null ? userId : null);
        auditLogRow.setActionCode("SPECIALIZED_PERMISSION_CHECK:" + action);
        auditLogRow.setTargetType("ACTION");
        auditLogRow.setTargetId(action);
        auditLogRow.setOutcome(result.isAllowed() ? "SUCCESS" : "DENIED");
        auditLogRow.setReasonCode(result.getReasonCode());
        auditLogRow.setCorrelationId(correlationId);
        auditLogRow.setSourceIp(clientIp);
        try {
            auditLogRow.setDetails(objectMapper.writeValueAsString(Map.of(
                    "decisionId", decisionId.toString(),
                    "allowed", result.isAllowed(),
                    "reason", result.getReason(),
                    "requiredApprovalLevel", result.getRequiredApprovalLevel() != null ? result.getRequiredApprovalLevel() : ""
            )));
        } catch (Exception e) {
            auditLogRow.setDetails("{}");
        }
        auditLogMapper.insert(auditLogRow);

        return result;
    }

    private EvaluationResult evaluatePolicyRule(SpecializedPermissionPolicyRow policy, Map<String, String> context) {
        String policyType = policy.getPolicyType();
        String ruleConfigJson = policy.getRuleConfig();

        try {
            JsonNode config = ruleConfigJson != null ? objectMapper.readTree(ruleConfigJson) : null;

            if ("AMOUNT_THRESHOLD".equalsIgnoreCase(policyType)) {
                String attrKey = config != null && config.has("attribute_key")
                        ? config.get("attribute_key").asText() : "refund_amount";
                BigDecimal maxThreshold = config != null && config.has("threshold_amount")
                        ? new BigDecimal(config.get("threshold_amount").asText()) : new BigDecimal("10000000");

                String amountStr = context != null ? context.get(attrKey) : null;
                if (amountStr == null) {
                    amountStr = context != null ? context.get("amount") : null;
                }

                if (amountStr == null) {
                    return EvaluationResult.builder()
                            .allowed(false)
                            .reasonCode("MISSING_AMOUNT_CONTEXT")
                            .reason("Context attribute '" + attrKey + "' is required for amount threshold evaluation")
                            .requiredApprovalLevel(formatApprovalLevel(policy.getRequiredApprovalLevel()))
                            .build();
                }

                BigDecimal requestedAmount = new BigDecimal(amountStr);
                if (requestedAmount.compareTo(maxThreshold) > 0) {
                    return EvaluationResult.builder()
                            .allowed(false)
                            .reasonCode("THRESHOLD_EXCEEDED")
                            .reason("Requested amount " + requestedAmount + " exceeds policy threshold " + maxThreshold)
                            .requiredApprovalLevel(formatApprovalLevel(policy.getRequiredApprovalLevel()))
                            .build();
                }

                return EvaluationResult.builder()
                        .allowed(true)
                        .reasonCode("WITHIN_THRESHOLD")
                        .reason("Amount is within permitted policy limit")
                        .build();

            } else if ("PERCENT_THRESHOLD".equalsIgnoreCase(policyType)) {
                String attrKey = config != null && config.has("attribute_key")
                        ? config.get("attribute_key").asText() : "discount_percentage";
                BigDecimal maxPercent = config != null && config.has("max_discount_percentage")
                        ? new BigDecimal(config.get("max_discount_percentage").asText()) : new BigDecimal("25");

                String percentStr = context != null ? context.get(attrKey) : null;
                if (percentStr == null) {
                    return EvaluationResult.builder()
                            .allowed(false)
                            .reasonCode("MISSING_PERCENT_CONTEXT")
                            .reason("Context attribute '" + attrKey + "' is required for percent threshold evaluation")
                            .requiredApprovalLevel(formatApprovalLevel(policy.getRequiredApprovalLevel()))
                            .build();
                }

                BigDecimal requestedPercent = new BigDecimal(percentStr);
                if (requestedPercent.compareTo(maxPercent) > 0) {
                    return EvaluationResult.builder()
                            .allowed(false)
                            .reasonCode("PERCENT_EXCEEDED")
                            .reason("Requested percentage " + requestedPercent + "% exceeds limit " + maxPercent + "%")
                            .requiredApprovalLevel(formatApprovalLevel(policy.getRequiredApprovalLevel()))
                            .build();
                }

                return EvaluationResult.builder()
                        .allowed(true)
                        .reasonCode("WITHIN_PERCENT_LIMIT")
                        .reason("Discount percentage is within allowed policy limit")
                        .build();
            } else {
                return EvaluationResult.builder()
                        .allowed(true)
                        .reasonCode("POLICY_MATCHED")
                        .reason("Policy evaluation passed")
                        .build();
            }

        } catch (Exception ex) {
            log.error("Failed to parse policy rule for action: {}", policy.getActionCode(), ex);
            return EvaluationResult.builder()
                    .allowed(false)
                    .reasonCode("POLICY_EVALUATION_ERROR")
                    .reason("Internal policy evaluation error: " + ex.getMessage())
                    .requiredApprovalLevel(formatApprovalLevel(policy.getRequiredApprovalLevel()))
                    .build();
        }
    }

    private String formatApprovalLevel(Integer level) {
        if (level == null || level <= 1) {
            return "MANAGER_APPROVAL_REQUIRED";
        } else if (level == 2) {
            return "DIRECTOR_APPROVAL_REQUIRED";
        } else {
            return "EXECUTIVE_APPROVAL_REQUIRED";
        }
    }
}

