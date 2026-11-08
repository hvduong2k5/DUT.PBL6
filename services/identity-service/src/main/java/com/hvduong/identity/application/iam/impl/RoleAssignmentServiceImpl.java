package com.hvduong.identity.application.iam.impl;

import java.time.OffsetDateTime;
import java.util.Collections;
import java.util.ConcurrentModificationException;
import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.UUID;
import java.util.stream.Collectors;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.application.dto.request.AssignRoleRequest;
import com.hvduong.identity.application.dto.request.UpdateRoleAssignmentRequest;
import com.hvduong.identity.application.dto.response.RoleAssignmentResponse;
import com.hvduong.identity.application.iam.PermissionSyncService;
import com.hvduong.identity.application.iam.RoleAssignmentService;
import com.hvduong.identity.common.constant.IdentityConstants;
import com.hvduong.identity.common.enums.AssignmentStatus;
import com.hvduong.identity.common.enums.PrincipalType;
import com.hvduong.identity.domain.entity.IamRoleAssignmentRow;
import com.hvduong.identity.domain.entity.IamRoleRow;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;

import lombok.RequiredArgsConstructor;

@Service
@RequiredArgsConstructor
public class RoleAssignmentServiceImpl implements RoleAssignmentService {

    private static final String ADMIN_ROLE_CODE = IdentityConstants.ROLE_ADMIN;

    private final IamRoleAssignmentMapper roleAssignmentMapper;
    private final IdentityPrincipalMapper principalMapper;
    private final AuditService auditService;
    private final PermissionSyncService permissionSyncService;
    private final IamRoleMapper roleMapper;

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
        if (principal == null || !PrincipalType.EMPLOYEE.matches(principal.getPrincipalType())) {
            throw new IllegalArgumentException("Target principal must be an EMPLOYEE");
        }

        // 2. Validate scope
        if (!IdentityConstants.SCOPE_GLOBAL.equalsIgnoreCase(request.getScopeType()) && (request.getScopeId() == null || request.getScopeId().trim().isEmpty())) {
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
        assignment.setAssignmentStatus(AssignmentStatus.ACTIVE.name());
        assignment.setValidFrom(request.getValidFrom() != null ? request.getValidFrom() : OffsetDateTime.now());
        assignment.setValidUntil(request.getValidUntil());
        assignment.setAssignedByPrincipalId(assignedBy);
        assignment.setReason(request.getReason());
        assignment.setVersion(0L);

        roleAssignmentMapper.insert(assignment);

        // Project the recomputed grants into Keycloak. Adding a grant does not force re-login.
        permissionSyncService.syncPrincipal(
                request.getPrincipalId(), false, PermissionSyncService.Trigger.ROLE_ASSIGN, assignedBy);

        // 4. Audit log
        auditService.recordAudit(
                PrincipalType.EMPLOYEE.name(),
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
            // Revoking a grant reduces privileges: sync and force re-login so the next
            // token cannot carry the revoked permission.
            permissionSyncService.syncPrincipal(
                    existing.getPrincipalId(), true, PermissionSyncService.Trigger.ROLE_REVOKE, revokedBy);

            auditService.recordAudit(
                    PrincipalType.EMPLOYEE.name(),
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
    @Transactional
    public RoleAssignmentResponse updateAssignment(UUID assignmentId, UpdateRoleAssignmentRequest request, UUID updatedBy) {
        IamRoleAssignmentRow existing = roleAssignmentMapper.findById(assignmentId);
        if (existing == null) {
            throw new NoSuchElementException("Không tìm thấy bản ghi gán vai trò: " + assignmentId);
        }

        if (!"ACTIVE".equals(existing.getAssignmentStatus())) {
            throw new IllegalStateException("Chỉ có thể cập nhật bản ghi gán vai trò đang ở trạng thái ACTIVE");
        }

        if (request != null && request.getValidUntil() != null) {
            if (existing.getValidFrom() != null && request.getValidUntil().isBefore(existing.getValidFrom())) {
                throw new IllegalArgumentException("validUntil không thể diễn ra trước validFrom");
            }
        }

        Long expectedVersion = request != null ? request.getExpectedVersion() : null;
        OffsetDateTime validUntil = request != null ? request.getValidUntil() : null;
        Boolean clearValidUntil = request != null ? request.getClearValidUntil() : null;
        String reason = request != null ? request.getReason() : null;

        int updated = roleAssignmentMapper.updateAssignment(
                assignmentId, validUntil, clearValidUntil, reason, expectedVersion
        );
        if (updated == 0) {
            throw new ConcurrentModificationException("Cập nhật bản ghi gán vai trò thất bại do xung đột phiên bản (Optimistic Locking)");
        }

        // Recomputed grants sync into Keycloak
        permissionSyncService.syncPrincipal(
                existing.getPrincipalId(), false, PermissionSyncService.Trigger.ROLE_ASSIGN, updatedBy
        );

        auditService.recordAudit(
                PrincipalType.EMPLOYEE.name(),
                updatedBy,
                null,
                "IAM_ROLE_UPDATE",
                "IAM_ASSIGNMENT",
                assignmentId.toString(),
                "SUCCESS",
                null,
                null,
                null,
                Map.of(
                        "assignmentId", assignmentId.toString(),
                        "reason", reason != null ? reason : ""
                )
        );

        IamRoleAssignmentRow refreshed = roleAssignmentMapper.findById(assignmentId);
        return toResponse(refreshed);
    }

    @Override
    @Transactional(readOnly = true)
    public List<IamRoleAssignmentRow> getActiveAssignments(UUID principalId) {
        return roleAssignmentMapper.findActiveByPrincipal(principalId, OffsetDateTime.now());
    }

    @Override
    @Transactional(readOnly = true)
    public List<RoleAssignmentResponse> listAssignments(
            UUID principalId,
            UUID roleId,
            String roleCode,
            String scopeType,
            String scopeId,
            String status,
            int page,
            int size
    ) {
        int validPage = Math.max(0, page);
        int validSize = size <= 0 ? 20 : Math.min(size, 100);
        int offset = validPage * validSize;
        List<IamRoleAssignmentRow> rows = roleAssignmentMapper.listAssignments(
                principalId, roleId, roleCode, scopeType, scopeId, status, offset, validSize
        );
        if (rows == null) return Collections.emptyList();
        return rows.stream().map(this::toResponse).collect(Collectors.toList());
    }

    private RoleAssignmentResponse toResponse(IamRoleAssignmentRow row) {
        String roleCode = null;
        String roleName = null;
        if (roleMapper != null && row.getRoleId() != null) {
            IamRoleRow role = roleMapper.findById(row.getRoleId());
            if (role != null) {
                roleCode = role.getRoleCode();
                roleName = role.getRoleName();
            }
        }
        return RoleAssignmentResponse.builder()
                .assignmentId(row.getAssignmentId())
                .principalId(row.getPrincipalId())
                .roleId(row.getRoleId())
                .roleCode(roleCode)
                .roleName(roleName)
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
}
