package com.hvduong.identity.infrastructure.keycloak;

import org.keycloak.admin.client.Keycloak;
import org.keycloak.admin.client.resource.UsersResource;
import org.keycloak.representations.idm.CredentialRepresentation;
import org.keycloak.representations.idm.UserRepresentation;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import lombok.extern.slf4j.Slf4j;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

@Slf4j
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
        // Self-registration chooses the permanent password; invitation flows
        // must use a separate temporary-credential operation.
        credential.setTemporary(false);

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
        } catch (Exception ex) {
            log.error("Failed to create/update Keycloak realm role '{}'", roleName, ex);
        }
    }

    public void deleteRealmRole(String roleName) {
        try {
            keycloak.realm(realm).roles().deleteRole(roleName);
        } catch (Exception ex) {
            log.error("Failed to delete Keycloak realm role '{}'", roleName, ex);
        }
    }

    public void resetUserPassword(String keycloakUserId, String newPassword, boolean temporary) {
        CredentialRepresentation credential = new CredentialRepresentation();
        credential.setType(CredentialRepresentation.PASSWORD);
        credential.setValue(newPassword);
        credential.setTemporary(temporary);
        keycloak.realm(realm).users().get(keycloakUserId).resetPassword(credential);
    }

    /**
     * Best-effort logout. Prefer {@link #logoutUserStrict(String)} on revocation paths
     * where a failure must be visible to the caller.
     */
    public void logoutUser(String keycloakUserId) {
        try {
            keycloak.realm(realm).users().get(keycloakUserId).logout();
        } catch (Exception ex) {
            log.warn("Failed to log out Keycloak user '{}'", keycloakUserId, ex);
        }
    }

    /**
     * Logout that propagates failures so callers can audit/retry them.
     */
    public void logoutUserStrict(String keycloakUserId) {
        keycloak.realm(realm).users().get(keycloakUserId).logout();
    }

    public Optional<UserRepresentation> getUserById(String keycloakUserId) {
        try {
            return Optional.ofNullable(keycloak.realm(realm).users().get(keycloakUserId).toRepresentation());
        } catch (Exception e) {
            return Optional.empty();
        }
    }

    /**
     * Replace the {@code permissions} user attribute. Failures propagate to the caller so
     * the sync can be audited and retried (see {@code PermissionSyncService}).
     */
    public void updateUserPermissions(String keycloakUserId, List<String> permissions) {
        org.keycloak.admin.client.resource.UserResource userResource = keycloak.realm(realm).users().get(keycloakUserId);
        UserRepresentation user = userResource.toRepresentation();
        Map<String, List<String>> attributes = user.getAttributes();
        if (attributes == null) {
            attributes = new HashMap<>();
        } else {
            attributes = new HashMap<>(attributes);
        }
        attributes.put("permissions", permissions != null ? permissions : Collections.emptyList());
        user.setAttributes(attributes);
        userResource.update(user);
    }
}
