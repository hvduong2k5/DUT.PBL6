package com.hvduong.identity.application.account;

import com.hvduong.identity.dto.response.InviteEmployeeResponse;

import java.util.UUID;

public interface EmployeeInvitationService {

    /**
     * @deprecated Sử dụng {@link InviteEmployeeResponse}
     */
    @Deprecated
    class InviteResult extends InviteEmployeeResponse {}

    InviteEmployeeResponse inviteEmployee(UUID invitedBy, int validDays);

    boolean acceptInvitation(UUID invitationId, String idpSubject);

    boolean revokeInvitation(UUID invitationId, UUID revokedBy);
}
