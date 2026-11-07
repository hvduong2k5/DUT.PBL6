package com.hvduong.identity.common.enums;

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
}
