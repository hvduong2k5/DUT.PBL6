package com.hvduong.catalog.application.dto.response;

import lombok.Builder;
import lombok.Data;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

/** Response chi tiết sản phẩm. */
@Data
@Builder
public class ProductDetailResponse {
    private UUID productId;
    private UUID categoryId;
    private String categoryName;
    private String name;
    private String slug;
    private String description;
    private String ingredients;
    private String storageGuide;
    private Integer ocopStar;
    private String ocopCertificateNo;
    private String story;
    private String approvalStatus;
    private String listingStatus;
    private List<String> imageUrls;
    private List<ProductVariantResponse> variants;
    private Instant createdAt;
    private Instant updatedAt;
    private long version;
}
