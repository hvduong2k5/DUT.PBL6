package com.hvduong.catalog.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Một bản ghi giá theo kênh của SKU, dùng cho màn hình lịch sử giá admin. */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminPriceResponse {
  private String priceId;
  private String channel;
  private String currencyCode;
  private Long amountUnits;
  private Integer amountNanos;
  private String status;
  private String effectiveFrom;
  private String effectiveTo;
  private Long revision;
}
