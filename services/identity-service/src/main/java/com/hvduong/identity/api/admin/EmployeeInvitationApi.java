package com.hvduong.identity.api.admin;

import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.request.InviteEmployeeRequest;
import com.hvduong.identity.dto.response.InviteEmployeeResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@Tag(name = "Employee Invitations", description = "Quản lý lời mời tạo tài khoản cho nhân viên nội bộ theo chuẩn RESTful (Mục 3.3.1)")
@RequestMapping({"/api/v1/admin/employee-invitations", "/api/v1/admin/employees/invitations"})
public interface EmployeeInvitationApi {

    @Operation(summary = "Tạo thư mời nhân viên [Chuẩn RESTful 3.3.1]", description = "Sinh mã lời mời token kèm thời hạn có hiệu lực (mặc định 7 ngày) qua POST /api/v1/admin/employee-invitations.")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping
    ResponseEntity<ApiResponse<InviteEmployeeResponse>> inviteEmployee(
            @RequestBody(required = false) InviteEmployeeRequest request,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    );

    @Operation(summary = "Thu hồi thư mời nhân viên [Chuẩn RESTful 3.3.1: HTTP DELETE]", description = "Hủy bỏ hiệu lực của thư mời đã gửi bằng phương thức HTTP DELETE trên URI danh từ số nhiều.")
    @SecurityRequirement(name = "bearerAuth")
    @DeleteMapping("/{invitationId}")
    ResponseEntity<ApiResponse<Void>> revokeInvitation(
            @Parameter(description = "ID của thư mời", required = true) @PathVariable UUID invitationId,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    );

    @Operation(summary = "Thu hồi thư mời nhân viên [Legacy Alias]", description = "Hỗ trợ tương thích ngược endpoint POST /{invitationId}/revoke.")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping("/{invitationId}/revoke")
    default ResponseEntity<ApiResponse<Void>> revokeInvitationLegacy(
            @Parameter(description = "ID của thư mời", required = true) @PathVariable UUID invitationId,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    ) {
        return revokeInvitation(invitationId, actorId);
    }
}
