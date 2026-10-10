package com.hvduong.catalog.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.math.BigDecimal;
import java.util.UUID;

/** Response cho biến thể sản phẩm (SKU). */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProductVariantResponse {
    private UUID variantId;
    private String skuCode;
    private String name;
    private BigDecimal weightValue;
    private String weightUnit;
    private String flavor;
    private String packagingType;
    private Integer shelfLifeDays;
    private MoneyResponse price;
    private String listingStatus;
    private Long stockAvailable;
}
