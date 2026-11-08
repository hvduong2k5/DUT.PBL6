package com.hvduong.identity.common.enums;

import com.fasterxml.jackson.annotation.JsonCreator;
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

    @JsonCreator
    public static AssignmentStatus fromString(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        for (AssignmentStatus status : values()) {
            if (status.value.equalsIgnoreCase(value.trim())) {
                return status;
            }
        }
        throw new IllegalArgumentException("Unknown assignment status: " + value);
    }

    public boolean matches(String value) {
        return this.value.equalsIgnoreCase(value);
    }
}
