package com.hvduong.identity.application.iam;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.List;
import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.application.iam.impl.PermissionSyncServiceImpl;
import com.hvduong.identity.domain.entity.IamRoleAssignmentRow;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRolePermissionMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.session.BffSessionStore;

@ExtendWith(MockitoExtension.class)
class PermissionSyncServiceTest {

    @Mock
    private IamRoleAssignmentMapper roleAssignmentMapper;
    @Mock
    private IamRolePermissionMapper rolePermissionMapper;
    @Mock
    private IdentityPrincipalMapper principalMapper;
    @Mock
    private KeycloakAdminClient keycloakAdminClient;
    @Mock
    private AuditService auditService;
    @Mock
    private BffSessionStore bffSessionStore;

    private PermissionSyncServiceImpl service;

    @BeforeEach
    void setUp() {
        service = new PermissionSyncServiceImpl(
                roleAssignmentMapper, rolePermissionMapper, principalMapper, keycloakAdminClient, auditService);
        ReflectionTestUtils.setField(service, "bffSessionStore", bffSessionStore);
    }

    @Test
    void resolveEffectivePermissions_ReadsFromIamDatabase() {
        UUID principalId = UUID.randomUUID();
        when(rolePermissionMapper.findEffectivePermissionCodesByPrincipal(eq(principalId), any()))
                .thenReturn(List.of("order:read", "order:update"));

        List<String> permissions = service.resolveEffectivePermissions(principalId);

        assertThat(permissions).containsExactly("order:read", "order:update");
    }

    @Test
    void syncPrincipal_PushesRecomputedPermissionsWithoutRelogin() {
        UUID principalId = UUID.randomUUID();
        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpSubject("kc-sub-1");

        when(principalMapper.findById(principalId)).thenReturn(principal);
        when(rolePermissionMapper.findEffectivePermissionCodesByPrincipal(eq(principalId), any()))
                .thenReturn(List.of("order:read"));

        PermissionSyncService.SyncResult result = service.syncPrincipal(
                principalId, false, PermissionSyncService.Trigger.ROLE_ASSIGN, null);

        assertThat(result.success()).isTrue();
        assertThat(result.permissionCount()).isEqualTo(1);
        verify(keycloakAdminClient).updateUserPermissions("kc-sub-1", List.of("order:read"));
        verify(bffSessionStore, never()).removeSessionsByPrincipalId(any());
        verify(keycloakAdminClient, never()).logoutUserStrict(any());
    }

    @Test
    void syncPrincipal_ForceReloginDropsBffAndKeycloakSessions() {
        UUID principalId = UUID.randomUUID();
        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpSubject("kc-sub-2");

        when(principalMapper.findById(principalId)).thenReturn(principal);
        when(rolePermissionMapper.findEffectivePermissionCodesByPrincipal(eq(principalId), any()))
                .thenReturn(List.of());

        service.syncPrincipal(principalId, true, PermissionSyncService.Trigger.ROLE_REVOKE, null);

        verify(bffSessionStore).removeSessionsByPrincipalId(principalId);
        verify(keycloakAdminClient).logoutUserStrict("kc-sub-2");
    }

    @Test
    void syncPrincipal_KeycloakFailureIsAuditedAndNotThrown() {
        UUID principalId = UUID.randomUUID();
        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpSubject("kc-sub-3");

        when(principalMapper.findById(principalId)).thenReturn(principal);
        when(rolePermissionMapper.findEffectivePermissionCodesByPrincipal(eq(principalId), any()))
                .thenReturn(List.of("order:read"));
        org.mockito.Mockito.doThrow(new RuntimeException("keycloak down"))
                .when(keycloakAdminClient).updateUserPermissions(eq("kc-sub-3"), any());

        PermissionSyncService.SyncResult result = service.syncPrincipal(
                principalId, false, PermissionSyncService.Trigger.ROLE_PERMISSION_CHANGE, UUID.randomUUID());

        assertThat(result.success()).isFalse();
        assertThat(result.errorCode()).isEqualTo("KEYCLOAK_SYNC_FAILED");
        verify(auditService).recordAudit(
                any(), any(), any(), eq("IAM_PERMISSION_SYNC_FAILED"), eq("PRINCIPAL"),
                eq(principalId.toString()), eq("FAILED"), eq("KEYCLOAK_SYNC_FAILED"), any(), any(), any());
    }

    @Test
    void syncPrincipal_UnlinkedPrincipalReportsFailure() {
        UUID principalId = UUID.randomUUID();
        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpSubject(null);

        when(principalMapper.findById(principalId)).thenReturn(principal);

        PermissionSyncService.SyncResult result = service.syncPrincipal(
                principalId, false, PermissionSyncService.Trigger.MANUAL_RESYNC, null);

        assertThat(result.success()).isFalse();
        assertThat(result.errorCode()).isEqualTo("PRINCIPAL_NOT_LINKED");
        verify(keycloakAdminClient, never()).updateUserPermissions(any(), any());
    }

    @Test
    void syncAllPrincipalsForRole_SyncsEachDistinctHolderOnce() {
        UUID roleId = UUID.randomUUID();
        UUID p1 = UUID.randomUUID();
        UUID p2 = UUID.randomUUID();

        IamRoleAssignmentRow a1 = new IamRoleAssignmentRow();
        a1.setPrincipalId(p1);
        a1.setRoleId(roleId);
        IamRoleAssignmentRow a2 = new IamRoleAssignmentRow();
        a2.setPrincipalId(p1);
        a2.setRoleId(roleId);
        IamRoleAssignmentRow a3 = new IamRoleAssignmentRow();
        a3.setPrincipalId(p2);
        a3.setRoleId(roleId);

        when(roleAssignmentMapper.findActiveByRoleId(eq(roleId), any())).thenReturn(List.of(a1, a2, a3));

        IdentityPrincipalRow principal1 = new IdentityPrincipalRow();
        principal1.setPrincipalId(p1);
        principal1.setIdpSubject("kc-1");
        IdentityPrincipalRow principal2 = new IdentityPrincipalRow();
        principal2.setPrincipalId(p2);
        principal2.setIdpSubject("kc-2");
        when(principalMapper.findById(p1)).thenReturn(principal1);
        when(principalMapper.findById(p2)).thenReturn(principal2);
        when(rolePermissionMapper.findEffectivePermissionCodesByPrincipal(any(), any())).thenReturn(List.of());

        int synced = service.syncAllPrincipalsForRole(
                roleId, true, PermissionSyncService.Trigger.ROLE_DISABLE, null);

        assertThat(synced).isEqualTo(2);
        verify(bffSessionStore).removeSessionsByPrincipalId(p1);
        verify(bffSessionStore).removeSessionsByPrincipalId(p2);
    }
}
