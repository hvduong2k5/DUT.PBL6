package com.hvduong.identity.api.admin.impl;

import com.hvduong.identity.api.admin.IamAssignmentApi;
import com.hvduong.identity.application.iam.RoleAssignmentService;
import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.request.AssignRoleRequest;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RestController;

import java.util.UUID;

@RestController
@RequiredArgsConstructor
public class IamAssignmentController implements IamAssignmentApi {

    private final RoleAssignmentService roleAssignmentService;

    @Override
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('ROLE_ASSIGN')")
    public ResponseEntity<ApiResponse<UUID>> assignRole(
            @RequestBody AssignRoleRequest request,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    ) {
        UUID assignedBy = com.hvduong.identity.security.VerifiedActor.id();
        UUID assignmentId = roleAssignmentService.assignRole(request, assignedBy);
        return ResponseEntity.ok(ApiResponse.success("Gán vai trò cho nhân viên thành công", assignmentId));
    }

    @Override
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('ROLE_ASSIGN')")
    public ResponseEntity<ApiResponse<Void>> revokeRole(
            @PathVariable UUID assignmentId,
            @RequestParam(required = false) String reason,
            @RequestParam(required = false, defaultValue = "0") long expectedVersion,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    ) {
        UUID revokedBy = com.hvduong.identity.security.VerifiedActor.id();
        boolean revoked = roleAssignmentService.revokeRole(assignmentId, revokedBy, reason, expectedVersion);
        if (revoked) {
            return ResponseEntity.ok(ApiResponse.success("Thu hồi vai trò thành công", null));
        }
        return ResponseEntity.status(HttpStatus.BAD_REQUEST)
                .body(ApiResponse.error("REVOKE_FAILED", "Không thể thu hồi vai trò (bản ghi không tồn tại hoặc phiên bản xung đột)"));
    }
}
