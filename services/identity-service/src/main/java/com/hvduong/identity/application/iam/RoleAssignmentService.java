package com.hvduong.identity.application.iam;

import com.hvduong.identity.application.dto.request.AssignRoleRequest;
import com.hvduong.identity.application.dto.request.UpdateRoleAssignmentRequest;
import com.hvduong.identity.application.dto.response.RoleAssignmentResponse;
import com.hvduong.identity.domain.entity.IamRoleAssignmentRow;

import java.util.List;
import java.util.UUID;

public interface RoleAssignmentService {

    UUID assignRole(AssignRoleRequest request, UUID assignedBy);

    RoleAssignmentResponse updateAssignment(UUID assignmentId, UpdateRoleAssignmentRequest request, UUID updatedBy);

    boolean revokeRole(UUID assignmentId, UUID revokedBy, String reason, long expectedVersion);

    List<IamRoleAssignmentRow> getActiveAssignments(UUID principalId);

    default List<RoleAssignmentResponse> listAssignments(UUID principalId, UUID roleId, String scopeType, String status, int page, int size) {
        return listAssignments(principalId, roleId, null, scopeType, null, status, page, size);
    }

    List<RoleAssignmentResponse> listAssignments(
            UUID principalId,
            UUID roleId,
            String roleCode,
            String scopeType,
            String scopeId,
            String status,
            int page,
            int size
    );
}
