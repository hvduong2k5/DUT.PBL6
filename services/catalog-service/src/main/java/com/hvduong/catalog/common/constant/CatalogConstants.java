package com.hvduong.catalog.common.constant;

/**
 * Hằng số định danh cho Catalog domain. Tránh hard-code magic strings (channel, listing status,
 * sale status, contract version...) xuyên toàn bộ code.
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

  /** Base URL công khai của media/CDN, ghép với {@code object_key} để tạo URL ảnh. */
  public static final String MEDIA_BASE_URL = "https://cdn.omama.vn/";

  /** Các role ảnh hợp lệ theo enum {@code image_role}. */
  public static final java.util.Set<String> IMAGE_ROLES =
      java.util.Set.of("COVER", "GALLERY", "THUMBNAIL");

  /** Các trạng thái kiểm duyệt ảnh hợp lệ theo enum {@code media_status}. */
  public static final java.util.Set<String> IMAGE_STATUSES =
      java.util.Set.of("PENDING_SCAN", "APPROVED", "REJECTED");

  /** Các loại bao bì hợp lệ theo enum {@code packaging_type}. */
  public static final java.util.Set<String> PACKAGING_TYPES =
      java.util.Set.of(
          "HOP_GIAY_KRAFT", "HOP_THIEC", "GOI_HUT_CHAN_KHONG", "HOP_QUA_TANG", "DANG_ROI");

  /** Đơn vị khối lượng hợp lệ. */
  public static final java.util.Set<String> WEIGHT_UNITS = java.util.Set.of("GRAM", "KILOGRAM");
}
