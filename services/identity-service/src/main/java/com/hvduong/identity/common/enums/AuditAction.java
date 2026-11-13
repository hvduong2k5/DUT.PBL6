package com.hvduong.identity.common.enums;

import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Standardized audit log actions for Identity & IAM operations.
 */
public enum AuditAction {
    ACCOUNT_SUSPEND("ACCOUNT_SUSPEND"),
    ACCOUNT_REACTIVATE("ACCOUNT_REACTIVATE"),
    ROLE_ASSIGN("ROLE_ASSIGN"),
    ROLE_REVOKE("ROLE_REVOKE"),
    ROLE_DISABLE("ROLE_DISABLE"),
    PERMISSION_CHANGE("PERMISSION_CHANGE");

    private final String value;

    AuditAction(String value) {
        this.value = value;
    }

    @JsonValue
    public String getValue() {
        return value;
    }
}
