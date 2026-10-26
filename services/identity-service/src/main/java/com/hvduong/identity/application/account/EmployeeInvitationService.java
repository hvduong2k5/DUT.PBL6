package com.hvduong.identity.application.account;

import lombok.Builder;
import lombok.Data;

import java.time.OffsetDateTime;
import java.util.UUID;

public interface EmployeeInvitationService {

    @Data
    @Builder
    class InviteResult {
        private UUID invitationId;
        private UUID principalId;
        private String status;
        private OffsetDateTime expiresAt;
    }

    InviteResult inviteEmployee(UUID invitedBy, int validDays);

    boolean acceptInvitation(UUID invitationId, String idpSubject);

    boolean revokeInvitation(UUID invitationId, UUID revokedBy);
}
