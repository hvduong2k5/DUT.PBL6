package com.hvduong.identity.application.iam;

import com.hvduong.identity.application.iam.impl.RoleManagementServiceImpl;
import com.hvduong.identity.dto.request.CreateRoleRequest;
import com.hvduong.identity.dto.request.UpdateRoleRequest;
import com.hvduong.identity.dto.response.RoleDetailResponse;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import com.hvduong.identity.infrastructure.persistence.entity.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.OffsetDateTime;
import java.util.List;
import java.util.NoSuchElementException;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

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

    private RoleManagementService service;

    @BeforeEach
    void setUp() {
        service = new RoleManagementServiceImpl(
                roleMapper,
                rolePermissionMapper,
                roleAssignmentMapper,
                keycloakAdminClient,
                auditService
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

        when(roleMapper.findById(roleId)).thenReturn(row);
        when(roleMapper.updateDetails(eq(roleId), any(), any(), any(), any(), eq(1L))).thenReturn(1);

        UpdateRoleRequest dto = new UpdateRoleRequest();
        dto.setRoleName("Thu ngân ca sáng");
        dto.setDescription("Mô tả mới");

        service.updateRole(roleId, dto, UUID.randomUUID());

        verify(keycloakAdminClient).createOrUpdateRealmRole(eq("CASHIER"), eq("Mô tả mới"));
        verify(auditService).recordAudit(any(), any(), any(), eq("UPDATE_ROLE"), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void deleteRole_Protected_ThrowsException() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("SUPER_ADMIN");
        row.setIsProtected(true);

        when(roleMapper.findById(roleId)).thenReturn(row);

        assertThrows(IllegalStateException.class, () -> service.deleteRole(roleId, UUID.randomUUID()));
    }

    @Test
    void deleteRole_ActiveAssignmentsExist_ThrowsException() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("STAFF");
        row.setIsProtected(false);

        when(roleMapper.findById(roleId)).thenReturn(row);
        when(roleAssignmentMapper.countActiveByRoleId(eq(roleId), any())).thenReturn(5);

        assertThrows(IllegalStateException.class, () -> service.deleteRole(roleId, UUID.randomUUID()));
    }

    @Test
    void deleteRole_Success() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);
        row.setRoleCode("TEMPORARY_STAFF");
        row.setIsProtected(false);

        when(roleMapper.findById(roleId)).thenReturn(row);
        when(roleAssignmentMapper.countActiveByRoleId(eq(roleId), any())).thenReturn(0);

        service.deleteRole(roleId, UUID.randomUUID());

        verify(rolePermissionMapper).deleteByRoleId(roleId);
        verify(roleMapper).deleteById(roleId);
        verify(keycloakAdminClient).deleteRealmRole("TEMPORARY_STAFF");
        verify(auditService).recordAudit(any(), any(), any(), eq("DELETE_ROLE"), any(), any(), any(), any(), any(), any(), any());
    }

    @Test
    void replaceRolePermissions_Success() {
        UUID roleId = UUID.randomUUID();
        IamRoleRow row = new IamRoleRow();
        row.setRoleId(roleId);

        when(roleMapper.findById(roleId)).thenReturn(row);

        service.replaceRolePermissions(roleId, List.of("order:read", "order:update"), UUID.randomUUID());

        verify(rolePermissionMapper).deleteByRoleId(roleId);
        verify(rolePermissionMapper, times(2)).insert(any(IamRolePermissionRow.class));
    }
}

