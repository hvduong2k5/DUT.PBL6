package com.hvduong.catalog.common.constant;

/**
 * Ánh xạ giữa {@link com.hvduong.catalog.common.enums.ListingStatus} (trong DB)
 * và {@code saleStatus} dùng trong contract admin / BFF (ON_SALE / PAUSED / DISCONTINUED).
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
  public static SaleStatus fromListingStatus(com.hvduong.catalog.common.enums.ListingStatus listingStatus) {
    return switch (listingStatus) {
      case ACTIVE -> ON_SALE;
      case SUSPENDED -> PAUSED;
      case ARCHIVED -> DISCONTINUED;
      default -> DISCONTINUED;
    };
  }
}
