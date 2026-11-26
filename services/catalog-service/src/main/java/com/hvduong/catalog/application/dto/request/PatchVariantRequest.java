package com.hvduong.catalog.application.dto.request;

import java.math.BigDecimal;

import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Positive;
import jakarta.validation.constraints.Size;
import lombok.Data;

/**
 * Input DTO cho PATCH SKU — cập nhật một phần thông tin biến thể.
 *
 * <p>{@code skuCode} là định danh nghiệp vụ bất biến, không cho phép đổi qua PATCH.
 *
 * <p>Quy ước PATCH: field {@code null}/không gửi = giữ nguyên; chuỗi rỗng = xóa field optional
 * (inventorySkuCode, flavor, packagingType).
 */
@Data
public class PatchVariantRequest {

  @Size(max = 100)
  private String inventorySkuCode;

  @Size(max = 500)
  private String variantName;

  @Positive(message = "weightValue phải lớn hơn 0")
  private BigDecimal weightValue;

  @Pattern(
      regexp = "^(GRAM|KILOGRAM)$",
      message = "Đơn vị trọng lượng không hợp lệ. Chỉ chấp nhận: GRAM, KILOGRAM")
  private String weightUnit;

  @Size(max = 200)
  private String flavor;

  @Pattern(
      regexp = "^(HOP_GIAY_KRAFT|HOP_THIEC|GOI_HUT_CHAN_KHONG|HOP_QUA_TANG|DANG_ROI)$",
      message =
          "Loại bao bì không hợp lệ. Chỉ chấp nhận: HOP_GIAY_KRAFT, HOP_THIEC, GOI_HUT_CHAN_KHONG, HOP_QUA_TANG, DANG_ROI")
  private String packagingType;

  @Positive(message = "shelfLifeDays phải lớn hơn 0")
  private Integer shelfLifeDays;

  /** ON_SALE | PAUSED | DISCONTINUED (chấp nhận cả ACTIVE/SUSPENDED/ARCHIVED). */
  @Pattern(
      regexp = "^(ON_SALE|PAUSED|DISCONTINUED|ACTIVE|SUSPENDED|ARCHIVED)$",
      message = "Trạng thái bán không hợp lệ. Chỉ chấp nhận: ON_SALE, PAUSED, DISCONTINUED")
  private String saleStatus;

  @NotNull(message = "expectedVersion không được bỏ trống")
  private Long expectedVersion;
}
