package com.hvduong.identity.application.iam;

import com.hvduong.identity.application.iam.impl.RoleAssignmentServiceImpl;

import com.hvduong.identity.application.dto.request.AssignRoleRequest;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.domain.entity.IamRoleAssignmentRow;
import com.hvduong.identity.domain.entity.IamRoleRow;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.common.constant.IdentityConstants;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.List;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class RoleAssignmentServiceTest {

    @Mock
    private IamRoleAssignmentMapper roleAssignmentMapper;

    @Mock
    private IdentityPrincipalMapper principalMapper;

    @Mock
    private AuditService auditService;

    @Mock
    private IamRoleMapper roleMapper;

    @Mock
    private PermissionSyncService permissionSyncService;

    private RoleAssignmentService assignmentService;

    @BeforeEach
    void setUp() {
        assignmentService = new RoleAssignmentServiceImpl(
                roleAssignmentMapper, principalMapper, auditService, permissionSyncService, roleMapper);
        IamRoleRow activeRole = new IamRoleRow();
        activeRole.setIsEnabled(true);
        lenient().when(roleMapper.findById(any())).thenReturn(activeRole);
        lenient().when(roleMapper.findByIdForUpdate(any())).thenReturn(activeRole);
    }

    @Test
    void shouldAssignRoleSuccessfullyToEmployee() {
        // Arrange
        UUID principalId = UUID.randomUUID();
        UUID roleId = UUID.randomUUID();
        UUID assignedBy = UUID.randomUUID();

        IdentityPrincipalRow employee = new IdentityPrincipalRow();
        employee.setPrincipalId(principalId);
        employee.setPrincipalType("EMPLOYEE");

        when(principalMapper.findById(principalId)).thenReturn(employee);

        AssignRoleRequest req = new AssignRoleRequest();
        req.setPrincipalId(principalId);
        req.setRoleId(roleId);
        req.setScopeType("GLOBAL");
        req.setReason("Promoted to Lead");

        // Act
        UUID assignmentId = assignmentService.assignRole(req, assignedBy);

        // Assert
        assertThat(assignmentId).isNotNull();
        verify(roleAssignmentMapper, times(1)).insert(any());
        verify(permissionSyncService).syncPrincipal(
                principalId, false, PermissionSyncService.Trigger.ROLE_ASSIGN, assignedBy);
        verify(auditService, times(1)).recordAudit(any(), any(), any(), any(), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void shouldRejectDisabledRoleBeforeInsertingAssignment() {
        UUID roleId = UUID.randomUUID();
        var role = new IamRoleRow();
        role.setRoleCode("DISABLED_STAFF");
        role.setIsEnabled(false);
        when(roleMapper.findByIdForUpdate(roleId)).thenReturn(role);
        var request = new AssignRoleRequest();
        request.setRoleId(roleId);
        request.setPrincipalId(UUID.randomUUID());

        assertThatThrownBy(() -> assignmentService.assignRole(request, UUID.randomUUID()))
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("Cannot assign disabled role");
        verifyNoInteractions(roleAssignmentMapper, permissionSyncService);
    }

    @Test
    void shouldRejectAssignmentWhenPrincipalNotEmployee() {
        // Arrange
        UUID principalId = UUID.randomUUID();
        IdentityPrincipalRow customer = new IdentityPrincipalRow();
        customer.setPrincipalId(principalId);
        customer.setPrincipalType("CUSTOMER");

        when(principalMapper.findById(principalId)).thenReturn(customer);

        AssignRoleRequest req = new AssignRoleRequest();
        req.setPrincipalId(principalId);
        req.setRoleId(UUID.randomUUID());
        req.setScopeType("GLOBAL");

        // Act & Assert
        assertThatThrownBy(() -> assignmentService.assignRole(req, UUID.randomUUID()))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("Target principal must be an EMPLOYEE");

        verify(roleAssignmentMapper, never()).insert(any());
    }

    @Test
    void shouldRejectNonGlobalScopeWithoutScopeId() {
        // Arrange
        UUID principalId = UUID.randomUUID();
        IdentityPrincipalRow employee = new IdentityPrincipalRow();
        employee.setPrincipalId(principalId);
        employee.setPrincipalType("EMPLOYEE");

        when(principalMapper.findById(principalId)).thenReturn(employee);

        AssignRoleRequest req = new AssignRoleRequest();
        req.setPrincipalId(principalId);
        req.setRoleId(UUID.randomUUID());
        req.setScopeType("WAREHOUSE");
        req.setScopeId(null); // Missing scopeId for WAREHOUSE scope

        // Act & Assert
        assertThatThrownBy(() -> assignmentService.assignRole(req, UUID.randomUUID()))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("Scope ID is required for non-GLOBAL assignments");

        verify(roleAssignmentMapper, never()).insert(any());
    }

    @Test
    void shouldRejectSelfAssignment() {
        UUID actorId = UUID.randomUUID();

        AssignRoleRequest req = new AssignRoleRequest();
        req.setPrincipalId(actorId); // same as actor
        req.setRoleId(UUID.randomUUID());
        req.setScopeType("GLOBAL");

        assertThatThrownBy(() -> assignmentService.assignRole(req, actorId))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("Self-assignment is forbidden");

        verify(roleAssignmentMapper, never()).insert(any());
    }

    @Test
    void shouldRejectRevokingLastActiveAdmin() {
        UUID assignmentId = UUID.randomUUID();
        UUID adminRoleId = UUID.randomUUID();

        IamRoleAssignmentRow existing = new IamRoleAssignmentRow();
        existing.setAssignmentId(assignmentId);
        existing.setRoleId(adminRoleId);
        existing.setPrincipalId(UUID.randomUUID());

        IamRoleRow adminRole = new IamRoleRow();
        adminRole.setRoleId(adminRoleId);
        adminRole.setRoleCode("ADMIN");
        adminRole.setIsEnabled(true);

        when(roleAssignmentMapper.findById(assignmentId)).thenReturn(existing);
        when(roleMapper.findById(adminRoleId)).thenReturn(adminRole);
        when(roleAssignmentMapper.countActiveAdmins(eq("ADMIN"), any())).thenReturn(1);

        assertThatThrownBy(() -> assignmentService.revokeRole(assignmentId, UUID.randomUUID(), "Test", 0L))
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("Cannot revoke the last active ADMIN");

        verify(roleAssignmentMapper, never()).revokeAssignment(any(), any(), any(), any());
    }

    @Test
    void shouldListAssignments() {
        IamRoleAssignmentRow row = new IamRoleAssignmentRow();
        row.setAssignmentId(UUID.randomUUID());
        row.setPrincipalId(UUID.randomUUID());
        row.setRoleId(UUID.randomUUID());
        row.setScopeType("GLOBAL");
        row.setAssignmentStatus("ACTIVE");
        row.setVersion(0L);

        when(roleAssignmentMapper.listAssignments(null, null, null, null, null, "ACTIVE", 0, 20))
                .thenReturn(List.of(row));

        var result = assignmentService.listAssignments(null, null, null, "ACTIVE", 0, 20);
        assertThat(result).hasSize(1);
        assertThat(result.get(0).getAssignmentId()).isEqualTo(row.getAssignmentId());
    }

    @Test
    void shouldUpdateAssignmentSuccessfully() {
        UUID assignmentId = UUID.randomUUID();
        UUID principalId = UUID.randomUUID();
        UUID updatedBy = UUID.randomUUID();
        java.time.OffsetDateTime now = java.time.OffsetDateTime.now();

        IamRoleAssignmentRow existing = new IamRoleAssignmentRow();
        existing.setAssignmentId(assignmentId);
        existing.setPrincipalId(principalId);
        existing.setAssignmentStatus("ACTIVE");
        existing.setValidFrom(now.minusDays(1));
        existing.setVersion(0L);

        when(roleAssignmentMapper.findById(assignmentId)).thenReturn(existing);
        when(roleAssignmentMapper.updateAssignment(eq(assignmentId), any(), any(), eq("Gia hạn thêm"), eq(0L)))
                .thenReturn(1);

        com.hvduong.identity.application.dto.request.UpdateRoleAssignmentRequest req =
                new com.hvduong.identity.application.dto.request.UpdateRoleAssignmentRequest();
        req.setValidUntil(now.plusDays(30));
        req.setReason("Gia hạn thêm");
        req.setExpectedVersion(0L);

        var updated = assignmentService.updateAssignment(assignmentId, req, updatedBy);

        assertThat(updated).isNotNull();
        verify(permissionSyncService).syncPrincipal(
                principalId, false, PermissionSyncService.Trigger.ROLE_ASSIGN, updatedBy);
    }

    @Test
    void shouldSyncAndForceReloginWhenRevokingRole() {
        UUID assignmentId = UUID.randomUUID();
        UUID principalId = UUID.randomUUID();
        UUID roleId = UUID.randomUUID();
        UUID revokedBy = UUID.randomUUID();

        IamRoleAssignmentRow existing = new IamRoleAssignmentRow();
        existing.setAssignmentId(assignmentId);
        existing.setPrincipalId(principalId);
        existing.setRoleId(roleId);

        when(roleAssignmentMapper.findById(assignmentId)).thenReturn(existing);
        when(roleAssignmentMapper.revokeAssignment(assignmentId, revokedBy, "Revoked", 0L)).thenReturn(1);

        boolean revoked = assignmentService.revokeRole(assignmentId, revokedBy, "Revoked", 0L);

        assertThat(revoked).isTrue();
        verify(permissionSyncService).syncPrincipal(
                principalId, true, PermissionSyncService.Trigger.ROLE_REVOKE, revokedBy);
    }

    @Test
    void shouldRefuseToRevokeLastAdmin() {
        UUID assignmentId = UUID.randomUUID();
        UUID roleId = UUID.randomUUID();

        IamRoleAssignmentRow existing = new IamRoleAssignmentRow();
        existing.setAssignmentId(assignmentId);
        existing.setPrincipalId(UUID.randomUUID());
        existing.setRoleId(roleId);

        IamRoleRow adminRole = new IamRoleRow();
        adminRole.setRoleId(roleId);
        adminRole.setRoleCode(IdentityConstants.ROLE_ADMIN);
        adminRole.setIsEnabled(true);

        when(roleAssignmentMapper.findById(assignmentId)).thenReturn(existing);
        when(roleMapper.findById(roleId)).thenReturn(adminRole);
        when(roleAssignmentMapper.countActiveAdmins(eq(IdentityConstants.ROLE_ADMIN), any()))
                .thenReturn(1);

        assertThatThrownBy(() -> assignmentService.revokeRole(assignmentId, UUID.randomUUID(), "Revoked", 0L))
                .isInstanceOf(IllegalStateException.class);
        verify(roleAssignmentMapper, never()).revokeAssignment(any(), any(), any(), anyLong());
    }

    @Test
    void shouldLockAdminRoleRowBeforeCountingAdmins() {
        UUID assignmentId = UUID.randomUUID();
        UUID principalId = UUID.randomUUID();
        UUID roleId = UUID.randomUUID();
        UUID revokedBy = UUID.randomUUID();

        IamRoleAssignmentRow existing = new IamRoleAssignmentRow();
        existing.setAssignmentId(assignmentId);
        existing.setPrincipalId(principalId);
        existing.setRoleId(roleId);

        IamRoleRow adminRole = new IamRoleRow();
        adminRole.setRoleId(roleId);
        adminRole.setRoleCode(IdentityConstants.ROLE_ADMIN);
        adminRole.setIsEnabled(true);

        when(roleAssignmentMapper.findById(assignmentId)).thenReturn(existing);
        when(roleMapper.findById(roleId)).thenReturn(adminRole);
        when(roleAssignmentMapper.countActiveAdmins(eq(IdentityConstants.ROLE_ADMIN), any()))
                .thenReturn(2);
        when(roleAssignmentMapper.revokeAssignment(assignmentId, revokedBy, "Revoked", 0L)).thenReturn(1);

        boolean revoked = assignmentService.revokeRole(assignmentId, revokedBy, "Revoked", 0L);

        assertThat(revoked).isTrue();
        verify(roleMapper).findByIdForUpdate(roleId);
    }

    @Test
    void shouldAbortRevokeWhenPermissionSyncFails() {
        UUID assignmentId = UUID.randomUUID();
        UUID principalId = UUID.randomUUID();
        UUID roleId = UUID.randomUUID();
        UUID revokedBy = UUID.randomUUID();

        IamRoleAssignmentRow existing = new IamRoleAssignmentRow();
        existing.setAssignmentId(assignmentId);
        existing.setPrincipalId(principalId);
        existing.setRoleId(roleId);

        when(roleAssignmentMapper.findById(assignmentId)).thenReturn(existing);
        when(roleAssignmentMapper.revokeAssignment(assignmentId, revokedBy, "Revoked", 0L)).thenReturn(1);
        when(permissionSyncService.syncPrincipal(
                principalId, true, PermissionSyncService.Trigger.ROLE_REVOKE, revokedBy))
                .thenReturn(PermissionSyncService.SyncResult.failed(principalId, "KEYCLOAK_SYNC_FAILED"));

        assertThatThrownBy(() -> assignmentService.revokeRole(assignmentId, revokedBy, "Revoked", 0L))
                .isInstanceOf(IllegalStateException.class);
    }
}
