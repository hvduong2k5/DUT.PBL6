package com.hvduong.identity.common.enums;

import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Lifecycle status of an IAM role assignment.
 */
public enum AssignmentStatus {
    ACTIVE("ACTIVE"),
    REVOKED("REVOKED"),
    EXPIRED("EXPIRED");

    private final String value;

    AssignmentStatus(String value) {
        this.value = value;
    }

    @JsonValue
    public String getValue() {
        return value;
    }
}
