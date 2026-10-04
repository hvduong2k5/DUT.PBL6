package com.hvduong.catalog.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminProductListItemResponse {
    private String productId;
    private String name;
    private String categoryId;
    private String categoryName;
    private String saleStatus;
    private String saleStatusLabel;
    private Integer skuCount;
    private Integer onSaleSkuCount;
    private Long basePriceFromVnd;
    private Boolean foodInformationComplete;
    private String updatedAt;
    private Long revision;
}
