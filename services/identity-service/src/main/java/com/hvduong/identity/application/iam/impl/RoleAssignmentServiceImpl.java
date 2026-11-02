package com.hvduong.identity.application.iam.impl;

import com.hvduong.identity.application.iam.RoleAssignmentService;

import com.hvduong.identity.dto.request.AssignRoleRequest;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.entity.IamRoleAssignmentRow;
import com.hvduong.identity.infrastructure.persistence.entity.IamRolePermissionRow;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRolePermissionMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.*;

@Service
@RequiredArgsConstructor
public class RoleAssignmentServiceImpl implements RoleAssignmentService {

    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final IdentityPrincipalMapper principalMapper;
    private final AuditService auditService;

    @Autowired(required = false)
    private KeycloakAdminClient keycloakAdminClient;

    @Autowired(required = false)
    private IamRolePermissionMapper rolePermissionMapper;

    @Override
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
        UUID assignmentId = UUID.randomUUID();
        IamRoleAssignmentRow assignment = new IamRoleAssignmentRow();
        assignment.setAssignmentId(assignmentId);
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

        // Sync permissions to Keycloak User Attributes
        syncPermissionsToKeycloak(request.getPrincipalId());

        // 4. Audit log
        auditService.recordAudit(
                "EMPLOYEE",
                assignedBy,
                null,
                "IAM_ROLE_ASSIGN",
                "IAM_ASSIGNMENT",
                assignmentId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of(
                        "targetPrincipalId", request.getPrincipalId().toString(),
                        "roleId", request.getRoleId().toString(),
                        "scopeType", request.getScopeType()
                )
        );

        return assignmentId;
    }

    @Override
    @Transactional
    public boolean revokeRole(UUID assignmentId, UUID revokedBy, String reason, long expectedVersion) {
        IamRoleAssignmentRow existing = roleAssignmentMapper.findById(assignmentId);
        int updated = roleAssignmentMapper.revokeAssignment(assignmentId, revokedBy, reason, expectedVersion);
        if (updated > 0) {
            if (existing != null) {
                syncPermissionsToKeycloak(existing.getPrincipalId());
            }

            auditService.recordAudit(
                    "EMPLOYEE",
                    revokedBy,
                    null,
                    "IAM_ROLE_REVOKE",
                    "IAM_ASSIGNMENT",
                    assignmentId.toString(),
                    "SUCCESS",
                    null,
                    null,
                    null,
                    Map.of("reason", reason != null ? reason : "")
            );
            return true;
        }
        return false;
    }

    @Override
    @Transactional(readOnly = true)
    public List<IamRoleAssignmentRow> getActiveAssignments(UUID principalId) {
        return roleAssignmentMapper.findActiveByPrincipal(principalId, OffsetDateTime.now());
    }

    private void syncPermissionsToKeycloak(UUID principalId) {
        if (keycloakAdminClient == null || rolePermissionMapper == null || principalId == null) {
            return;
        }
        try {
            IdentityPrincipalRow principal = principalMapper.findById(principalId);
            if (principal == null || principal.getIdpSubject() == null) {
                return;
            }
            List<IamRoleAssignmentRow> activeAssignments = roleAssignmentMapper.findActiveByPrincipal(principalId, OffsetDateTime.now());
            Set<String> perms = new HashSet<>();
            if (activeAssignments != null) {
                for (IamRoleAssignmentRow a : activeAssignments) {
                    if (a.getRoleId() != null) {
                        List<IamRolePermissionRow> rolePerms = rolePermissionMapper.findByRoleId(a.getRoleId());
                        if (rolePerms != null) {
                            for (IamRolePermissionRow p : rolePerms) {
                                perms.add(p.getPermissionCode());
                            }
                        }
                    }
                }
            }
            keycloakAdminClient.updateUserPermissions(principal.getIdpSubject(), new ArrayList<>(perms));
        } catch (Exception ignored) {}
    }
}
