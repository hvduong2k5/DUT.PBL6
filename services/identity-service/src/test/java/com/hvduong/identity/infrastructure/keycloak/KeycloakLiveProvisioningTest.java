package com.hvduong.identity.infrastructure.keycloak;

import org.junit.jupiter.api.Test;
import org.keycloak.OAuth2Constants;
import org.keycloak.admin.client.KeycloakBuilder;
import java.util.UUID;
import static org.junit.jupiter.api.Assertions.*;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

/** Opt-in against a disposable local realm, never the shared deployment. */
class KeycloakLiveProvisioningTest {
    @Test
    void provisionCustomersAgainstRealKeycloak() {
        String base = System.getenv("KEYCLOAK_TEST_URL");
        assumeTrue(base != null, "Real Keycloak fixture not configured");
        assertTrue(base.startsWith("http://127.0.0.1:"));
        try (var kc = KeycloakBuilder.builder().serverUrl(base).realm("master")
                .clientId("admin-cli").grantType(OAuth2Constants.PASSWORD)
                .username("admin").password(System.getenv("KEYCLOAK_TEST_ADMIN_PASSWORD")).build()) {
            String realm = System.getenv("KEYCLOAK_TEST_REALM");
            assertTrue(realm.startsWith("order-auth-test-"));
            var adapter = new KeycloakAdminClient(kc, realm);
            for (String suffix : new String[]{"A", "B"}) {
                UUID principal = UUID.fromString(System.getenv("KEYCLOAK_TEST_PRINCIPAL_" + suffix));
                String subject = adapter.createUser(principal, System.getenv("KEYCLOAK_TEST_EMAIL_" + suffix),
                        "Integration", suffix, System.getenv("KEYCLOAK_TEST_CUSTOMER_PASSWORD"));
                var resource = kc.realm(realm).users().get(subject);
                assertEquals(principal.toString(), resource.toRepresentation().getAttributes().get("app_user_id").getFirst());
                assertTrue(resource.roles().realmLevel().listAll().stream().anyMatch(r -> "CUSTOMER".equals(r.getName())));
                assertNotEquals(principal.toString(), subject);
                // Test-only account setup: no email provider or password-change UI in this slice.
                var user = resource.toRepresentation();
                user.setEmailVerified(true);
                user.setRequiredActions(java.util.List.of());
                resource.update(user);
                var credential = new org.keycloak.representations.idm.CredentialRepresentation();
                credential.setType("password");
                credential.setValue(System.getenv("KEYCLOAK_TEST_CUSTOMER_PASSWORD"));
                credential.setTemporary(false);
                resource.resetPassword(credential);
            }
        }
    }
}
