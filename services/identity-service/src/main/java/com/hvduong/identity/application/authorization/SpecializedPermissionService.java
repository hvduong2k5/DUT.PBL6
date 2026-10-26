package com.hvduong.identity.application.authorization;

import lombok.Builder;
import lombok.Data;

import java.util.Map;

public interface SpecializedPermissionService {

    @Data
    @Builder
    class EvaluationResult {
        private boolean allowed;
        private String reason;
        private String requiredApprovalLevel;
        private String reasonCode;
    }

    EvaluationResult evaluatePermission(
            String userId,
            String action,
            Map<String, String> contextAttributes,
            String correlationId,
            String clientIp
    );
}
