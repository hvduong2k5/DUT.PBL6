package com.hvduong.catalog.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.UUID;

/** Response cho một item trong danh sách sản phẩm. */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProductListItemResponse {
    private UUID productId;
    private String name;
    private String slug;
    private String summary;
    private String thumbnailUrl;
    private MoneyResponse basePrice;
    private Integer ocopStar;
    private String categoryName;
    private Boolean inStock;
    private Long stockAvailable;
}
