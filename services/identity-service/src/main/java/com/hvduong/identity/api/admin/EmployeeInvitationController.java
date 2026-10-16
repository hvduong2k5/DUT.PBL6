package com.hvduong.identity.api.admin;

import com.hvduong.identity.application.account.EmployeeInvitationService;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/api/v1/admin/employees/invitations")
@RequiredArgsConstructor
public class EmployeeInvitationController {

    private final EmployeeInvitationService invitationService;

    @Data
    public static class InviteEmployeeRequest {
        private int validDays = 7;
    }

    @PostMapping
    public ResponseEntity<EmployeeInvitationService.InviteResult> inviteEmployee(
            @RequestBody(required = false) InviteEmployeeRequest request,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    ) {
        UUID invitedBy = null;
        try {
            if (actorId != null) {
                invitedBy = UUID.fromString(actorId);
            }
        } catch (Exception ignored) {}

        int validDays = request != null ? request.getValidDays() : 7;
        EmployeeInvitationService.InviteResult result = invitationService.inviteEmployee(invitedBy, validDays);
        return ResponseEntity.ok(result);
    }

    @PostMapping("/{invitationId}/revoke")
    public ResponseEntity<Void> revokeInvitation(
            @PathVariable UUID invitationId,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    ) {
        UUID revokedBy = null;
        try {
            if (actorId != null) {
                revokedBy = UUID.fromString(actorId);
            }
        } catch (Exception ignored) {}

        boolean revoked = invitationService.revokeInvitation(invitationId, revokedBy);
        return revoked ? ResponseEntity.noContent().build() : ResponseEntity.notFound().build();
    }
}

