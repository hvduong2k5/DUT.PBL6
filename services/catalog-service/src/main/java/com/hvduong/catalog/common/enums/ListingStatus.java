package com.hvduong.catalog.common.enums;

import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Trạng thái kinh doanh (listing) của Product/Variant.
 * ACTIVE = đang bán | SUSPENDED = tạm ngừng | ARCHIVED = ngừng vĩnh viễn
 */
public enum ListingStatus {
    ACTIVE("ACTIVE"),
    SUSPENDED("SUSPENDED"),
    ARCHIVED("ARCHIVED");

    private final String value;

    ListingStatus(String value) { this.value = value; }

    @JsonValue
    public String getValue() { return value; }
}
