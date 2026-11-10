package com.hvduong.identity.application.iam;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.List;
import java.util.NoSuchElementException;
import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.application.iam.impl.RoleManagementServiceImpl;
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

@ExtendWith(MockitoExtension.class)
class RoleManagementServiceTest {

    @Mock
    private IamRoleMapper roleMapper;

    @Mock
    private IamRolePermissionMapper rolePermissionMapper;

    @Mock
    private IamRoleAssignmentMapper roleAssignmentMapper;

    @Mock
    private KeycloakAdminClient keycloakAdminClient;

    @Mock
    private AuditService auditService;

    @Mock
    private PermissionSyncService permissionSyncService;

    private RoleManagementService service;

    @BeforeEach
    void setUp() {
        service = new RoleManagementServiceImpl(
                roleMapper,
                rolePermissionMapper,
                roleAssignmentMapper,
                keycloakAdminClient,
                auditService,
                permissionSyncService
        );
    }

    @Test
    void getAllRoles_ReturnsList() {
        IamRoleRow row = new IamRoleRow();
        row.setRoleCode("WAREHOUSE_STAFF");
        when(roleMapper.findAll()).thenReturn(List.of(row));

        List<IamRoleRow> list = service.getAllRoles();
        assertEquals(1, list.size());
        assertEquals("WAREHOUSE_STAFF", list.get(0).getRoleCode());
    }

    @Test
    void getRoleById_Found_ReturnsDetailWithPermissions() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("STORE_STAFF");
        row.setIsProtected(false);

        when(roleMapper.findById(roleId)).thenReturn(row);

        IamRolePermissionRow p = new IamRolePermissionRow();
        p.setRoleId(roleId);
        p.setPermissionCode("order:read");
        when(rolePermissionMapper.findByRoleId(roleId)).thenReturn(List.of(p));

