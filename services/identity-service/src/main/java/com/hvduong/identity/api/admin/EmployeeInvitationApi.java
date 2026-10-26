package com.hvduong.identity.api.admin;

import com.hvduong.identity.api.common.ApiResponse;
import com.hvduong.identity.application.account.EmployeeInvitationService;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.Data;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@Tag(name = "Employee Invitations", description = "Quản lý lời mời tạo tài khoản cho nhân viên nội bộ")
@RequestMapping("/api/v1/admin/employees/invitations")
public interface EmployeeInvitationApi {

    @Data
    class InviteEmployeeRequest {
        private int validDays = 7;
    }

    @Operation(summary = "Tạo thư mời nhân viên", description = "Sinh mã lời mời token kèm thời hạn có hiệu lực (mặc định 7 ngày).")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping
    ResponseEntity<ApiResponse<EmployeeInvitationService.InviteResult>> inviteEmployee(
            @RequestBody(required = false) InviteEmployeeRequest request,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    );

    @Operation(summary = "Thu hồi thư mời nhân viên", description = "Hủy bỏ hiệu lực của thư mời đã gửi.")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping("/{invitationId}/revoke")
    ResponseEntity<ApiResponse<Void>> revokeInvitation(
            @Parameter(description = "ID của thư mời", required = true) @PathVariable UUID invitationId,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    );
}

