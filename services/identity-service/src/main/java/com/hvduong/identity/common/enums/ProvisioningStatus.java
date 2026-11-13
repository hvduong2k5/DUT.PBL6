package com.hvduong.identity.common.enums;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Lifecycle provisioning status of an identity principal.
 */
public enum ProvisioningStatus {
    PENDING("PENDING"),
    ACTIVE("ACTIVE"),
    SUSPENDED("SUSPENDED"),
    TERMINATED("TERMINATED"),
    FAILED("FAILED");

    private final String value;

    ProvisioningStatus(String value) {
        this.value = value;
    }

    @JsonValue
    public String getValue() {
        return value;
    }

    @JsonCreator
    public static ProvisioningStatus fromString(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        for (ProvisioningStatus status : values()) {
            if (status.value.equalsIgnoreCase(value.trim())) {
                return status;
            }
        }
        throw new IllegalArgumentException("Unknown provisioning status: " + value);
    }

    public boolean matches(String value) {
        return this.value.equalsIgnoreCase(value);
    }
}
