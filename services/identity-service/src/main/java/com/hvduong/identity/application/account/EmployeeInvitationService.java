package com.hvduong.identity.application.account;

import com.hvduong.identity.application.dto.response.InviteEmployeeResponse;

import java.util.UUID;

public interface EmployeeInvitationService {

    /**
     * @deprecated Sử dụng {@link InviteEmployeeResponse}
     */
    @Deprecated
    class InviteResult extends InviteEmployeeResponse {}

    InviteEmployeeResponse inviteEmployee(UUID invitedBy, int validDays, String idpSubject);

    void acceptInvitation(UUID invitationId, String issuer, String idpSubject);

    boolean revokeInvitation(UUID invitationId, UUID revokedBy);
}
