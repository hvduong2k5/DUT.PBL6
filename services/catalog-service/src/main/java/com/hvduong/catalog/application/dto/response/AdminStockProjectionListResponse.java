package com.hvduong.catalog.application.dto.response;

import java.util.List;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Payload nằm trong {@code data} của GET /api/v1/admin/catalog/stock-projections. Khớp với {@code
 * CatalogAdminStockProjectionPageData} trong admin-stable.openapi.yaml.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminStockProjectionListResponse {

  private List<Item> items;
  private int page;
  private int pageSize;
  private long total;

  @Data
  @Builder
  @NoArgsConstructor
  @AllArgsConstructor
  public static class Item {
    private String skuCode;
    private Long sellableQuantity;
    private Long inventoryVersion;
    private String sourceUpdatedAt;
    private String syncedAt;
    private long linkedVariantCount;
  }
}
