package com.hvduong.catalog.application.mapper;

import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductImageResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListItemResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
import com.hvduong.catalog.common.constant.CatalogConstants;
import com.hvduong.catalog.common.constant.SaleStatus;
import com.hvduong.catalog.common.enums.ListingStatus;
import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductImage;
import com.hvduong.catalog.domain.entity.ProductVariant;

import java.math.BigDecimal;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.stream.Collectors;

/**
 * Centralized mapper cho Admin Catalog responses. Tập trung logic chuyển đổi từ Entity +
 * ChannelPrice sang Admin DTO, tránh散 khắp service layer.
 */
public final class AdminCatalogMapper {

  private AdminCatalogMapper() {}

  /** Map từ Product + variants + prices sang AdminProductListItemResponse. */
  public static AdminProductListItemResponse toListItemResponse(
      Product product,
      List<ProductVariant> variants,
      Map<UUID, ChannelPrice> currentPriceByVariant) {
    AdminProductListItemResponse item = new AdminProductListItemResponse();
    item.setProductId(product.getId().toString());
    item.setName(product.getName());
    item.setCategoryId(product.getCategoryId().toString());
    item.setCategoryName(product.getCategoryName());
    SaleStatus status = SaleStatus.fromListingStatus(product.getListingStatus());
    item.setSaleStatus(status.getCode());
    item.setSaleStatusLabel(status.getLabelVn());
    item.setSkuCount(variants.size());
    item.setOnSaleSkuCount(
        variants.stream()
            .filter(v -> v.getListingStatus() == ListingStatus.ACTIVE)
            .mapToInt(v -> 1)
            .sum());
    item.setBasePriceFromVnd(skuBasePriceFromVnd(variants, currentPriceByVariant));
    item.setFoodInformationComplete(foodInformationComplete(product));
    item.setUpdatedAt(formatInstant(product.getUpdatedAt()));
    item.setRevision(product.getVersion());
    return item;
  }

  /** Map từ ProductVariant + ChannelPrice sang AdminProductSkuResponse. */
  public static AdminProductDetailResponse.AdminProductSkuResponse toSkuResponse(
      ProductVariant variant, ChannelPrice price) {
    AdminProductDetailResponse.AdminProductSkuResponse sku =
        new AdminProductDetailResponse.AdminProductSkuResponse();
    sku.setSkuId(variant.getId().toString());
    sku.setSkuCode(variant.getSkuCode());
    sku.setLabel(variant.getVariantName());
    sku.setWeightGrams(weightGrams(variant.getWeightValue(), variant.getWeightUnit()));
    sku.setFlavor(variant.getFlavor());
    sku.setPackageType(packagingLabel(variant.getPackagingType()));
    sku.setBasePriceVnd(price != null ? price.getAmountUnits() : null);
    sku.setCurrency("VND");
    SaleStatus status = SaleStatus.fromListingStatus(variant.getListingStatus());
    sku.setSaleStatus(status.getCode());
    sku.setSaleStatusLabel(status.getLabelVn());
    sku.setUpdatedAt(formatInstant(variant.getUpdatedAt()));
    sku.setRevision(variant.getVersion());
    return sku;
  }

  public static Long skuBasePriceFromVnd(
      List<ProductVariant> variants, Map<UUID, ChannelPrice> currentPriceByVariant) {
    Long min = null;
    for (ProductVariant variant : variants) {
      ChannelPrice price = currentPriceByVariant.get(variant.getId());
      if (price != null && price.getAmountUnits() > 0) {
        Long units = price.getAmountUnits();
        min = (min == null || units < min) ? units : min;
      }
    }
    return min;
  }

  public static boolean foodInformationComplete(Product product) {
    return org.springframework.util.StringUtils.hasText(product.getIngredients())
        && org.springframework.util.StringUtils.hasText(product.getStorageGuide())
        && org.springframework.util.StringUtils.hasText(product.getName());
  }

  public static Integer weightGrams(BigDecimal weightValue, String weightUnit) {
    if (weightValue == null) {
      return null;
    }
    if ("KILOGRAM".equalsIgnoreCase(weightUnit)) {
      return weightValue.multiply(BigDecimal.valueOf(1000)).intValue();
    }
    return weightValue.intValue();
  }

  public static String packagingLabel(String packagingType) {
    if (packagingType == null) return null;
    return packagingType.replace('_', ' ').toLowerCase();
  }

  public static String formatInstant(Instant instant) {
    return instant == null ? null : instant.toString();
  }

  /** Chuẩn hóa cover image URL (placeholder nếu không có). */
  public static String coverImageUrl(List<ProductImage> images) {
    if (images == null || images.isEmpty()) return null;
    return images.stream()
        .filter(img -> "COVER".equals(img.getRole()) && "APPROVED".equals(img.getMediaStatus()))
        .findFirst()
        .map(img -> imageUrl(img.getObjectKey()))
        .orElse(null);
  }

  /** Ghép object key với media base URL. */
  public static String imageUrl(String objectKey) {
    if (objectKey == null || objectKey.isBlank()) {
      return null;
    }
    if (objectKey.startsWith("http://") || objectKey.startsWith("https://")) {
      return objectKey;
    }
    return CatalogConstants.MEDIA_BASE_URL + objectKey;
  }

  /** Map ProductImage entity sang DTO admin. */
  public static AdminProductImageResponse toAdminImageResponse(ProductImage image) {
    return AdminProductImageResponse.builder()
        .imageId(image.getId() == null ? null : image.getId().toString())
        .objectKey(image.getObjectKey())
        .url(imageUrl(image.getObjectKey()))
        .altText(image.getAltText())
        .role(image.getRole())
        .position(image.getPosition())
        .mediaStatus(image.getMediaStatus())
        .revision(image.getVersion())
        .createdAt(formatInstant(image.getCreatedAt()))
        .updatedAt(formatInstant(image.getUpdatedAt()))
        .build();
  }

  /** Chuẩn hóa cover image alt text. */
  public static String coverImageAlt(List<ProductImage> images, String productName) {
    if (images == null || images.isEmpty()) return productName;
    return images.stream()
        .filter(img -> "COVER".equals(img.getRole()) && "APPROVED".equals(img.getMediaStatus()))
        .findFirst()
        .map(ProductImage::getAltText)
        .filter(org.springframework.util.StringUtils::hasText)
        .orElse(productName);
  }

  /** Lấy danh sách status counts (ACTIVE, SUSPENDED, ARCHIVED) với count từ map. */
  public static List<AdminProductListResponse.StatusCount> toStatusCounts(
      Map<String, Long> statusCountMap) {
    return List.of(ListingStatus.ACTIVE, ListingStatus.SUSPENDED, ListingStatus.ARCHIVED).stream()
        .map(
            ls ->
                new AdminProductListResponse.StatusCount(
                    SaleStatus.fromListingStatus(ls).getCode(),
                    SaleStatus.fromListingStatus(ls).getLabelVn(),
                    statusCountMap.getOrDefault(ls.getValue(), 0L)))
        .collect(Collectors.toList());
  }

  /** Lấy danh sách categories từ AdminCategoryRow list. */
  public static List<AdminProductListResponse.CategoryOption> toCategoryOptions(
      List<com.hvduong.catalog.repository.mybatis.projection.AdminCategoryRow> rows) {
    return rows.stream()
        .map(
            row ->
                new AdminProductListResponse.CategoryOption(
                    row.getCategoryId().toString(), row.getCategoryName()))
        .collect(Collectors.toList());
  }
}
