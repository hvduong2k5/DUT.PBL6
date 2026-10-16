package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class IdentityAuditLogRow {
    private UUID auditId;
    private OffsetDateTime occurredAt;
    private String actorKind;
    private UUID actorPrincipalId;
    private String actorClientId;
    private String actionCode;
    private String targetType;
    private String targetId;
    private String outcome;
    private String reasonCode;
    private String requestId;
    private String correlationId;
    private String sourceIp;
    private String details;
}

