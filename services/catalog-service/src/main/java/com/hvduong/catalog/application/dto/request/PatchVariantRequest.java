package com.hvduong.catalog.application.dto.request;

import java.math.BigDecimal;

import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;
import jakarta.validation.constraints.Size;
import lombok.Data;

/**
 * Input DTO cho PATCH SKU — cập nhật một phần thông tin biến thể.
 *
 * <p>{@code skuCode} là định danh nghiệp vụ bất biến, không cho phép đổi qua PATCH.
 *
 * <p>Quy ước PATCH: field {@code null}/không gửi = giữ nguyên; chuỗi rỗng = xóa field optional
 * (flavor, packagingType).
 */
@Data
public class PatchVariantRequest {

  @Size(max = 500)
  private String variantName;

  @Positive(message = "weightValue phải lớn hơn 0")
  private BigDecimal weightValue;

  private String weightUnit;

  @Size(max = 200)
  private String flavor;

  private String packagingType;

  @Positive(message = "shelfLifeDays phải lớn hơn 0")
  private Integer shelfLifeDays;

  /** ON_SALE | PAUSED | DISCONTINUED (chấp nhận cả ACTIVE/SUSPENDED/ARCHIVED). */
  private String saleStatus;

  @NotNull(message = "expectedVersion không được bỏ trống")
  private Long expectedVersion;
}
