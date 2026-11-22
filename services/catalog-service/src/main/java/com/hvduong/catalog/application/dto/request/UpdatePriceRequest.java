package com.hvduong.catalog.application.dto.request;

import com.hvduong.catalog.common.enums.SalesChannel;

import java.time.Instant;
import java.util.List;

import jakarta.validation.Valid;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.PositiveOrZero;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Input DTO để cập nhật bảng giá Variant theo kênh. */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class UpdatePriceRequest {

  private Long expectedVariantVersion;

  /** Alias tương thích cho expectedVersion. */
  public Long getExpectedVersion() {
    return expectedVariantVersion;
  }

  public void setExpectedVersion(Long expectedVersion) {
    if (this.expectedVariantVersion == null) {
      this.expectedVariantVersion = expectedVersion;
    }
  }

  @NotEmpty(message = "Danh sách giá không được bỏ trống")
  @Valid
  private List<PriceEntry> prices;

  @Data
  @Builder
  @NoArgsConstructor
  @AllArgsConstructor
  public static class PriceEntry {

    @NotNull(message = "Kênh bán hàng không được để trống")
    private SalesChannel channel;

    @NotNull(message = "Giá không được để trống")
    @PositiveOrZero(message = "Giá phải lớn hơn hoặc bằng 0")
    private Long amountUnits;

    @Builder.Default private Integer amountNanos = 0;

    @Builder.Default private String currencyCode = "VND";

    private Instant effectiveFrom;

    private Instant effectiveTo;
  }
}
