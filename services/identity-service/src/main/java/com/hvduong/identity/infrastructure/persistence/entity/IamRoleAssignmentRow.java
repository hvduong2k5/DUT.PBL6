package com.hvduong.identity.infrastructure.persistence.entity;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class IamRoleAssignmentRow {
    private UUID assignmentId;
    private UUID principalId;
    private UUID roleId;
    private String scopeType;
    private String scopeId;
    private String assignmentStatus;
    private OffsetDateTime validFrom;
    private OffsetDateTime validUntil;
    private UUID assignedByPrincipalId;
    private UUID revokedByPrincipalId;
    private String reason;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
    private Long version;
}
