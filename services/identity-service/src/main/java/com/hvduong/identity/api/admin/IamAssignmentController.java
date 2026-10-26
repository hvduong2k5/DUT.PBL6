package com.hvduong.identity.api.admin;

import com.hvduong.identity.api.admin.dto.AssignRoleRequest;
import com.hvduong.identity.api.common.ApiResponse;
import com.hvduong.identity.application.iam.RoleAssignmentService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/api/v1/admin/iam/assignments")
@RequiredArgsConstructor
public class IamAssignmentController {

    private final RoleAssignmentService roleAssignmentService;

    @PostMapping
    public ResponseEntity<ApiResponse<UUID>> assignRole(@RequestBody AssignRoleRequest request) {
        // In a real scenario, assignedBy is extracted from the JWT/SecurityContext
        UUID assignedBy = UUID.fromString("00000000-0000-0000-0000-000000000001"); 
        UUID assignmentId = roleAssignmentService.assignRole(request, assignedBy);
        return ResponseEntity.ok(ApiResponse.success("Gán vai trò cho nhân viên thành công", assignmentId));
    }
}
