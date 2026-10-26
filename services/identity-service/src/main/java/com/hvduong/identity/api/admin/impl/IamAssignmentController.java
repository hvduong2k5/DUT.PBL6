package com.hvduong.identity.api.admin.impl;

import com.hvduong.identity.api.admin.IamAssignmentApi;
import com.hvduong.identity.api.admin.dto.AssignRoleRequest;
import com.hvduong.identity.api.common.ApiResponse;
import com.hvduong.identity.application.iam.RoleAssignmentService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

import java.util.UUID;

@RestController
@RequiredArgsConstructor
public class IamAssignmentController implements IamAssignmentApi {

    private final RoleAssignmentService roleAssignmentService;

    @Override
    public ResponseEntity<ApiResponse<UUID>> assignRole(@RequestBody AssignRoleRequest request) {
        UUID assignedBy = UUID.fromString("00000000-0000-0000-0000-000000000001"); 
        UUID assignmentId = roleAssignmentService.assignRole(request, assignedBy);
        return ResponseEntity.ok(ApiResponse.success("Gán vai trò cho nhân viên thành công", assignmentId));
    }
}
