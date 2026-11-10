package com.hvduong.identity.infrastructure.keycloak;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import org.keycloak.admin.client.Keycloak;
import org.keycloak.admin.client.resource.RoleResource;
import org.keycloak.admin.client.resource.RolesResource;
import org.keycloak.admin.client.resource.UserResource;
import org.keycloak.admin.client.resource.UsersResource;
import org.keycloak.representations.idm.CredentialRepresentation;
import org.keycloak.representations.idm.RoleRepresentation;
import org.keycloak.representations.idm.UserRepresentation;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.hvduong.identity.common.constant.IdentityConstants;
import com.hvduong.identity.common.enums.PrincipalType;

import jakarta.ws.rs.NotFoundException;
import lombok.extern.slf4j.Slf4j;

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
        return createUser(principalId, email, firstName, lastName, temporaryPassword, PrincipalType.CUSTOMER.name());
    }

    public String createUser(UUID principalId, String email, String firstName, String lastName, String temporaryPassword, String principalType) {
        if (principalId == null || principalId.equals(new UUID(0, 0))) {
            throw new IllegalArgumentException("A non-zero internal principal ID is required");
        }
        String effectivePrincipalType = (principalType != null && !principalType.isBlank())
                ? principalType.trim().toUpperCase()
                : PrincipalType.CUSTOMER.name();

        // Order/Profile customer authorization still requires the CUSTOMER
        // realm role in the token (realm_access.roles); the principal_type
        // attribute alone does not satisfy those consumers. Resolve the role
        // up front, mirroring dev, so a realm missing it fails before the
        // user exists. Keep assigning it until the consumers migrate
        // (7-B rollout step 3).
        RoleRepresentation customerRole = null;
        if (PrincipalType.CUSTOMER.name().equals(effectivePrincipalType)) {
            customerRole = keycloak.realm(realm).roles().get("CUSTOMER").toRepresentation();
        }

        UsersResource usersResource = keycloak.realm(realm).users();
        
        UserRepresentation user = new UserRepresentation();
        user.setUsername(email);
        user.setEmail(email);
        user.setFirstName(firstName);
        user.setLastName(lastName);
        user.setEnabled(true);
        user.setEmailVerified(false);
        user.setAttributes(Map.of(
                IdentityConstants.ATTR_APP_USER_ID, Collections.singletonList(principalId.toString()),
                IdentityConstants.ATTR_PRINCIPAL_TYPE, Collections.singletonList(effectivePrincipalType)
        ));

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
                if (customerRole != null) {
                    usersResource.get(subject).roles().realmLevel()
                            .add(Collections.singletonList(customerRole));
                }
                return subject;
            }
            throw new RuntimeException("Failed to create user in Keycloak, status: " + response.getStatusInfo().getReasonPhrase());
        }
    }
    
    public Optional<UserRepresentation> findUserByEmail(String email) {
        return keycloak.realm(realm).users().search(email, true).stream().findFirst();
    }

    public List<UserRepresentation> searchUsers(String query, int max) {
        if (query == null || query.isBlank()) {
            return Collections.emptyList();
        }
        try {
            return keycloak.realm(realm).users().search(query.trim(), 0, Math.max(10, max));
        } catch (Exception e) {
            log.warn("Failed to search users in Keycloak with query '{}'", query, e);
            return Collections.emptyList();
        }
    }

    public void setUserEnabled(String keycloakUserId, boolean enabled) {
       UserResource userResource = keycloak.realm(realm).users().get(keycloakUserId);
        UserRepresentation user = userResource.toRepresentation();
        user.setEnabled(enabled);
        userResource.update(user);
    }

    public void setUserAttribute(String keycloakUserId, String attributeName, String attributeValue) {
        if (keycloakUserId == null || keycloakUserId.isBlank() || attributeName == null || attributeName.isBlank()) {
            return;
        }
       UserResource userResource = keycloak.realm(realm).users().get(keycloakUserId);
        UserRepresentation user = userResource.toRepresentation();
        Map<String, List<String>> attributes = user.getAttributes();
        if (attributes == null) {
            attributes = new HashMap<>();
        } else {
            attributes = new HashMap<>(attributes);
        }
        if (attributeValue != null) {
            attributes.put(attributeName, Collections.singletonList(attributeValue));
        } else {
            attributes.remove(attributeName);
        }
        user.setAttributes(attributes);
        userResource.update(user);
    }

    public void setUserAttributes(String keycloakUserId, Map<String, List<String>> newAttributes) {
        if (keycloakUserId == null || keycloakUserId.isBlank() || newAttributes == null || newAttributes.isEmpty()) {
            return;
        }
       UserResource userResource = keycloak.realm(realm).users().get(keycloakUserId);
        UserRepresentation user = userResource.toRepresentation();
        Map<String, List<String>> attributes = user.getAttributes();
        if (attributes == null) {
            attributes = new HashMap<>();
        } else {
            attributes = new HashMap<>(attributes);
        }
        attributes.putAll(newAttributes);
        user.setAttributes(attributes);
        userResource.update(user);
    }

    public void createOrUpdateRealmRole(String roleName, String description) {
        try {
           RolesResource rolesResource = keycloak.realm(realm).roles();
            try {
               RoleResource roleResource = rolesResource.get(roleName);
                RoleRepresentation rep = roleResource.toRepresentation();
                rep.setDescription(description);
                roleResource.update(rep);
            } catch (NotFoundException e) {
                RoleRepresentation newRole = new RoleRepresentation();
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
     * Replace the flat {@code permissions} user attribute. Kept for consumer cũ
     * (Catalog, Identity tự kiểm) trong giai đoạn chuyển tiếp.
     */
    public void updateUserPermissions(String keycloakUserId, List<String> permissions) {
       UserResource userResource = keycloak.realm(realm).users().get(keycloakUserId);
        UserRepresentation user = userResource.toRepresentation();
        Map<String, List<String>> attributes = user.getAttributes();
        if (attributes == null) {
            attributes = new HashMap<>();
        } else {
            attributes = new HashMap<>(attributes);
        }
        attributes.put(IdentityConstants.ATTR_PERMISSIONS, permissions != null ? permissions : Collections.emptyList());
        user.setAttributes(attributes);
        userResource.update(user);
    }

    /**
     * Replace both {@code permissions} (flat, consumer cũ) and {@code permission_grants}
     * (scoped, consumer mới) user attributes. Failures propagate to the caller so
     * the sync can be audited and retried (see {@code PermissionSyncService}).
     *
     * <p>{@code permissionGrants} is a list of strings in the format
     * {@code permission#scope_type#scope_id} (scope_id may be empty), because Keycloak
     * user attributes are {@code Map<String, List<String>>}.
     */
    public void updateUserPermissionsWithGrants(
            String keycloakUserId,
            List<String> flatPermissions,
            List<String> permissionGrants
    ) {
        updateUserPermissionsWithGrants(keycloakUserId, flatPermissions, permissionGrants, null);
    }

    public void updateUserPermissionsWithGrants(
            String keycloakUserId,
            List<String> flatPermissions,
            List<String> permissionGrants,
            String principalType
    ) {
       UserResource userResource = keycloak.realm(realm).users().get(keycloakUserId);
        UserRepresentation user = userResource.toRepresentation();
        Map<String, List<String>> attributes = user.getAttributes();
        if (attributes == null) {
            attributes = new HashMap<>();
        } else {
            attributes = new HashMap<>(attributes);
        }
        attributes.remove(IdentityConstants.ATTR_PERMISSIONS);
        attributes.put(IdentityConstants.ATTR_PERMISSION_GRANTS, permissionGrants != null ? permissionGrants : Collections.emptyList());
        if (principalType != null && !principalType.isBlank()) {
            attributes.put(IdentityConstants.ATTR_PRINCIPAL_TYPE, Collections.singletonList(principalType.trim().toUpperCase()));
        }
        user.setAttributes(attributes);
        userResource.update(user);
    }
}
