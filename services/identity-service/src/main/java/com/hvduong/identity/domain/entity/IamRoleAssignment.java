package com.hvduong.identity.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Domain entity representing an IAM role assignment to a principal.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class IamRoleAssignment {
    private UUID assignmentId;
    private UUID principalId;
    private UUID roleId;
    private String scopeType;
    private String scopeId;
    private String assignmentStatus; // ACTIVE | REVOKED | EXPIRED
    private UUID assignedByPrincipalId;
    private String reason;
    private OffsetDateTime effectiveFrom;
    private OffsetDateTime effectiveTo;
    private OffsetDateTime createdAt;
    private int version;
}
