package com.hvduong.catalog.common.constant;

import com.hvduong.catalog.common.enums.ListingStatus;

/**
 * Ánh xạ giữa {@link com.hvduong.catalog.common.enums.ListingStatus} (trong DB) và {@code
 * saleStatus} dùng trong contract admin / BFF (ON_SALE / PAUSED / DISCONTINUED).
 */
public enum SaleStatus {
  ON_SALE("ON_SALE", "Đang bán"),
  PAUSED("PAUSED", "Tạm ngừng bán"),
  DISCONTINUED("DISCONTINUED", "Ngừng kinh doanh");

  private final String code;
  private final String labelVn;

  SaleStatus(String code, String labelVn) {
    this.code = code;
    this.labelVn = labelVn;
  }

  public String getCode() {
    return code;
  }

  public String getLabelVn() {
    return labelVn;
  }

  /** Ánh xạ từ ListingStatus (DB) sang SaleStatus (contract). */
  public static SaleStatus fromListingStatus(ListingStatus listingStatus) {
    if (listingStatus == null) {
      return DISCONTINUED;
    }
    return switch (listingStatus) {
      case ACTIVE -> ON_SALE;
      case SUSPENDED -> PAUSED;
      case ARCHIVED -> DISCONTINUED;
    };
  }

  /**
   * Chấp nhận cả mã contract (ON_SALE/PAUSED/DISCONTINUED) và mã DB (ACTIVE/SUSPENDED/ARCHIVED).
   * Trả {@code null} nếu không hợp lệ.
   */
  public static ListingStatus toListingStatus(String status) {
    if (status == null || status.isBlank()) {
      return null;
    }
    return switch (status.trim().toUpperCase(java.util.Locale.ROOT)) {
      case "ON_SALE", "ACTIVE" -> ListingStatus.ACTIVE;
      case "PAUSED", "SUSPENDED" -> ListingStatus.SUSPENDED;
      case "DISCONTINUED", "ARCHIVED" -> ListingStatus.ARCHIVED;
      default -> null;
    };
  }
}
