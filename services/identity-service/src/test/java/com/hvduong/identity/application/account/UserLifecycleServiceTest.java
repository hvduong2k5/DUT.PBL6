package com.hvduong.identity.application.account;

import com.hvduong.identity.application.account.impl.UserLifecycleServiceImpl;
import com.hvduong.identity.application.dto.response.UserProfileResponse;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.*;
import com.hvduong.identity.domain.entity.*;
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

    @Mock
    private org.springframework.web.client.RestTemplate restTemplate;

    @Mock
    private com.hvduong.identity.infrastructure.session.BffSessionStore bffSessionStore;

    @Mock
    private com.hvduong.identity.application.iam.PermissionSyncService permissionSyncService;

    private ObjectMapper objectMapper = new ObjectMapper();

    private UserLifecycleService service;

    @BeforeEach
    void setUp() {
        service = new UserLifecycleServiceImpl(
                principalMapper,
                roleAssignmentMapper,
                roleMapper,
                rolePermissionMapper,
                outboxEventMapper,
                keycloakAdminClient,
                auditService,
                objectMapper,
                restTemplate,
                bffSessionStore,
                permissionSyncService
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

        when(principalMapper.findByIssuerAndSubject("https://keycloak.omama.vn/realms/omama", subject)).thenReturn(row);

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
        roleRow.setIsEnabled(true);
        roleRow.setRoleName("Quản lý cửa hàng");
        when(roleMapper.findById(roleId)).thenReturn(roleRow);

        IamRolePermissionRow permRow = new IamRolePermissionRow();
        permRow.setRoleId(roleId);
        permRow.setPermissionCode("STORE_INVENTORY_VIEW");
        when(rolePermissionMapper.findByRoleId(roleId)).thenReturn(List.of(permRow));

        Optional<UserProfileResponse> profile = service.getProfileBySubject("https://keycloak.omama.vn/realms/omama", subject, "test@omama.vn");

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
        var grant = profile.get().getPermissionGrants().get(0);
        assertEquals("STORE_INVENTORY_VIEW", grant.permissionCode());
        assertEquals("STORE", grant.scopeType());
        assertEquals("STORE_01", grant.scopeId());
    }

    @Test
    void getProfileBySubject_Customer_ReturnsEmptyRoles() {
        String subject = "kc-cust-999";
        UUID principalId = UUID.randomUUID();

        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(principalId);
        row.setIdpSubject(subject);
        row.setPrincipalType("CUSTOMER");
        row.setProvisioningStatus("ACTIVE");

        when(principalMapper.findByIssuerAndSubject("https://keycloak.omama.vn/realms/omama", subject)).thenReturn(row);
        when(roleAssignmentMapper.findActiveByPrincipal(eq(principalId), any(OffsetDateTime.class)))
                .thenReturn(Collections.emptyList());

        Optional<UserProfileResponse> profile = service.getProfileBySubject("https://keycloak.omama.vn/realms/omama", subject, "cust@gmail.com");

        assertTrue(profile.isPresent());
        assertEquals("CUSTOMER", profile.get().getPrincipalType());
        assertTrue(profile.get().getRoles().isEmpty());
        assertTrue(profile.get().getPermissions().isEmpty());
    }

    @Test
    void currentProfileKeepsPermissionScopesBoundToEnabledAssignments() {
        var principal = new IdentityPrincipalRow();
        principal.setPrincipalId(UUID.randomUUID());
        principal.setIdpSubject("staff");
        principal.setPrincipalType("EMPLOYEE");
        principal.setProvisioningStatus("ACTIVE");
        when(principalMapper.findByIssuerAndSubject("issuer", "staff")).thenReturn(principal);
        var viewRole = new IamRoleRow();
        viewRole.setRoleId(UUID.randomUUID());
        viewRole.setRoleCode("VIEW_A");
        viewRole.setIsEnabled(true);
        var sensitiveRole = new IamRoleRow();
        sensitiveRole.setRoleId(UUID.randomUUID());
        sensitiveRole.setRoleCode("SENSITIVE_B");
        sensitiveRole.setIsEnabled(true);
        var disabled = new IamRoleRow();
        disabled.setRoleId(UUID.randomUUID());
        disabled.setIsEnabled(false);
        var assignments = new java.util.ArrayList<IamRoleAssignmentRow>();
        for (var role : List.of(viewRole, sensitiveRole, disabled)) {
            var assignment = new IamRoleAssignmentRow();
            assignment.setRoleId(role.getRoleId());
            assignment.setScopeType("WAREHOUSE");
            assignment.setScopeId(role == viewRole ? "A" : "B");
            assignments.add(assignment);
            when(roleMapper.findById(role.getRoleId())).thenReturn(role);
        }
        when(roleAssignmentMapper.findActiveByPrincipal(eq(principal.getPrincipalId()), any())).thenReturn(assignments);
        var view = new IamRolePermissionRow(); view.setPermissionCode("ORDER_VIEW");
        var sensitive = new IamRolePermissionRow(); sensitive.setPermissionCode("ORDER_SENSITIVE_VIEW");
        when(rolePermissionMapper.findByRoleId(viewRole.getRoleId())).thenReturn(List.of(view));
        when(rolePermissionMapper.findByRoleId(sensitiveRole.getRoleId())).thenReturn(List.of(sensitive));
        var profile = service.getProfileBySubject("issuer", "staff", "staff@example.test").orElseThrow();
        assertEquals(List.of(
            new com.hvduong.identity.application.dto.response.PermissionGrantResponse("ORDER_VIEW", "WAREHOUSE", "A"),
            new com.hvduong.identity.application.dto.response.PermissionGrantResponse("ORDER_SENSITIVE_VIEW", "WAREHOUSE", "B")
        ), profile.getPermissionGrants());
        assertEquals(2, profile.getRoles().size());
        verify(rolePermissionMapper, never()).findByRoleId(disabled.getRoleId());
    }

    @Test
    void getProfileBySubject_NotFound_ReturnsEmpty() {
        when(principalMapper.findByIssuerAndSubject("https://keycloak.omama.vn/realms/omama", "unknown")).thenReturn(null);

        Optional<UserProfileResponse> profile = service.getProfileBySubject("https://keycloak.omama.vn/realms/omama", "unknown", "unknown@omama.vn");

        assertTrue(profile.isEmpty());
    }

    @Test
    void getProfileBySubject_OpenidOnly_ResolvesEmailFromBoundKeycloakAccount() {
        String issuer = "https://keycloak.omama.vn/realms/omama";
        String subject = "verified-subject";
        IdentityPrincipalRow row = new IdentityPrincipalRow();
        row.setPrincipalId(UUID.randomUUID());
        row.setIdpSubject(subject);
        row.setPrincipalType("CUSTOMER");
        row.setProvisioningStatus("ACTIVE");
        when(principalMapper.findByIssuerAndSubject(issuer, subject)).thenReturn(row);
        var account = new org.keycloak.representations.idm.UserRepresentation();
        account.setEmail("customer@example.test");
        when(keycloakAdminClient.getUserById(subject)).thenReturn(Optional.of(account));

        var profile = service.getProfileBySubject(issuer, subject, null).orElseThrow();

        assertEquals("customer@example.test", profile.getEmail());
        assertEquals(row.getPrincipalId(), profile.getPrincipalId());
        assertTrue(profile.getRoles().isEmpty());
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

