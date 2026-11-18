package com.hvduong.identity.application.iam.impl;

import java.time.OffsetDateTime;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.Set;
import java.util.UUID;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.application.iam.RoleManagementService;
import com.hvduong.identity.application.dto.request.CreateRoleRequest;
import com.hvduong.identity.application.dto.request.UpdateRoleRequest;
import com.hvduong.identity.application.dto.response.RoleDetailResponse;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.domain.entity.IamRoleAssignmentRow;
import com.hvduong.identity.domain.entity.IamRolePermissionRow;
import com.hvduong.identity.domain.entity.IamRoleRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRolePermissionMapper;

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
    private final PermissionSyncService permissionSyncService;

    @Override
    @Transactional(readOnly = true)
    public List<IamRoleRow> getAllRoles() {
        return roleMapper.findAll();
    }

    @Override
    @Transactional(readOnly = true)
    public RoleDetailResponse getRoleById(UUID roleId) {
        IamRoleRow role = roleMapper.findUnarchivedById(roleId);
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
                "EMPLOYEE",
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
        IamRoleRow existing = roleMapper.findByIdForUpdate(roleId);
        if (existing == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        if (existing.getArchivedAt() != null) {
            throw new IllegalStateException("Role is already soft-deleted and cannot be reactivated: " + existing.getRoleCode());
        }

        // Rule: Protected role cannot be disabled
        if (Boolean.TRUE.equals(existing.getIsProtected()) && dto.getIsEnabled() != null && !dto.getIsEnabled()) {
            throw new IllegalStateException("Không thể vô hiệu hóa vai trò bảo vệ hệ thống: " + existing.getRoleCode());
        }

        boolean disabling = Boolean.TRUE.equals(existing.getIsEnabled()) && Boolean.FALSE.equals(dto.getIsEnabled());
        boolean enabling = Boolean.FALSE.equals(existing.getIsEnabled()) && Boolean.TRUE.equals(dto.getIsEnabled());
        Set<UUID> affected = disabling ? collectActivePrincipalIds(roleId) : Set.of();

        int updated = roleMapper.updateDetails(
                roleId,
                dto.getRoleName(),
                dto.getDescription(),
                dto.getIsEnabled(),
                updaterId,
                existing.getVersion()
        );

        if (updated == 0) {
            throw new IllegalStateException("Optimistic lock conflict while updating role: " + roleId);
        }

        String description = dto.getDescription() != null ? dto.getDescription() : existing.getDescription();
        if (disabling) {
            roleAssignmentMapper.revokeAllByRoleId(roleId, updaterId, "Role disabled");
            for (UUID principalId : affected) {
                permissionSyncService.syncPrincipal(
                        principalId, true, PermissionSyncService.Trigger.ROLE_DISABLE, updaterId);
            }
        }
        if (disabling || enabling || dto.getDescription() != null) {
            boolean enabled = dto.getIsEnabled() != null ? dto.getIsEnabled() : existing.getIsEnabled();
            keycloakAdminClient.createOrUpdateRealmRole(existing.getRoleCode(),
                    enabled ? description : disabledDescription(existing.getRoleCode(), description));
        }

        // Audit Log
        auditService.recordAudit(
                "EMPLOYEE",
                updaterId,
                null,
                disabling ? "ROLE_DISABLE" : "UPDATE_ROLE",
                "ROLE",
                roleId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of("roleCode", existing.getRoleCode(), "affectedPrincipals", affected.size())
        );
    }

    @Override
    @Transactional
    public void deleteRole(UUID roleId, UUID actorId) {
        IamRoleRow existing = roleMapper.findByIdForUpdate(roleId);
        if (existing == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

        if (Boolean.TRUE.equals(existing.getIsProtected())) {
            throw new IllegalStateException("Cannot delete protected system role: " + existing.getRoleCode());
        }

        if (existing.getArchivedAt() != null) {
            throw new IllegalStateException("Role is already archived: " + existing.getRoleCode());
        }

        int activeAssignments = roleAssignmentMapper.countActiveByRoleId(roleId, OffsetDateTime.now());
        if (activeAssignments > 0) {
            throw new IllegalStateException("Cannot delete role " + existing.getRoleCode()
                    + " because it is currently assigned to " + activeAssignments + " active user(s)");
        }

        int pendingAssignments = roleAssignmentMapper.countPendingByRoleId(roleId, OffsetDateTime.now());
        if (pendingAssignments > 0) {
            throw new IllegalStateException("Cannot delete role " + existing.getRoleCode()
                    + " because it has " + pendingAssignments + " assignment(s) scheduled in the future");
        }

        int totalAssignments = roleAssignmentMapper.countAllByRoleId(roleId);
        if (totalAssignments == 0) {
            // Hard delete: role has never been assigned — safe to fully remove.
            // deleteRealmRole propagates failures, so a Keycloak outage rolls
            // the whole delete back instead of leaving an orphan realm role.
            rolePermissionMapper.deleteByRoleId(roleId);
            roleMapper.deleteById(roleId);
            keycloakAdminClient.deleteRealmRole(existing.getRoleCode());
            auditService.recordAudit("EMPLOYEE", actorId, null, "DELETE_ROLE", "ROLE",
                    roleId.toString(), "SUCCESS", null, null, null,
                    Map.of("roleCode", existing.getRoleCode(), "type", "HARD_DELETE"));
            return;
        }

        softDeleteRoleInternal(existing, actorId);
    }

    private void softDeleteRoleInternal(IamRoleRow existing, UUID actorId) {
        UUID roleId = existing.getRoleId();

        int updated = roleMapper.softDeleteById(roleId, actorId, existing.getVersion());
        if (updated == 0) {
            throw new IllegalStateException("Optimistic lock conflict while soft-deleting role: " + roleId);
        }

        // DELETE only reaches this point when no active assignment exists; this also
        // cleans up any expired rows that were still flagged ACTIVE.
        roleAssignmentMapper.revokeAllByRoleId(roleId, actorId, "Role disabled");

        // Do not delete the realm role: historical tokens/attributes may still reference it.
        keycloakAdminClient.createOrUpdateRealmRole(
                existing.getRoleCode(), disabledDescription(existing.getRoleCode(), existing.getDescription()));

        auditService.recordAudit("EMPLOYEE", actorId, null, "ROLE_DISABLE", "ROLE",
                roleId.toString(), "SUCCESS", null, null, null,
                Map.of("roleCode", existing.getRoleCode(), "type", "SOFT_DELETE"));
    }

    private String disabledDescription(String roleCode, String description) {
        return "[DISABLED] " + (description != null ? description : roleCode);
    }

    private Set<UUID> collectActivePrincipalIds(UUID roleId) {
        List<IamRoleAssignmentRow> active = roleAssignmentMapper.findActiveByRoleId(roleId, OffsetDateTime.now());
        Set<UUID> principalIds = new HashSet<>();
        if (active == null) {
            return principalIds;
        }
        for (IamRoleAssignmentRow assignment : active) {
            if (assignment.getPrincipalId() != null) {
                principalIds.add(assignment.getPrincipalId());
            }
        }
        return principalIds;
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
        permissionSyncService.syncAllPrincipalsForRole(
                roleId, permissionsReduced, PermissionSyncService.Trigger.ROLE_PERMISSION_CHANGE, actorId);

        auditService.recordAudit(
                "EMPLOYEE",
                actorId,
                null,
                "REPLACE_ROLE_PERMISSIONS",
                "ROLE",
                roleId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of(
                        "permissionCount", permissionCodes != null ? permissionCodes.size() : 0,
                        "permissionsReduced", permissionsReduced
                )
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
        permissionSyncService.syncAllPrincipalsForRole(
                roleId, false, PermissionSyncService.Trigger.ROLE_PERMISSION_CHANGE, actorId);

        auditService.recordAudit(
                "EMPLOYEE",
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
        permissionSyncService.syncAllPrincipalsForRole(
                roleId, true, PermissionSyncService.Trigger.ROLE_PERMISSION_CHANGE, actorId);

        auditService.recordAudit(
                "EMPLOYEE",
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
}
