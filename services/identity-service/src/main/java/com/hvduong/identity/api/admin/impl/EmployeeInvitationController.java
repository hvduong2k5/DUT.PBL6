package com.hvduong.identity.api.admin.impl;

import com.hvduong.identity.api.admin.EmployeeInvitationApi;
import com.hvduong.identity.application.account.EmployeeInvitationService;
import com.hvduong.identity.common.response.ApiResponse;
import com.hvduong.identity.application.dto.request.InviteEmployeeRequest;
import com.hvduong.identity.application.dto.response.InviteEmployeeResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RestController;

import java.util.UUID;

@RestController
@RequiredArgsConstructor
public class EmployeeInvitationController implements EmployeeInvitationApi {

    private final EmployeeInvitationService invitationService;

    @Override
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('EMPLOYEE_INVITE')")
    public ResponseEntity<ApiResponse<InviteEmployeeResponse>> inviteEmployee(
            @RequestBody(required = false) InviteEmployeeRequest request,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    ) {
        UUID invitedBy = com.hvduong.identity.security.VerifiedActor.id();
        int validDays = request != null ? request.getValidDays() : 7;
        InviteEmployeeResponse result = invitationService.inviteEmployee(invitedBy, validDays);
        return ResponseEntity.ok(ApiResponse.success("Tạo thư mời nhân viên thành công", result));
    }

    @Override
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('EMPLOYEE_INVITATION_REVOKE')")
    public ResponseEntity<ApiResponse<Void>> revokeInvitation(
            @PathVariable UUID invitationId,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    ) {
        UUID revokedBy = com.hvduong.identity.security.VerifiedActor.id();
        boolean revoked = invitationService.revokeInvitation(invitationId, revokedBy);
        if (revoked) {
            return ResponseEntity.ok(ApiResponse.success("Thu hồi thư mời nhân viên thành công", null));
        } else {
            return ResponseEntity.status(HttpStatus.NOT_FOUND)
                    .body(ApiResponse.error("INVITATION_NOT_FOUND", "Không tìm thấy thư mời hoặc thư mời đã được xử lý"));
        }
    }
}
