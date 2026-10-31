package com.hvduong.identity.api.admin;

import com.hvduong.identity.dto.common.ApiResponse;
import com.hvduong.identity.dto.request.AssignRoleRequest;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@Tag(name = "IAM Role Assignments", description = "Gán vai trò và phạm vi làm việc cho nhân viên nội bộ theo chuẩn RESTful (Mục 3.3.1)")
@RequestMapping({"/api/v1/admin/role-assignments", "/api/v1/admin/iam/assignments"})
public interface IamAssignmentApi {

    @Operation(summary = "Gán vai trò cho nhân viên [Chuẩn RESTful 3.3.1]", description = "Gán vai trò chức danh kèm phạm vi nghiệp vụ (GLOBAL, WAREHOUSE, STORE) qua POST /api/v1/admin/role-assignments.")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping
    ResponseEntity<ApiResponse<UUID>> assignRole(
            @RequestBody AssignRoleRequest request,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    );

    @Operation(summary = "Thu hồi vai trò đã gán [Chuẩn RESTful 3.3.1: HTTP DELETE]", description = "Thu hồi vai trò đã gán cho nhân viên bằng phương thức HTTP DELETE trên URI danh từ số nhiều.")
    @SecurityRequirement(name = "bearerAuth")
    @DeleteMapping("/{assignmentId}")
    ResponseEntity<ApiResponse<Void>> revokeRole(
            @Parameter(description = "ID của bản ghi gán vai trò", required = true) @PathVariable UUID assignmentId,
            @RequestParam(required = false) String reason,
            @RequestParam(required = false, defaultValue = "0") long expectedVersion,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorId
    );
}
