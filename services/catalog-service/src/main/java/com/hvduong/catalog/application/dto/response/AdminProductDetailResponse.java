package com.hvduong.catalog.application.dto.response;

import com.hvduong.catalog.domain.entity.Product;

import java.util.List;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

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

  // ── Nội dung đầy đủ để admin chỉnh sửa (PATCH) ─────────────────────────
  private String slug;
  private String description;
  private Integer ocopStar;
  private String ocopCertificateNo;
  private String story;
  private String approvalStatus;
  private String listingStatus;

  private String coverImageUrl;
  private String coverImageAlt;
  private List<AdminProductImageResponse> images;

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

    /** Utility factory method map từ Product entity */
    public static AdminFoodInformationResponse from(Product product) {
      if (product == null) {
        return null;
      }
      return AdminFoodInformationResponse.builder()
          .ingredients(product.getIngredients())
          .allergenStatement("Thông tin cảnh báo dị ứng đang được cập nhật")
          .storageInstructions(product.getStorageGuide())
          .manufacturingDatePolicy("Ngày sản xuất ghi trên bao bì")
          .shelfLifeDescription("Hạn sử dụng ghi trên bao bì")
          .build();
    }
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
