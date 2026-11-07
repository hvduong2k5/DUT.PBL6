package com.hvduong.identity.common.enums;

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
}
