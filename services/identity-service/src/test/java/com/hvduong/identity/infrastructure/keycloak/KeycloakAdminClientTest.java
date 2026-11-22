package com.hvduong.identity.infrastructure.keycloak;

import com.hvduong.identity.common.constant.IdentityConstants;
import com.hvduong.identity.common.enums.PrincipalType;
import jakarta.ws.rs.core.Response;
import org.junit.jupiter.api.Test;
import org.keycloak.admin.client.Keycloak;
import org.keycloak.representations.idm.RoleRepresentation;
import org.keycloak.representations.idm.UserRepresentation;
import org.mockito.ArgumentCaptor;

import java.net.URI;
import java.util.List;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class KeycloakAdminClientTest {
    @Test
    void customerProvisioningBindsInternalPrincipalAndPrincipalTypeWithoutRole() {
        Keycloak kc = mock(Keycloak.class, RETURNS_DEEP_STUBS);
        var users = kc.realm("omama").users();
        Response response = mock(Response.class);
        when(response.getStatus()).thenReturn(201);
        when(response.getLocation()).thenReturn(URI.create("https://identity.test/users/keycloak-subject"));
        when(users.create(any(UserRepresentation.class))).thenReturn(response);
        UUID principal = UUID.randomUUID();

        String subject = new KeycloakAdminClient(kc, "omama")
                .createUser(principal, "customer@example.test", "Test", "Customer", "test-password");

        assertEquals("keycloak-subject", subject);
        ArgumentCaptor<UserRepresentation> captured = ArgumentCaptor.forClass(UserRepresentation.class);
        verify(users).create(captured.capture());
        assertEquals(List.of(principal.toString()), captured.getValue().getAttributes().get(IdentityConstants.ATTR_APP_USER_ID));
        assertEquals(List.of(PrincipalType.CUSTOMER.name()), captured.getValue().getAttributes().get(IdentityConstants.ATTR_PRINCIPAL_TYPE));
        verify(response).close();
    }

    @Test
    void missingPrincipalCannotProvisionAnAccount() {
        Keycloak kc = mock(Keycloak.class);
        var client = new KeycloakAdminClient(kc, "omama");
        assertThrows(IllegalArgumentException.class, () -> client.createUser(null, "a", "b", "c", "d"));
        assertThrows(IllegalArgumentException.class, () -> client.createUser(new UUID(0, 0), "a", "b", "c", "d"));
        verifyNoInteractions(kc);
    }

    @Test
    void nonCreatedResponseThrowsAndClosesResponse() {
        Keycloak kc = mock(Keycloak.class, RETURNS_DEEP_STUBS);
        var users = kc.realm("omama").users();
        Response response = mock(Response.class, RETURNS_DEEP_STUBS);
        when(response.getStatus()).thenReturn(409);
        when(response.getStatusInfo().getReasonPhrase()).thenReturn("Conflict");
        when(users.create(any(UserRepresentation.class))).thenReturn(response);
        assertThrows(RuntimeException.class, () -> new KeycloakAdminClient(kc, "omama")
                .createUser(UUID.randomUUID(), "a", "b", "c", "d"));
        verify(response).close();
    }

    @Test
    void employeeProvisioningBindsPrincipalTypeEmployee() {
        Keycloak kc = mock(Keycloak.class, RETURNS_DEEP_STUBS);
        var users = kc.realm("omama").users();
        Response response = mock(Response.class);
        when(response.getStatus()).thenReturn(201);
        when(response.getLocation()).thenReturn(URI.create("https://identity.test/users/kc-employee-sub"));
        when(users.create(any(UserRepresentation.class))).thenReturn(response);
        UUID principal = UUID.randomUUID();

        String subject = new KeycloakAdminClient(kc, "omama")
                .createUser(principal, "employee@example.test", "Staff", "User", "staff-password", PrincipalType.EMPLOYEE.name());

        assertEquals("kc-employee-sub", subject);
        ArgumentCaptor<UserRepresentation> captured = ArgumentCaptor.forClass(UserRepresentation.class);
        verify(users).create(captured.capture());
        assertEquals(List.of(principal.toString()), captured.getValue().getAttributes().get(IdentityConstants.ATTR_APP_USER_ID));
        assertEquals(List.of(PrincipalType.EMPLOYEE.name()), captured.getValue().getAttributes().get(IdentityConstants.ATTR_PRINCIPAL_TYPE));
        verify(response).close();
    }

    @Test
    void setUserAttributeUpdatesKeycloakUserRepresentation() {
        Keycloak kc = mock(Keycloak.class, RETURNS_DEEP_STUBS);
        var userResource = kc.realm("omama").users().get("kc-sub-attr");
        UserRepresentation user = new UserRepresentation();
        user.setId("kc-sub-attr");
        when(userResource.toRepresentation()).thenReturn(user);

        var client = new KeycloakAdminClient(kc, "omama");
        client.setUserAttribute("kc-sub-attr", IdentityConstants.ATTR_PRINCIPAL_TYPE, PrincipalType.EMPLOYEE.name());

        ArgumentCaptor<UserRepresentation> captured = ArgumentCaptor.forClass(UserRepresentation.class);
        verify(userResource).update(captured.capture());
        assertEquals(List.of(PrincipalType.EMPLOYEE.name()), captured.getValue().getAttributes().get(IdentityConstants.ATTR_PRINCIPAL_TYPE));
    }

    @Test
    void updateUserPermissionsWithGrantsSetsPrincipalType() {
        Keycloak kc = mock(Keycloak.class, RETURNS_DEEP_STUBS);
        var userResource = kc.realm("omama").users().get("kc-sub-grants");
        UserRepresentation user = new UserRepresentation();
        user.setId("kc-sub-grants");
        when(userResource.toRepresentation()).thenReturn(user);

        var client = new KeycloakAdminClient(kc, "omama");
        client.updateUserPermissionsWithGrants(
                "kc-sub-grants",
                List.of("ORDER_VIEW#GLOBAL"),
                PrincipalType.EMPLOYEE.name()
        );

        ArgumentCaptor<UserRepresentation> captured = ArgumentCaptor.forClass(UserRepresentation.class);
        verify(userResource).update(captured.capture());
        assertEquals(List.of("ORDER_VIEW#GLOBAL"), captured.getValue().getAttributes().get(IdentityConstants.ATTR_PERMISSION_GRANTS));
        assertEquals(List.of(PrincipalType.EMPLOYEE.name()), captured.getValue().getAttributes().get(IdentityConstants.ATTR_PRINCIPAL_TYPE));
        assertNull(captured.getValue().getAttributes().get(IdentityConstants.ATTR_PERMISSIONS));
    }
}
