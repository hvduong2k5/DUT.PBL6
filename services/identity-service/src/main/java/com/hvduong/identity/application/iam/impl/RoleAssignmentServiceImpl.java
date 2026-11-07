package com.hvduong.identity.application.iam.impl;

import com.hvduong.identity.application.iam.RoleAssignmentService;

import com.hvduong.identity.application.dto.request.AssignRoleRequest;
import com.hvduong.identity.application.dto.response.RoleAssignmentResponse;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.domain.entity.IamRoleAssignmentRow;
import com.hvduong.identity.domain.entity.IamRolePermissionRow;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRolePermissionMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.session.BffSessionStore;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.*;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class RoleAssignmentServiceImpl implements RoleAssignmentService {

    private static final String ADMIN_ROLE_CODE = "ADMIN";

    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final IdentityPrincipalMapper principalMapper;
    private final AuditService auditService;

    @Autowired(required = false)
    private KeycloakAdminClient keycloakAdminClient;

    @Autowired(required = false)
    private IamRolePermissionMapper rolePermissionMapper;

    @Autowired(required = false)
    private IamRoleMapper roleMapper;

    @Autowired(required = false)
    private BffSessionStore bffSessionStore;

    @Override
    @Transactional
    public UUID assignRole(AssignRoleRequest request, UUID assignedBy) {
        // Guard: self-assignment forbidden
        if (assignedBy != null && assignedBy.equals(request.getPrincipalId())) {
            throw new IllegalArgumentException("Self-assignment is forbidden: actor cannot assign a role to themselves");
        }

        // Verify role exists and is active (is_enabled = TRUE) when roleMapper is available
        if (roleMapper != null && request.getRoleId() != null) {
            var roleRow = roleMapper.findById(request.getRoleId());
            if (roleRow == null || Boolean.FALSE.equals(roleRow.getIsEnabled())) {
                throw new NoSuchElementException("Active role not found with id: " + request.getRoleId());
            }
        }

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
        if (existing == null) {
            return false;
        }

        // Last-admin protection: prevent revoking the last active ADMIN
        if (existing.getRoleId() != null && roleMapper != null) {
            var roleRow = roleMapper.findById(existing.getRoleId());
            if (roleRow != null && ADMIN_ROLE_CODE.equals(roleRow.getRoleCode())) {
                int activeAdmins = roleAssignmentMapper.countActiveAdmins(ADMIN_ROLE_CODE, OffsetDateTime.now());
                if (activeAdmins <= 1) {
                    throw new IllegalStateException("Cannot revoke the last active ADMIN assignment. At least one ADMIN must remain.");
                }
            }
        }

        int updated = roleAssignmentMapper.revokeAssignment(assignmentId, revokedBy, reason, expectedVersion);
        if (updated > 0) {
            if (existing != null) {
                syncAndForceRelogin(existing.getPrincipalId());
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

    @Override
    @Transactional(readOnly = true)
    public List<RoleAssignmentResponse> listAssignments(UUID principalId, UUID roleId, String scopeType, String status, int page, int size) {
        int offset = page * size;
        List<IamRoleAssignmentRow> rows = roleAssignmentMapper.listAssignments(principalId, roleId, scopeType, status, offset, size);
        if (rows == null) return Collections.emptyList();
        return rows.stream().map(this::toResponse).collect(Collectors.toList());
    }

    private RoleAssignmentResponse toResponse(IamRoleAssignmentRow row) {
        return RoleAssignmentResponse.builder()
                .assignmentId(row.getAssignmentId())
                .principalId(row.getPrincipalId())
                .roleId(row.getRoleId())
                .scopeType(row.getScopeType())
                .scopeId(row.getScopeId())
                .assignmentStatus(row.getAssignmentStatus())
                .assignedByPrincipalId(row.getAssignedByPrincipalId())
                .reason(row.getReason())
                .validFrom(row.getValidFrom())
                .validUntil(row.getValidUntil())
                .createdAt(row.getCreatedAt())
                .updatedAt(row.getUpdatedAt())
                .version(row.getVersion() != null ? row.getVersion() : 0L)
                .build();
    }

    /**
     * Sync permissions to Keycloak AND force re-login (logout user + remove BFF session).
     * Called on revoke/reduce operations only.
     */
    void syncAndForceRelogin(UUID principalId) {
        if (principalId == null) return;
        try {
            if (bffSessionStore != null) {
                bffSessionStore.removeSessionsByPrincipalId(principalId);
            }
            if (keycloakAdminClient == null) return;
            IdentityPrincipalRow principal = principalMapper.findById(principalId);
            if (principal == null || principal.getIdpSubject() == null) return;

            List<IamRoleAssignmentRow> activeAssignments = roleAssignmentMapper.findActiveByPrincipal(principalId, OffsetDateTime.now());
            Set<String> perms = new HashSet<>();
            if (activeAssignments != null && rolePermissionMapper != null) {
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
            keycloakAdminClient.logoutUser(principal.getIdpSubject());
        } catch (Exception ignored) {}
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
