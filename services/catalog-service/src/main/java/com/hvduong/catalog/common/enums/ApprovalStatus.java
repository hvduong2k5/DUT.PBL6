package com.hvduong.catalog.common.enums;

import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Trạng thái phê duyệt của Product.
 * State machine: DRAFT → PENDING → APPROVED / REJECTED → DRAFT (revision)
 */
public enum ApprovalStatus {
    DRAFT("DRAFT"),
    PENDING("PENDING"),
    APPROVED("APPROVED"),
    REJECTED("REJECTED");

    private final String value;

    ApprovalStatus(String value) { this.value = value; }

    @JsonValue
    public String getValue() { return value; }
}
