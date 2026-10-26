package com.hvduong.identity.application.iam;

import com.hvduong.identity.application.iam.impl.RoleAssignmentServiceImpl;

import com.hvduong.identity.api.admin.dto.AssignRoleRequest;
import com.hvduong.identity.application.audit.AuditService;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IamRoleAssignmentMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import com.hvduong.identity.infrastructure.persistence.entity.IdentityPrincipalRow;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class RoleAssignmentServiceTest {

    @Mock
    private IamRoleAssignmentMapper roleAssignmentMapper;

    @Mock
    private IdentityPrincipalMapper principalMapper;

    @Mock
    private AuditService auditService;

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
}

