package com.hvduong.identity.infrastructure.persistence.mybatis;

import com.hvduong.identity.application.dto.request.AssignRoleRequest;
import com.hvduong.identity.application.dto.request.CreateRoleRequest;
import com.hvduong.identity.application.dto.request.UpdateRoleRequest;
import com.hvduong.identity.application.iam.RoleAssignmentService;
import com.hvduong.identity.application.iam.RoleManagementService;
import com.hvduong.identity.domain.entity.IdentityPrincipalRow;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.IdentityPrincipalMapper;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.NoSuchElementException;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.*;

@SpringBootTest(properties = "grpc.server.port=-1")
@ActiveProfiles("test")
@Transactional
class RoleLifecycleIntegrationTest {
    @Autowired private RoleManagementService roles;
    @Autowired private RoleAssignmentService assignments;
    @Autowired private IdentityPrincipalMapper principals;
    @MockBean private KeycloakAdminClient keycloak;

    @Test
    void disableRevokesGrantsAndReenablePermitsNewAssignments() {
        var employee = new IdentityPrincipalRow();
        employee.setPrincipalId(UUID.randomUUID());
        employee.setIdpIssuer("https://issuer.example.test");
        employee.setIdpSubject(employee.getPrincipalId().toString());
        employee.setPrincipalType("EMPLOYEE");
        employee.setProvisioningStatus("ACTIVE");
        employee.setVersion(0L);
        principals.insert(employee);

        UUID roleId = roles.createRole(new CreateRoleRequest("LIFECYCLE_ROLE", "Lifecycle", "Original"), null);
        roles.replaceRolePermissions(roleId, List.of("ROLE_VIEW"), null);
        var assignment = new AssignRoleRequest();
        assignment.setPrincipalId(employee.getPrincipalId());
        assignment.setRoleId(roleId);
        assignment.setScopeType("GLOBAL");
        assignments.assignRole(assignment, null);
        verify(keycloak).updateUserPermissionsWithGrants(employee.getIdpSubject(), List.of("ROLE_VIEW#GLOBAL"), "EMPLOYEE");

        var disabled = new UpdateRoleRequest();
        disabled.setIsEnabled(false);
        disabled.setRoleName("Inactive lifecycle");
        roles.updateRole(roleId, disabled, null);
        assertThat(roles.getRoleById(roleId).getIsEnabled()).isFalse();
        assertThat(roles.getRoleById(roleId).getRoleName()).isEqualTo("Inactive lifecycle");
        assertThat(roles.getAllRoles()).anySatisfy(role -> {
            assertThat(role.getRoleId()).isEqualTo(roleId);
            assertThat(role.getIsEnabled()).isFalse();
        });
        assertThat(assignments.getActiveAssignments(employee.getPrincipalId())).isEmpty();
        verify(keycloak).updateUserPermissionsWithGrants(employee.getIdpSubject(), List.of(), "EMPLOYEE");
        verify(keycloak).logoutUserStrict(employee.getIdpSubject());
        assertThatThrownBy(() -> assignments.assignRole(assignment, null))
                .isInstanceOf(IllegalStateException.class).hasMessageContaining("disabled role");

        var enabled = new UpdateRoleRequest();
        enabled.setIsEnabled(true);
        roles.updateRole(roleId, enabled, null);
        assertThat(roles.getRoleById(roleId).getIsEnabled()).isTrue();
        assertThat(assignments.getActiveAssignments(employee.getPrincipalId())).isEmpty();
        assignments.assignRole(assignment, null);
        verify(keycloak, times(2)).updateUserPermissionsWithGrants(employee.getIdpSubject(), List.of("ROLE_VIEW#GLOBAL"), "EMPLOYEE");
    }

    @Test
    void deletesDisabledRoleWithoutAssignments() {
        UUID roleId = roles.createRole(new CreateRoleRequest("UNUSED_ROLE", "Unused", null), null);
        var disabled = new UpdateRoleRequest();
        disabled.setIsEnabled(false);
        roles.updateRole(roleId, disabled, null);
        roles.deleteRole(roleId, null);
        verify(keycloak).deleteRealmRole("UNUSED_ROLE");
        assertThatThrownBy(() -> roles.getRoleById(roleId)).isInstanceOf(NoSuchElementException.class);
    }
}
