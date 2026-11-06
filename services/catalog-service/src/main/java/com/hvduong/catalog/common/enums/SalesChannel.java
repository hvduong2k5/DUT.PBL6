package com.hvduong.catalog.common.enums;

import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Kênh bán hàng — đồng nhất với proto SalesChannel enum và DB enum sales_channel.
 */
public enum SalesChannel {
    D2C_WEB("D2C_WEB"),
    MOBILE_APP("MOBILE_APP"),
    B2B_WHOLESALE("B2B_WHOLESALE"),
    POS_QUAY("POS_QUAY"),
    SHOPEE("SHOPEE"),
    TIKTOK("TIKTOK");

    private final String value;

    SalesChannel(String value) { this.value = value; }

    @JsonValue
    public String getValue() { return value; }
}
