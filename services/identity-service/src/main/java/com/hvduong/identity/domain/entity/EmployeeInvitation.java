package com.hvduong.identity.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Domain entity representing an employee invitation.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class EmployeeInvitation {
    private UUID invitationId;
    private String email;
    private String invitationToken;
    private String status; // PENDING | ACCEPTED | EXPIRED | REVOKED
    private UUID invitedByPrincipalId;
    private OffsetDateTime expiresAt;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
}
