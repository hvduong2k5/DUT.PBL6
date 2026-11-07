package com.hvduong.identity.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Domain entity representing an identity audit log entry.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class IdentityAuditLog {
    private UUID logId;
    private UUID principalId;
    private UUID actorPrincipalId;
    private String eventType; // e.g. ROLE_ASSIGN, ROLE_REVOKE, PERMISSION_CHANGE, ROLE_DISABLE, ACCOUNT_SUSPEND
    private String eventDetail;
    private String ipAddress;
    private OffsetDateTime createdAt;
}
