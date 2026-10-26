package com.hvduong.identity.application.iam;

import com.hvduong.identity.api.admin.dto.AssignRoleRequest;
import com.hvduong.identity.infrastructure.persistence.entity.IamRoleAssignmentRow;

import java.util.List;
import java.util.UUID;

public interface RoleAssignmentService {

    UUID assignRole(AssignRoleRequest request, UUID assignedBy);

    boolean revokeRole(UUID assignmentId, UUID revokedBy, String reason, long expectedVersion);

    List<IamRoleAssignmentRow> getActiveAssignments(UUID principalId);
}
