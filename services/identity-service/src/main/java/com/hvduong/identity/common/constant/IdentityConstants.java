package com.hvduong.identity.common.constant;

/**
 * Shared non-enum constants for Identity Service (attributes, claims, headers).
 * Domain states and lifecycles must use their respective enums:
 * {@link com.hvduong.identity.common.enums.PrincipalType},
 * {@link com.hvduong.identity.common.enums.ProvisioningStatus},
 * {@link com.hvduong.identity.common.enums.AssignmentStatus},
 * {@link com.hvduong.identity.common.enums.InvitationStatus},
 * {@link com.hvduong.identity.common.enums.ScopeType}.
 */
public final class IdentityConstants {

    private IdentityConstants() {}

    // Keycloak User Attributes
    public static final String ATTR_APP_USER_ID = "app_user_id";
    public static final String ATTR_PRINCIPAL_TYPE = "principal_type";
    public static final String ATTR_PERMISSION_GRANTS = "permission_grants";
    public static final String ATTR_PERMISSIONS = "permissions";

    // JWT Claims
    public static final String CLAIM_SUB = "sub";
    public static final String CLAIM_USER_ID = "user_id";
    public static final String CLAIM_PRINCIPAL_TYPE = "principal_type";
    public static final String CLAIM_PERMISSION_GRANTS = "permission_grants";
    public static final String CLAIM_EMAIL = "email";
    public static final String CLAIM_PREFERRED_USERNAME = "preferred_username";

    // Shared Roles & Scopes
    public static final String ROLE_ADMIN = "ADMIN";
    public static final String SCOPE_GLOBAL = "GLOBAL";
    public static final String ACTOR_KIND_SERVICE = "SERVICE";

    // HTTP Headers
    public static final String HEADER_IDEMPOTENCY_KEY = "Idempotency-Key";
}
