package com.hvduong.identity.common.enums;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Lifecycle status of an employee invitation.
 */
public enum InvitationStatus {
    PENDING("PENDING"),
    ACCEPTED("ACCEPTED"),
    EXPIRED("EXPIRED"),
    REVOKED("REVOKED");

    private final String value;

    InvitationStatus(String value) {
        this.value = value;
    }

    @JsonValue
    public String getValue() {
        return value;
    }

    @JsonCreator
    public static InvitationStatus fromString(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        for (InvitationStatus status : values()) {
            if (status.value.equalsIgnoreCase(value.trim())) {
                return status;
            }
        }
        throw new IllegalArgumentException("Unknown invitation status: " + value);
    }

    public boolean matches(String value) {
        return this.value.equalsIgnoreCase(value);
    }
}
