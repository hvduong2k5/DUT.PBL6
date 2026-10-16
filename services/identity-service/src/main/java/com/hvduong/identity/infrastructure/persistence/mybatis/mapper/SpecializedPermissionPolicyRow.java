package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class SpecializedPermissionPolicyRow {
    private UUID policyId;
    private String policyCode;
    private Integer policyVersion;
    private String actionCode;
    private String policyName;
    private String policyType;
    private String scopeType;
    private String scopeId;
    private String ruleConfig;
    private Integer requiredApprovalLevel;
    private Boolean failClosed;
    private String lifecycleStatus;
    private OffsetDateTime validFrom;
    private OffsetDateTime validUntil;
    private UUID createdByPrincipalId;
    private OffsetDateTime createdAt;
}

