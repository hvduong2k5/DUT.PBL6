package com.hvduong.identity.infrastructure.keycloak;

import org.keycloak.admin.client.Keycloak;
import org.keycloak.admin.client.resource.UsersResource;
import org.keycloak.representations.idm.CredentialRepresentation;
import org.keycloak.representations.idm.UserRepresentation;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.util.Collections;
import java.util.Optional;
import java.util.Map;
import java.util.UUID;

@Service
public class KeycloakAdminClient {

    private final Keycloak keycloak;
    private final String realm;

    public KeycloakAdminClient(Keycloak keycloak, @Value("${keycloak.realm}") String realm) {
        this.keycloak = keycloak;
        this.realm = realm;
    }

    public String createUser(UUID principalId, String email, String firstName, String lastName, String temporaryPassword) {
        if (principalId == null || principalId.equals(new UUID(0, 0))) {
            throw new IllegalArgumentException("A non-zero internal principal ID is required");
        }
        var customerRole = keycloak.realm(realm).roles().get("CUSTOMER").toRepresentation();
        UsersResource usersResource = keycloak.realm(realm).users();
        
        UserRepresentation user = new UserRepresentation();
        user.setUsername(email);
        user.setEmail(email);
        user.setFirstName(firstName);
        user.setLastName(lastName);
        user.setEnabled(true);
        user.setEmailVerified(false);
        user.setAttributes(Map.of("app_user_id", Collections.singletonList(principalId.toString())));

        CredentialRepresentation credential = new CredentialRepresentation();
        credential.setType(CredentialRepresentation.PASSWORD);
        credential.setValue(temporaryPassword);
        credential.setTemporary(true);

        user.setCredentials(Collections.singletonList(credential));

        try (var response = usersResource.create(user)) {
            if (response.getStatus() == 201) {
                String path = response.getLocation().getPath();
                String subject = path.substring(path.lastIndexOf('/') + 1);
                usersResource.get(subject).roles().realmLevel().add(Collections.singletonList(customerRole));
                return subject;
            }
            throw new RuntimeException("Failed to create user in Keycloak, status: " + response.getStatusInfo().getReasonPhrase());
        }
    }
    
    public Optional<UserRepresentation> findUserByEmail(String email) {
        return keycloak.realm(realm).users().search(email, true).stream().findFirst();
    }

    public void setUserEnabled(String keycloakUserId, boolean enabled) {
        org.keycloak.admin.client.resource.UserResource userResource = keycloak.realm(realm).users().get(keycloakUserId);
        UserRepresentation user = userResource.toRepresentation();
        user.setEnabled(enabled);
        userResource.update(user);
    }

    public void createOrUpdateRealmRole(String roleName, String description) {
        try {
            org.keycloak.admin.client.resource.RolesResource rolesResource = keycloak.realm(realm).roles();
            try {
                org.keycloak.admin.client.resource.RoleResource roleResource = rolesResource.get(roleName);
                org.keycloak.representations.idm.RoleRepresentation rep = roleResource.toRepresentation();
                rep.setDescription(description);
                roleResource.update(rep);
            } catch (jakarta.ws.rs.NotFoundException e) {
                org.keycloak.representations.idm.RoleRepresentation newRole = new org.keycloak.representations.idm.RoleRepresentation();
                newRole.setName(roleName);
                newRole.setDescription(description);
                rolesResource.create(newRole);
            }
        } catch (Exception ignored) {}
    }

    public void deleteRealmRole(String roleName) {
        try {
            keycloak.realm(realm).roles().deleteRole(roleName);
        } catch (Exception ignored) {}
    }

    public void resetUserPassword(String keycloakUserId, String newPassword, boolean temporary) {
        CredentialRepresentation credential = new CredentialRepresentation();
        credential.setType(CredentialRepresentation.PASSWORD);
        credential.setValue(newPassword);
        credential.setTemporary(temporary);
        keycloak.realm(realm).users().get(keycloakUserId).resetPassword(credential);
    }

    public void logoutUser(String keycloakUserId) {
        try {
            keycloak.realm(realm).users().get(keycloakUserId).logout();
        } catch (Exception ignored) {}
    }

    public Optional<UserRepresentation> getUserById(String keycloakUserId) {
        try {
            return Optional.ofNullable(keycloak.realm(realm).users().get(keycloakUserId).toRepresentation());
        } catch (Exception e) {
            return Optional.empty();
        }
    }
}
