package com.hvduong.identity.application.iam.impl;

import com.hvduong.identity.application.iam.RoleManagementService;
import com.hvduong.identity.dto.request.CreateRoleRequest;
import com.hvduong.identity.dto.request.UpdateRoleRequest;
import com.hvduong.identity.dto.response.RoleDetailResponse;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.entity.*;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.OffsetDateTime;
import java.util.*;

@Slf4j
@Service
@RequiredArgsConstructor
public class RoleManagementServiceImpl implements RoleManagementService {

    private final IamRoleMapper roleMapper;
    private final IamRolePermissionMapper rolePermissionMapper;
    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final KeycloakAdminClient keycloakAdminClient;
    private final AuditService auditService;

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
        IamRoleRow existing = roleMapper.findById(roleId);
        if (existing == null) {
            throw new NoSuchElementException("Role not found with id: " + roleId);
        }

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
        IamRoleRow existing = roleMapper.findById(roleId);
        if (existing == null) {
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

        // Remove role permissions association first
        rolePermissionMapper.deleteByRoleId(roleId);

        // Delete role
        roleMapper.deleteById(roleId);

        // Remove from Keycloak
        keycloakAdminClient.deleteRealmRole(existing.getRoleCode());

        // Audit Log
        auditService.recordAudit(
                "USER",
                actorId,
                null,
                "DELETE_ROLE",
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

        rolePermissionMapper.deleteByRoleId(roleId);

        if (permissionCodes != null) {
            for (String code : permissionCodes) {
                if (code != null && !code.trim().isEmpty()) {
                    IamRolePermissionRow row = new IamRolePermissionRow();
                    row.setRoleId(roleId);
                    row.setPermissionCode(code.trim());
                    rolePermissionMapper.insert(row);
                }
            }
        }

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

        rolePermissionMapper.deleteByRoleIdAndPermission(roleId, permissionCode.trim());

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
}

