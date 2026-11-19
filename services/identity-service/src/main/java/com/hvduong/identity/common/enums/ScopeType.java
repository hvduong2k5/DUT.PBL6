package com.hvduong.identity.common.enums;

import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Scope type for IAM role assignments.
 */
public enum ScopeType {
    GLOBAL("GLOBAL"),
    WAREHOUSE("WAREHOUSE"),
    BRANCH("BRANCH"),
    CHANNEL("CHANNEL");

    private final String value;

    ScopeType(String value) {
        this.value = value;
    }

    @JsonValue
    public String getValue() {
        return value;
    }
}
