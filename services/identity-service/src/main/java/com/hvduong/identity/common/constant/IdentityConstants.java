package com.hvduong.identity.common.constant;

/**
 * Domain constants for Identity Service.
 */
public final class IdentityConstants {

    private IdentityConstants() {}

    public static final String ROLE_ADMIN = "ADMIN";
    public static final String PRINCIPAL_TYPE_CUSTOMER = "CUSTOMER";
    public static final String PRINCIPAL_TYPE_EMPLOYEE = "EMPLOYEE";

    public static final String ASSIGNMENT_STATUS_ACTIVE = "ACTIVE";
    public static final String ASSIGNMENT_STATUS_REVOKED = "REVOKED";
    public static final String ASSIGNMENT_STATUS_EXPIRED = "EXPIRED";

    public static final String PROVISIONING_STATUS_ACTIVE = "ACTIVE";
    public static final String PROVISIONING_STATUS_SUSPENDED = "SUSPENDED";
    public static final String PROVISIONING_STATUS_TERMINATED = "TERMINATED";

    public static final String SCOPE_GLOBAL = "GLOBAL";
    public static final String HEADER_IDEMPOTENCY_KEY = "Idempotency-Key";
}
