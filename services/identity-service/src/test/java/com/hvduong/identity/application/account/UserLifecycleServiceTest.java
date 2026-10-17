package com.hvduong.identity.application.account;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.OffsetDateTime;
import java.util.Collections;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class UserLifecycleServiceTest {

    @Mock
    private IdentityPrincipalMapper principalMapper;

    @Mock
    private IamRoleAssignmentMapper roleAssignmentMapper;

    @Mock
    private IamRoleMapper roleMapper;

    @Mock
    private IamRolePermissionMapper rolePermissionMapper;

    @Mock
    private OutboxEventMapper outboxEventMapper;

    @Mock
    private KeycloakAdminClient keycloakAdminClient;

    @Mock
    private AuditService auditService;

    private ObjectMapper objectMapper = new ObjectMapper();

    private UserLifecycleService service;

    @BeforeEach
    void setUp() {
        service = new UserLifecycleService(
                principalMapper,
                roleAssignmentMapper,
                roleMapper,
                rolePermissionMapper,
                outboxEventMapper,
                keycloakAdminClient,
                auditService,
                objectMapper
        );
    }

    @Test
    void getProfileBySubject_Found_ReturnsProfileWithRoles() {
        String subject = "kc-subject-123";
        UUID principalId = UUID.randomUUID();

        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(principalId);
        row.setIdpSubject(subject);
        row.setPrincipalType("EMPLOYEE");
        row.setProvisioningStatus("ACTIVE");

        when(principalMapper.findBySubject(subject)).thenReturn(row);

        UUID roleId = UUID.randomUUID();
        IamRoleAssignmentRow assignment = new IamRoleAssignmentRow();
        assignment.setRoleId(roleId);
        assignment.setScopeType("STORE");
        assignment.setScopeId("STORE_01");

        when(roleAssignmentMapper.findActiveByPrincipal(eq(principalId), any(OffsetDateTime.class)))
                .thenReturn(List.of(assignment));

        IamRoleRow roleRow = new IamRoleRow();
        roleRow.setRoleId(roleId);
        roleRow.setRoleCode("STORE_MANAGER");
        roleRow.setRoleName("Quản lý cửa hàng");
        when(roleMapper.findById(roleId)).thenReturn(roleRow);

        IamRolePermissionRow permRow = new IamRolePermissionRow();
        permRow.setRoleId(roleId);
        permRow.setPermissionCode("STORE_INVENTORY_VIEW");
        when(rolePermissionMapper.findByRoleId(roleId)).thenReturn(List.of(permRow));

        Optional<UserLifecycleService.UserProfileDto> profile = service.getProfileBySubject(subject, "test@omama.vn");

        assertTrue(profile.isPresent());
        assertEquals(principalId, profile.get().getPrincipalId());
        assertEquals("EMPLOYEE", profile.get().getPrincipalType());
        assertEquals(1, profile.get().getRoles().size());
        assertEquals("STORE_MANAGER", profile.get().getRoles().get(0).getRoleCode());
        assertEquals("Quản lý cửa hàng", profile.get().getRoles().get(0).getRoleName());
        assertEquals("STORE", profile.get().getRoles().get(0).getScopeType());
        assertEquals("STORE_01", profile.get().getRoles().get(0).getScopeId());
        assertEquals(1, profile.get().getPermissions().size());
        assertEquals("STORE_INVENTORY_VIEW", profile.get().getPermissions().get(0));
    }

    @Test
    void getProfileBySubject_Customer_ReturnsDefaultCustomerRole() {
        String subject = "kc-cust-999";
        UUID principalId = UUID.randomUUID();

        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(principalId);
        row.setIdpSubject(subject);
        row.setPrincipalType("CUSTOMER");
        row.setProvisioningStatus("ACTIVE");

        when(principalMapper.findBySubject(subject)).thenReturn(row);
        when(roleAssignmentMapper.findActiveByPrincipal(eq(principalId), any(OffsetDateTime.class)))
                .thenReturn(Collections.emptyList());

        Optional<UserLifecycleService.UserProfileDto> profile = service.getProfileBySubject(subject, "cust@gmail.com");

        assertTrue(profile.isPresent());
        assertEquals("CUSTOMER", profile.get().getPrincipalType());
        assertEquals(1, profile.get().getRoles().size());
        assertEquals("CUSTOMER", profile.get().getRoles().get(0).getRoleCode());
        assertTrue(profile.get().getPermissions().isEmpty());
    }

    @Test
    void getProfileBySubject_NotFound_ReturnsEmpty() {
        when(principalMapper.findBySubject("unknown")).thenReturn(null);

        Optional<UserLifecycleService.UserProfileDto> profile = service.getProfileBySubject("unknown", "unknown@omama.vn");

        assertTrue(profile.isEmpty());
    }

    @Test
    void suspendAccount_Success_UpdatesDbKeycloakAuditAndOutbox() {
        UUID principalId = UUID.randomUUID();
        UUID actorId = UUID.randomUUID();

        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(principalId);
        row.setIdpSubject("kc-user-456");
        row.setVersion(1L);

        when(principalMapper.findById(principalId)).thenReturn(row);
        when(principalMapper.updateProvisioningStatus(principalId, "SUSPENDED", "kc-user-456", 1L)).thenReturn(1);

        service.suspendAccount(principalId, "Violation of terms", actorId);

        verify(keycloakAdminClient).setUserEnabled("kc-user-456", false);
        verify(auditService).recordAudit(
                eq("USER"),
                eq(actorId),
                isNull(),
                eq("SUSPEND_ACCOUNT"),
                eq("PRINCIPAL"),
                eq(principalId.toString()),
                eq("SUCCESS"),
                eq("Violation of terms"),
                isNull(),
                isNull(),
                anyMap()
        );
        verify(outboxEventMapper).insert(any(OutboxEventRow.class));
    }

    @Test
    void suspendAccount_NotFound_ThrowsException() {
        UUID principalId = UUID.randomUUID();
        when(principalMapper.findById(principalId)).thenReturn(null);

        assertThrows(IllegalArgumentException.class, () -> 
                service.suspendAccount(principalId, "reason", UUID.randomUUID())
        );
    }

    @Test
    void reactivateAccount_Success_UpdatesDbKeycloakAuditAndOutbox() {
        UUID principalId = UUID.randomUUID();
        UUID actorId = UUID.randomUUID();

        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(principalId);
        row.setIdpSubject("kc-user-456");
        row.setVersion(2L);

        when(principalMapper.findById(principalId)).thenReturn(row);
        when(principalMapper.updateProvisioningStatus(principalId, "ACTIVE", "kc-user-456", 2L)).thenReturn(1);

        service.reactivateAccount(principalId, actorId);

        verify(keycloakAdminClient).setUserEnabled("kc-user-456", true);
        verify(auditService).recordAudit(
                eq("USER"),
                eq(actorId),
                isNull(),
                eq("REACTIVATE_ACCOUNT"),
                eq("PRINCIPAL"),
                eq(principalId.toString()),
                eq("SUCCESS"),
                isNull(),
                isNull(),
                isNull(),
                anyMap()
        );
        verify(outboxEventMapper).insert(any(OutboxEventRow.class));
    }
}

