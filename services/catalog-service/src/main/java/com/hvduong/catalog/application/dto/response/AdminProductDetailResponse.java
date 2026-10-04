package com.hvduong.catalog.application.dto.response;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import java.util.List;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminProductDetailResponse {
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
    private String shortDescription;
    private String longDescription;
    private String coverImageUrl;
    private String coverImageAlt;
    
    private AdminFoodInformationResponse foodInformation;
    private List<AdminProductSkuResponse> skus;
    
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class AdminFoodInformationResponse {
        private String ingredients;
        private String allergenStatement;
        private String storageInstructions;
        private String manufacturingDatePolicy;
        private String shelfLifeDescription;
    }
    
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class AdminProductSkuResponse {
        private String skuId;
        private String skuCode;
        private String label;
        private Integer weightGrams;
        private String flavor;
        private String packageType;
        private Long basePriceVnd;
        private String currency;
        private String saleStatus;
        private String saleStatusLabel;
        private String updatedAt;
        private Long revision;
    }
}
