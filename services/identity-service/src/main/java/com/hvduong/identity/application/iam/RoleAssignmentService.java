package com.hvduong.identity.application.iam;

import com.hvduong.identity.api.admin.dto.AssignRoleRequest;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalRow;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.UUID;

@Service
@RequiredArgsConstructor
public class RoleAssignmentService {

    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final IdentityPrincipalMapper principalMapper;

    @Transactional
    public UUID assignRole(AssignRoleRequest request, UUID assignedBy) {
        // 1. Verify principal exists and is an EMPLOYEE
        IdentityPrincipalRow principal = principalMapper.findById(request.getPrincipalId());
        if (principal == null || !"EMPLOYEE".equals(principal.getPrincipalType())) {
            throw new IllegalArgumentException("Target principal must be an EMPLOYEE");
        }

        // 2. Validate scope
        if (!"GLOBAL".equals(request.getScopeType()) && (request.getScopeId() == null || request.getScopeId().trim().isEmpty())) {
            throw new IllegalArgumentException("Scope ID is required for non-GLOBAL assignments");
        }

        // 3. Create assignment row
        IamRoleAssignmentRow assignment = new IamRoleAssignmentRow();
        assignment.setAssignmentId(UUID.randomUUID());
        assignment.setPrincipalId(request.getPrincipalId());
        assignment.setRoleId(request.getRoleId());
        assignment.setScopeType(request.getScopeType());
        assignment.setScopeId(request.getScopeId());
        assignment.setAssignmentStatus("ACTIVE");
        assignment.setValidFrom(request.getValidFrom() != null ? request.getValidFrom() : OffsetDateTime.now());
        assignment.setValidUntil(request.getValidUntil());
        assignment.setAssignedByPrincipalId(assignedBy);
        assignment.setReason(request.getReason());
        assignment.setVersion(0L);

        roleAssignmentMapper.insert(assignment);
        
        return assignment.getAssignmentId();
    }
}
