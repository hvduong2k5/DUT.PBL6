package com.hvduong.catalog.repository.mybatis.projection;

import java.time.Instant;

import lombok.Data;

@Data
public class AdminStockProjectionRow {
  private String skuCode;
  private Long sellableQuantity;
  private Long inventoryVersion;
  private Instant sourceUpdatedAt;
  private Instant syncedAt;
  private Long linkedVariantCount;
}
