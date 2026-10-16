package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class EmployeeInvitationRow {
    private UUID invitationId;
    private UUID principalId;
    private UUID invitedByPrincipalId;
    private String invitationStatus;
    private OffsetDateTime expiresAt;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
    private OffsetDateTime acceptedAt;
    private OffsetDateTime revokedAt;
}