        RoleDetailResponse detail = service.getRoleById(roleId);
        assertNotNull(detail);
        assertEquals("STORE_STAFF", detail.getRoleCode());
        assertEquals(1, detail.getPermissions().size());
        assertEquals("order:read", detail.getPermissions().get(0));
    }

    @Test
    void getRoleById_NotFound_ThrowsException() {
        UUID roleId = UUID.randomUUID();
        when(roleMapper.findById(roleId)).thenReturn(null);

        assertThrows(NoSuchElementException.class, () -> service.getRoleById(roleId));
    }

    @Test
    void createRole_Success() {
        CreateRoleRequest dto = new CreateRoleRequest();
        dto.setRoleCode("CASHIER");
        dto.setRoleName("Thu ngân");
        dto.setDescription("Nhân viên thu ngân");

        when(roleMapper.findByCode("CASHIER")).thenReturn(null);

        UUID creatorId = UUID.randomUUID();
        UUID roleId = service.createRole(dto, creatorId);

        assertNotNull(roleId);
        verify(roleMapper).insert(any(IamRoleRow.class));
        verify(keycloakAdminClient).createOrUpdateRealmRole(eq("CASHIER"), eq("Nhân viên thu ngân"));
        verify(auditService).recordAudit(any(), any(), any(), eq("CREATE_ROLE"), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void createRole_DuplicateCode_ThrowsException() {
        CreateRoleRequest dto = new CreateRoleRequest();
        dto.setRoleCode("CASHIER");

        when(roleMapper.findByCode("CASHIER")).thenReturn(new IamRoleRow());

        assertThrows(IllegalArgumentException.class, () -> service.createRole(dto, UUID.randomUUID()));
    }

    @Test
    void updateRole_Success() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("CASHIER");
        row.setVersion(1L);
        row.setIsEnabled(true);

        when(roleMapper.findByIdIncludeDisabled(roleId)).thenReturn(row);
        when(roleMapper.updateDetails(eq(roleId), any(), any(), any(), any(), eq(1L))).thenReturn(1);

        UpdateRoleRequest dto = new UpdateRoleRequest();
        dto.setRoleName("Thu ngân ca sáng");
        dto.setDescription("Mô tả mới");

        service.updateRole(roleId, dto, UUID.randomUUID());

        verify(keycloakAdminClient).createOrUpdateRealmRole(eq("CASHIER"), eq("Mô tả mới"));
        verify(auditService).recordAudit(any(), any(), any(), eq("UPDATE_ROLE"), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void updateRole_Disable_SyncsAndForcesReloginForAffectedPrincipals() {
        UUID roleId = UUID.randomUUID();
        UUID principalId = UUID.randomUUID();
        UUID actorId = UUID.randomUUID();

        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("STORE_STAFF");
        row.setIsProtected(false);
        row.setIsEnabled(true);
        row.setVersion(2L);
        row.setDescription("Nhân viên cửa hàng");

        IamRoleAssignmentRow assignment = new IamRoleAssignmentRow();
        assignment.setPrincipalId(principalId);
        assignment.setRoleId(roleId);

        when(roleMapper.findByIdIncludeDisabled(roleId)).thenReturn(row);
        when(roleAssignmentMapper.findActiveByRoleId(eq(roleId), any())).thenReturn(List.of(assignment));
        when(roleMapper.softDeleteById(eq(roleId), eq(actorId), eq(2L))).thenReturn(1);

        UpdateRoleRequest dto = new UpdateRoleRequest();
        dto.setIsEnabled(false);

        service.updateRole(roleId, dto, actorId);

        verify(roleAssignmentMapper).revokeAllByRoleId(eq(roleId), eq(actorId), any());
        verify(permissionSyncService).syncPrincipal(
                eq(principalId), eq(true), eq(PermissionSyncService.Trigger.ROLE_DISABLE), eq(actorId));
        verify(auditService).recordAudit(any(), any(), any(), eq("ROLE_DISABLE"), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void deleteRole_Protected_ThrowsException() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("ADMIN");
        row.setIsProtected(true);
        row.setIsEnabled(true);

        when(roleMapper.findByIdIncludeDisabled(roleId)).thenReturn(row);

        assertThrows(IllegalStateException.class, () -> service.deleteRole(roleId, UUID.randomUUID()));
    }

    @Test
    void deleteRole_ActiveAssignmentsExist_ThrowsException() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("STAFF");
        row.setIsProtected(false);
        row.setIsEnabled(true);

        when(roleMapper.findByIdIncludeDisabled(roleId)).thenReturn(row);
        when(roleAssignmentMapper.countActiveByRoleId(eq(roleId), any())).thenReturn(5);

        assertThrows(IllegalStateException.class, () -> service.deleteRole(roleId, UUID.randomUUID()));
    }

    @Test
    void deleteRole_Success_HardDeleteWhenNoHistory() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("TEMPORARY_STAFF");
        row.setIsProtected(false);
        row.setIsEnabled(true);
        row.setVersion(0L);

        when(roleMapper.findByIdIncludeDisabled(roleId)).thenReturn(row);
        when(roleAssignmentMapper.countActiveByRoleId(eq(roleId), any())).thenReturn(0);
        when(roleAssignmentMapper.countAllByRoleId(roleId)).thenReturn(0);

        service.deleteRole(roleId, UUID.randomUUID());

        verify(rolePermissionMapper).deleteByRoleId(roleId);
        verify(roleMapper).deleteById(roleId);
        verify(keycloakAdminClient).deleteRealmRole("TEMPORARY_STAFF");
        verify(auditService).recordAudit(any(), any(), any(), eq("DELETE_ROLE"), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void deleteRole_Success_SoftDeleteWhenHistoryExists() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("OLD_STAFF");
        row.setIsProtected(false);
        row.setIsEnabled(true);
        row.setVersion(2L);

        when(roleMapper.findByIdIncludeDisabled(roleId)).thenReturn(row);
        when(roleAssignmentMapper.countActiveByRoleId(eq(roleId), any())).thenReturn(0);
        when(roleAssignmentMapper.countAllByRoleId(roleId)).thenReturn(3);
        when(roleMapper.softDeleteById(eq(roleId), any(), eq(2L))).thenReturn(1);

        service.deleteRole(roleId, UUID.randomUUID());

        verify(roleMapper).softDeleteById(eq(roleId), any(), eq(2L));
        verify(roleAssignmentMapper).revokeAllByRoleId(eq(roleId), any(), any());
        verify(auditService).recordAudit(any(), any(), any(), eq("ROLE_DISABLE"), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void replaceRolePermissions_ReducedPermissionsForceRelogin() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setIsProtected(false);

        when(roleMapper.findById(roleId)).thenReturn(row);
        when(rolePermissionMapper.findByRoleId(roleId)).thenReturn(List.of(
                permissionRow(roleId, "order:read"),
                permissionRow(roleId, "order:delete")));

        service.replaceRolePermissions(roleId, List.of("order:read"), UUID.randomUUID());

        verify(rolePermissionMapper).deleteByRoleId(roleId);
        verify(permissionSyncService).syncAllPrincipalsForRole(
                eq(roleId), eq(true), eq(PermissionSyncService.Trigger.ROLE_PERMISSION_CHANGE), any());
    }

    @Test
    void replaceRolePermissions_AddedPermissionsDoNotForceRelogin() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setIsProtected(false);

        when(roleMapper.findById(roleId)).thenReturn(row);
        when(rolePermissionMapper.findByRoleId(roleId)).thenReturn(List.of(permissionRow(roleId, "order:read")));

        service.replaceRolePermissions(roleId, List.of("order:read", "order:update"), UUID.randomUUID());

        verify(permissionSyncService).syncAllPrincipalsForRole(
                eq(roleId), eq(false), eq(PermissionSyncService.Trigger.ROLE_PERMISSION_CHANGE), any());
    }

    @Test
    void removePermissionFromRole_ForcesRelogin() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow role = new IamRoleRow();
        role.setRoleId(roleId);
        role.setIsProtected(false);

        when(roleMapper.findById(roleId)).thenReturn(role);

        service.removePermissionFromRole(roleId, "order:delete", UUID.randomUUID());

        verify(permissionSyncService).syncAllPrincipalsForRole(
                eq(roleId), eq(true), eq(PermissionSyncService.Trigger.ROLE_PERMISSION_CHANGE), any());
    }

    @Test
    void addPermissionToRole_DoesNotForceRelogin() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow role = new IamRoleRow();
        role.setRoleId(roleId);
        role.setIsProtected(false);

        when(roleMapper.findById(roleId)).thenReturn(role);

        service.addPermissionToRole(roleId, "order:read", UUID.randomUUID());

        verify(permissionSyncService).syncAllPrincipalsForRole(
                eq(roleId), eq(false), eq(PermissionSyncService.Trigger.ROLE_PERMISSION_CHANGE), any());
        verify(permissionSyncService, never()).syncPrincipal(any(), eq(true), any(), any());
    }

    private IamRolePermissionRow permissionRow(UUID roleId, String code) {
        IamRolePermissionRow row = new IamRolePermissionRow();
        row.setRoleId(roleId);
        row.setPermissionCode(code);
        return row;
    }
}
