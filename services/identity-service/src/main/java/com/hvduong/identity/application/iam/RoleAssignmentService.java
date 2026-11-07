package com.hvduong.identity.application.iam;

import com.hvduong.identity.application.dto.request.AssignRoleRequest;
import com.hvduong.identity.application.dto.response.RoleAssignmentResponse;
import com.hvduong.identity.domain.entity.IamRoleAssignmentRow;

import java.util.List;
import java.util.UUID;

public interface RoleAssignmentService {

    UUID assignRole(AssignRoleRequest request, UUID assignedBy);

    boolean revokeRole(UUID assignmentId, UUID revokedBy, String reason, long expectedVersion);

    List<IamRoleAssignmentRow> getActiveAssignments(UUID principalId);

    /** List assignments with optional filters (principalId, roleId, scopeType, status). Empty result = 200 with empty list. */
    List<RoleAssignmentResponse> listAssignments(UUID principalId, UUID roleId, String scopeType, String status, int page, int size);
}
