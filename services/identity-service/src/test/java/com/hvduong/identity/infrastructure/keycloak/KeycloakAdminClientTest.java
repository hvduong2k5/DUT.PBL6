package com.hvduong.identity.infrastructure.keycloak;

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
    void customerProvisioningBindsInternalPrincipalAndRole() {
        Keycloak kc = mock(Keycloak.class, RETURNS_DEEP_STUBS);
        var users = kc.realm("omama").users();
        RoleRepresentation role = new RoleRepresentation();
        role.setName("CUSTOMER");
        when(kc.realm("omama").roles().get("CUSTOMER").toRepresentation()).thenReturn(role);
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
        assertEquals(List.of(principal.toString()), captured.getValue().getAttributes().get("app_user_id"));
        verify(users.get(subject).roles().realmLevel()).add(List.of(role));
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
    void roleAssignmentFailureDoesNotReportSuccessfulProvisioning() {
        Keycloak kc = mock(Keycloak.class, RETURNS_DEEP_STUBS);
        var users = kc.realm("omama").users();
        Response response = mock(Response.class);
        when(response.getStatus()).thenReturn(201);
        when(response.getLocation()).thenReturn(URI.create("https://identity.test/users/subject"));
        when(users.create(any(UserRepresentation.class))).thenReturn(response);
        var realmMappings = users.get("subject").roles().realmLevel();
        doThrow(new RuntimeException("role mapping unavailable"))
                .when(realmMappings).add(anyList());
        assertThrows(RuntimeException.class, () -> new KeycloakAdminClient(kc, "omama")
                .createUser(UUID.randomUUID(), "a", "b", "c", "d"));
        verify(response).close();
    }
}
