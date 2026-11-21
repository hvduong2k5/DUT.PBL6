package com.hvduong.catalog.common.enums;

import com.fasterxml.jackson.annotation.JsonValue;

/** Kênh bán hàng — đồng nhất với proto SalesChannel enum và DB enum sales_channel. */
public enum SalesChannel {
  D2C_WEB("D2C_WEB"),
  MOBILE_APP("MOBILE_APP"),
  B2B_WHOLESALE("B2B_WHOLESALE"),
  POS_QUAY("POS_QUAY"),
  SHOPEE("SHOPEE"),
  TIKTOK("TIKTOK");

  private final String value;

  SalesChannel(String value) {
    this.value = value;
  }

  @JsonValue
  public String getValue() {
    return value;
  }

  public static SalesChannel from(String name) {
    if (name == null || name.isBlank()) {
      return D2C_WEB;
    }
    for (SalesChannel sc : values()) {
      if (sc.name().equalsIgnoreCase(name.trim()) || sc.value.equalsIgnoreCase(name.trim())) {
        return sc;
      }
    }
    throw new IllegalArgumentException("Unknown sales channel: " + name);
  }
}
