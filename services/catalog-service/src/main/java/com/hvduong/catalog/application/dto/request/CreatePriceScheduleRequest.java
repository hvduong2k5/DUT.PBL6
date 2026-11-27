package com.hvduong.catalog.application.dto.request;

import com.hvduong.catalog.common.enums.SalesChannel;

import java.time.Instant;

import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.PositiveOrZero;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** DTO để tạo mới hoặc hẹn giờ hiệu lực một bản ghi giá theo kênh cho SKU. */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class CreatePriceScheduleRequest {

  @NotNull(message = "Kênh bán hàng không được để trống")
  private SalesChannel channel;

  @NotNull(message = "Giá không được để trống")
  @PositiveOrZero(message = "Giá phải lớn hơn hoặc bằng 0")
  private Long amountUnits;

  @Builder.Default
  @Min(value = 0, message = "Nanos không được nhỏ hơn 0")
  @Max(value = 999999999, message = "Nanos không được vượt quá 999,999,999")
  private Integer amountNanos = 0;

  @Builder.Default private String currencyCode = "VND";

  @NotNull(message = "Thời điểm bắt đầu hiệu lực không được để trống")
  private Instant effectiveFrom;

  private Instant effectiveTo;
}
