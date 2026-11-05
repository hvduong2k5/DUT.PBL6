package com.hvduong.catalog.common.constant;

/**
 * Hằng số định danh cho Catalog domain. Tránh hard-code magic strings
 * (channel, listing status, sale status, contract version...) xuyên toàn bộ code.
 */
public final class CatalogConstants {

  private CatalogConstants() {}

  /** Channels được phép cho admin read / write. */
  public static final String CHANNEL_D2C_WEB = "D2C_WEB";
  public static final String CHANNEL_MOBILE_APP = "MOBILE_APP";

  /** Phiên bản contract checkout snapshot. */
  public static final String CONTRACT_VERSION_CHECKOUT_V1 = "catalog-checkout-1.0";

  /** Thời gian hết hạn mặc định của snapshot (giây). */
  public static final long SNAPSHOT_VALID_FOR_SECONDS = 300;

  /** Số lượng item tối đa trên một snapshot request. */
  public static final int SNAPSHOT_MAX_ITEMS = 100;

  /** Số lượng item tối thiểu trên một snapshot request. */
  public static final int SNAPSHOT_MIN_QUANTITY = 1;

  /** Số lượng item tối đa trên một snapshot request (từng SKU). */
  public static final int SNAPSHOT_MAX_QUANTITY = 999;
}
