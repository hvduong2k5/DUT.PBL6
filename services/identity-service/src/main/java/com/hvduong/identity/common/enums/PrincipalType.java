package com.hvduong.identity.common.enums;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Identity principal classification (`CUSTOMER` or `EMPLOYEE`).
 */
public enum PrincipalType {
    CUSTOMER("CUSTOMER"),
    EMPLOYEE("EMPLOYEE");

    private final String value;

    PrincipalType(String value) {
        this.value = value;
    }

    @JsonValue
    public String getValue() {
        return value;
    }

    @JsonCreator
    public static PrincipalType fromString(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        for (PrincipalType type : values()) {
            if (type.value.equalsIgnoreCase(value.trim())) {
                return type;
            }
        }
        throw new IllegalArgumentException("Unknown principal type: " + value);
    }

    public boolean matches(String value) {
        return this.value.equalsIgnoreCase(value);
    }
}
