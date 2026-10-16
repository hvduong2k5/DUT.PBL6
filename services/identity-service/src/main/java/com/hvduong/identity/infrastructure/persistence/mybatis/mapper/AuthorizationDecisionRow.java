package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class AuthorizationDecisionRow {
    private UUID decisionId;
    private String actorKind;
    private UUID actorPrincipalId;
    private String actorClientId;
    private String actionCode;
    private String decision;
    private String reasonCode;
    private UUID policyId;
    private Integer policyVersion;
    private String correlationId;
    private String contextSummary;
    private OffsetDateTime evaluatedAt;
}

