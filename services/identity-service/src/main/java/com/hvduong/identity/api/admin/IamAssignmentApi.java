package com.hvduong.identity.api.admin;

import com.hvduong.identity.api.admin.dto.AssignRoleRequest;
import com.hvduong.identity.api.common.ApiResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;

import java.util.UUID;

@Tag(name = "IAM Role Assignments", description = "Gán vai trò và phạm vi làm việc cho nhân viên nội bộ")
@RequestMapping("/api/v1/admin/iam/assignments")
public interface IamAssignmentApi {

    @Operation(summary = "Gán vai trò cho nhân viên", description = "Gán vai trò chức danh kèm phạm vi nghiệp vụ (GLOBAL, WAREHOUSE, STORE).")
    @SecurityRequirement(name = "bearerAuth")
    @PostMapping
    ResponseEntity<ApiResponse<UUID>> assignRole(@RequestBody AssignRoleRequest request);
}

