package com.hvduong.identity.application.iam;

import com.hvduong.identity.application.iam.impl.RoleAssignmentServiceImpl;

import com.hvduong.identity.application.dto.request.AssignRoleRequest;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.persistence.entity.IamRoleAssignmentRow;
import com.hvduong.identity.infrastructure.persistence.entity.IamRoleRow;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
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

    private RoleAssignmentService assignmentService;

    @BeforeEach
    void setUp() {
        assignmentService = new RoleAssignmentServiceImpl(roleAssignmentMapper, principalMapper, auditService);
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
        verify(auditService, times(1)).recordAudit(any(), any(), any(), any(), any(), any(), any(), any(), any(), any(), any());
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
        ReflectionTestUtils.setField(assignmentService, "roleMapper", roleMapper);

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

        when(roleAssignmentMapper.listAssignments(null, null, null, "ACTIVE", 0, 20))
                .thenReturn(List.of(row));

        var result = assignmentService.listAssignments(null, null, null, "ACTIVE", 0, 20);
        assertThat(result).hasSize(1);
        assertThat(result.get(0).getAssignmentId()).isEqualTo(row.getAssignmentId());
    }

    @Test
    void shouldForceReloginWhenRevokingRole() {
        var keycloakAdminClient = mock(com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient.class);
        var bffSessionStore = mock(com.hvduong.identity.infrastructure.session.BffSessionStore.class);
        ReflectionTestUtils.setField(assignmentService, "keycloakAdminClient", keycloakAdminClient);
        ReflectionTestUtils.setField(assignmentService, "bffSessionStore", bffSessionStore);

        UUID assignmentId = UUID.randomUUID();
        UUID principalId = UUID.randomUUID();
        UUID roleId = UUID.randomUUID();

        IamRoleAssignmentRow existing = new IamRoleAssignmentRow();
        existing.setAssignmentId(assignmentId);
        existing.setPrincipalId(principalId);
        existing.setRoleId(roleId);

        IdentityPrincipalRow principal = new IdentityPrincipalRow();
        principal.setPrincipalId(principalId);
        principal.setIdpSubject("kc-sub-123");

        when(roleAssignmentMapper.findById(assignmentId)).thenReturn(existing);
        when(roleAssignmentMapper.revokeAssignment(assignmentId, null, "Revoked", 0L)).thenReturn(1);
        when(principalMapper.findById(principalId)).thenReturn(principal);

        boolean revoked = assignmentService.revokeRole(assignmentId, null, "Revoked", 0L);

        assertThat(revoked).isTrue();
        verify(bffSessionStore).removeSessionsByPrincipalId(principalId);
        verify(keycloakAdminClient).logoutUser("kc-sub-123");
    }
}

