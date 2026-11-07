package com.hvduong.identity.application.iam.impl;

import java.time.OffsetDateTime;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.Set;
import java.util.UUID;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.application.iam.RoleManagementService;
import com.hvduong.identity.dto.request.CreateRoleRequest;
import com.hvduong.identity.dto.request.UpdateRoleRequest;
import com.hvduong.identity.dto.response.RoleDetailResponse;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.entity.IamRoleAssignmentRow;
import com.hvduong.identity.infrastructure.persistence.entity.IamRolePermissionRow;
import com.hvduong.identity.infrastructure.persistence.entity.IamRoleRow;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRolePermissionMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.session.BffSessionStore;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
@RequiredArgsConstructor
public class RoleManagementServiceImpl implements RoleManagementService {

    private final IamRoleMapper roleMapper;
    private final IamRolePermissionMapper rolePermissionMapper;
    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final KeycloakAdminClient keycloakAdminClient;
    private final AuditService auditService;

    @Autowired(required = false)
    private IdentityPrincipalMapper principalMapper;

    @Autowired(required = false)
    private BffSessionStore bffSessionStore;

    @Override
    @Transactional(readOnly = true)
    public List<IamRoleRow> getAllRoles() {
        return roleMapper.findAll();
    }

    @Override
    @Transactional(readOnly = true)
    public RoleDetailResponse getRoleById(UUID roleId) {
        IamRoleRow role = roleMapper.findById(roleId);
        if (role == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        List<String> permissions = rolePermissionMapper.findByRoleId(roleId).stream()
                .map(IamRolePermissionRow::getPermissionCode)
                .toList();

        return RoleDetailResponse.builder()
                .roleId(role.getRoleId())
                .roleCode(role.getRoleCode())
                .roleName(role.getRoleName())
                .description(role.getDescription())
                .isProtected(role.getIsProtected())
                .isEnabled(role.getIsEnabled())
                .version(role.getVersion())
                .createdAt(role.getCreatedAt())
                .updatedAt(role.getUpdatedAt())
                .permissions(permissions)
                .build();
    }

    @Override
    @Transactional
    public UUID createRole(CreateRoleRequest dto, UUID creatorId) {
        if (dto.getRoleCode() == null || dto.getRoleCode().trim().isEmpty()) {
            throw new IllegalArgumentException("Role code cannot be empty");
        }

        String normalizedCode = dto.getRoleCode().trim().toUpperCase();
        IamRoleRow existing = roleMapper.findByCode(normalizedCode);
        if (existing != null) {
            throw new IllegalArgumentException("Role code already exists: " + normalizedCode);
        }

        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode(normalizedCode);
        row.setRoleName(dto.getRoleName() != null ? dto.getRoleName() : normalizedCode);
        row.setDescription(dto.getDescription());
        row.setIsProtected(false);
        row.setIsEnabled(true);
        row.setVersion(0L);
        row.setCreatedByPrincipalId(creatorId);

        roleMapper.insert(row);

        // Sync role to Keycloak
        keycloakAdminClient.createOrUpdateRealmRole(normalizedCode, dto.getDescription());

        // Audit Log
        auditService.recordAudit(
                "USER",
                creatorId,
                null,
                "CREATE_ROLE",
                "ROLE",
                roleId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("roleCode", normalizedCode)
        );

        return roleId;
    }

    @Override
    @Transactional
    public void updateRole(UUID roleId, UpdateRoleRequest dto, UUID updaterId) {
        // Always load including disabled so we can give meaningful 404 vs 409
        IamRoleRow existing = roleMapper.findByIdIncludeDisabled(roleId);
        if (existing == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        // Block reactivation of soft-deleted roles
        if (!existing.getIsEnabled()) {
            throw new IllegalStateException("Role is already soft-deleted and cannot be reactivated: " + existing.getRoleCode());
        }

        // Rule: Protected role cannot be disabled
        if (Boolean.TRUE.equals(existing.getIsProtected()) && dto.getIsEnabled() != null && !dto.getIsEnabled()) {
            throw new IllegalStateException("Không thể vô hiệu hóa vai trò bảo vệ hệ thống: " + existing.getRoleCode());
        }

        // If caller wants to disable → trigger soft-delete flow
        if (Boolean.FALSE.equals(dto.getIsEnabled())) {
            softDeleteRoleInternal(existing, updaterId);
            return;
        }

        // Normal update (roleName, description)
        int updated = roleMapper.updateDetails(
                roleId,
                dto.getRoleName(),
                dto.getDescription(),
                null, // isEnabled change handled above
                updaterId,
                existing.getVersion()
        );

        if (updated == 0) {
            throw new IllegalStateException("Optimistic lock conflict while updating role: " + roleId);
        }

        // Sync description change to Keycloak
        if (dto.getDescription() != null) {
            keycloakAdminClient.createOrUpdateRealmRole(existing.getRoleCode(), dto.getDescription());
        }

        // Audit Log
        auditService.recordAudit(
                "USER",
                updaterId,
                null,
                "UPDATE_ROLE",
                "ROLE",
                roleId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("roleCode", existing.getRoleCode())
        );
    }

    @Override
    @Transactional
    public void deleteRole(UUID roleId, UUID actorId) {
        IamRoleRow existing = roleMapper.findByIdIncludeDisabled(roleId);
        if (existing == null || !existing.getIsEnabled()) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        if (Boolean.TRUE.equals(existing.getIsProtected())) {
            throw new IllegalStateException("Cannot delete protected system role: " + existing.getRoleCode());
        }

        int activeAssignments = roleAssignmentMapper.countActiveByRoleId(roleId, OffsetDateTime.now());
        if (activeAssignments > 0) {
            throw new IllegalStateException("Cannot delete role " + existing.getRoleCode()
                    + " because it is currently assigned to " + activeAssignments + " active user(s)");
        }

        int totalAssignments = roleAssignmentMapper.countAllByRoleId(roleId);
        if (totalAssignments == 0) {
            // Hard delete: role has never been assigned — safe to fully remove
            rolePermissionMapper.deleteByRoleId(roleId);
            roleMapper.deleteById(roleId);
            keycloakAdminClient.deleteRealmRole(existing.getRoleCode());
            auditService.recordAudit("USER", actorId, null, "DELETE_ROLE", "ROLE",
                    roleId.toString(), "SUCCESS", null, null, null,
                    Map.of("roleCode", existing.getRoleCode(), "type", "HARD_DELETE"));
        } else {
            // Soft delete: role has historical assignments — preserve audit trail
            softDeleteRoleInternal(existing, actorId);
        }
    }

    /** Shared soft-delete logic: set is_enabled=FALSE, revoke active assignments, audit. */
    private void softDeleteRoleInternal(IamRoleRow existing, UUID actorId) {
        UUID roleId = existing.getRoleId();

        int updated = roleMapper.softDeleteById(roleId, actorId, existing.getVersion());
        if (updated == 0) {
            throw new IllegalStateException("Optimistic lock conflict while soft-deleting role: " + roleId);
        }

        // Revoke any remaining active assignments (should be 0, but as safety measure)
        roleAssignmentMapper.revokeAllByRoleId(roleId, actorId, "Role disabled");

        // Disable in Keycloak (do not delete — historical tokens may reference it)
        try {
            keycloakAdminClient.createOrUpdateRealmRole(existing.getRoleCode(), "[DISABLED] " + existing.getDescription());
        } catch (Exception ignored) {}

        auditService.recordAudit("USER", actorId, null, "ROLE_DISABLE", "ROLE",
                roleId.toString(), "SUCCESS", null, null, null,
                Map.of("roleCode", existing.getRoleCode(), "type", "SOFT_DELETE"));
    }

    @Override
    @Transactional(readOnly = true)
    public List<String> getPermissionsForRole(UUID roleId) {
        IamRoleRow existing = roleMapper.findById(roleId);
        if (existing == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        return rolePermissionMapper.findByRoleId(roleId).stream()
                .map(IamRolePermissionRow::getPermissionCode)
                .toList();
    }

    @Override
    @Transactional
    public void replaceRolePermissions(UUID roleId, List<String> permissionCodes, UUID actorId) {
        IamRoleRow existing = roleMapper.findById(roleId);
        if (existing == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        if (Boolean.TRUE.equals(existing.getIsProtected())) {
            throw new IllegalStateException("Không thể thay đổi danh sách quyền của vai trò bảo vệ hệ thống: " + existing.getRoleCode());
        }

        // Check existing permissions to detect if any permission is being revoked/reduced
        List<IamRolePermissionRow> oldRows = rolePermissionMapper.findByRoleId(roleId);
        Set<String> oldPerms = new HashSet<>();
        if (oldRows != null) {
            for (IamRolePermissionRow r : oldRows) {
                if (r.getPermissionCode() != null) {
                    oldPerms.add(r.getPermissionCode().trim());
                }
            }
        }

        Set<String> newPerms = new HashSet<>();
        rolePermissionMapper.deleteByRoleId(roleId);

        if (permissionCodes != null) {
            for (String code : permissionCodes) {
                if (code != null && !code.trim().isEmpty()) {
                    String trimmed = code.trim();
                    newPerms.add(trimmed);
                    IamRolePermissionRow row = new IamRolePermissionRow();
                    row.setRoleId(roleId);
                    row.setPermissionCode(trimmed);
                    rolePermissionMapper.insert(row);
                }
            }
        }

        boolean permissionsReduced = !newPerms.containsAll(oldPerms);
        syncActivePrincipalsForRole(roleId, permissionsReduced);

        auditService.recordAudit(
                "USER",
                actorId,
                null,
                "REPLACE_ROLE_PERMISSIONS",
                "ROLE",
                roleId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("permissionCount", permissionCodes != null ? permissionCodes.size() : 0)
        );
    }

    @Override
    @Transactional
    public void addPermissionToRole(UUID roleId, String permissionCode, UUID actorId) {
        IamRoleRow existing = roleMapper.findById(roleId);
        if (existing == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        if (Boolean.TRUE.equals(existing.getIsProtected())) {
            throw new IllegalStateException("Không thể thêm quyền vào vai trò bảo vệ hệ thống: " + existing.getRoleCode());
        }

        if (permissionCode == null || permissionCode.trim().isEmpty()) {
            throw new IllegalArgumentException("Permission code cannot be empty");
        }

        String normalizedCode = permissionCode.trim();
        IamRolePermissionRow row = new IamRolePermissionRow();
        row.setRoleId(roleId);
        row.setPermissionCode(normalizedCode);
        
        try {
            rolePermissionMapper.insert(row);
        } catch (Exception ex) {
            log.warn("Permission {} already assigned or invalid for role {}", normalizedCode, roleId);
        }

        // Adding permission does NOT force re-login; only syncs permissions to Keycloak
        syncActivePrincipalsForRole(roleId, false);

        auditService.recordAudit(
                "USER",
                actorId,
                null,
                "ADD_PERMISSION_TO_ROLE",
                "ROLE",
                roleId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("permissionCode", normalizedCode)
        );
    }

    @Override
    @Transactional
    public void removePermissionFromRole(UUID roleId, String permissionCode, UUID actorId) {
        IamRoleRow existing = roleMapper.findById(roleId);
        if (existing == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        if (Boolean.TRUE.equals(existing.getIsProtected())) {
            throw new IllegalStateException("Không thể gỡ bỏ quyền của vai trò bảo vệ hệ thống: " + existing.getRoleCode());
        }

        rolePermissionMapper.deleteByRoleIdAndPermission(roleId, permissionCode.trim());

        // Removing permission forces re-login for all active holders of this role
        syncActivePrincipalsForRole(roleId, true);

        auditService.recordAudit(
                "USER",
                actorId,
                null,
                "REMOVE_PERMISSION_FROM_ROLE",
                "ROLE",
                roleId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("permissionCode", permissionCode)
        );
    }

    /**
     * Sync active permissions for all principals currently holding {@code roleId}.
     * When {@code forceRelogin} is true, also revokes Keycloak sessions and removes BFF sessions.
     */
    private void syncActivePrincipalsForRole(UUID roleId, boolean forceRelogin) {
        if (roleAssignmentMapper == null || roleId == null) return;
        try {
            List<IamRoleAssignmentRow> activeForRole = roleAssignmentMapper.findActiveByRoleId(roleId, OffsetDateTime.now());
            if (activeForRole == null || activeForRole.isEmpty()) return;

            Set<UUID> principalIds = new HashSet<>();
            for (IamRoleAssignmentRow a : activeForRole) {
                if (a.getPrincipalId() != null) {
                    principalIds.add(a.getPrincipalId());
                }
            }

            for (UUID principalId : principalIds) {
                if (forceRelogin && bffSessionStore != null) {
                    bffSessionStore.removeSessionsByPrincipalId(principalId);
                }
                if (principalMapper != null && keycloakAdminClient != null) {
                    IdentityPrincipalRow principal = principalMapper.findById(principalId);
                    if (principal != null && principal.getIdpSubject() != null) {
                        List<IamRoleAssignmentRow> userAssignments = roleAssignmentMapper.findActiveByPrincipal(principalId, OffsetDateTime.now());
                        Set<String> perms = new HashSet<>();
                        if (userAssignments != null) {
                            for (IamRoleAssignmentRow ua : userAssignments) {
                                if (ua.getRoleId() != null) {
                                    List<IamRolePermissionRow> rolePerms = rolePermissionMapper.findByRoleId(ua.getRoleId());
                                    if (rolePerms != null) {
                                        for (IamRolePermissionRow p : rolePerms) {
                                            perms.add(p.getPermissionCode());
                                        }
                                    }
                                }
                            }
                        }
                        keycloakAdminClient.updateUserPermissions(principal.getIdpSubject(), new ArrayList<>(perms));
                        if (forceRelogin) {
                            keycloakAdminClient.logoutUser(principal.getIdpSubject());
                        }
                    }
                }
            }
        } catch (Exception ignored) {}
    }
}

