package com.hvduong.catalog.application.mapper;

import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse.AdminFoodInformationResponse;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse.AdminProductSkuResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListItemResponse;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.domain.entity.Category;
import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductVariant;

import java.util.List;

import org.mapstruct.Mapper;
import org.mapstruct.Mapping;

/** MapStruct mapper: Entity → Application DTO Response. */
@Mapper(componentModel = "spring")
public interface CatalogDtoMapper {

  // ── Category ──────────────────────────────────────────────────────────────
  @Mapping(source = "id", target = "categoryId")
  @Mapping(target = "children", ignore = true)
  CategoryResponse toCategoryResponse(Category category);

  List<CategoryResponse> toCategoryResponseList(List<Category> categories);

  // ── Product List Item ─────────────────────────────────────────────────────
  @Mapping(source = "id", target = "productId")
  @Mapping(target = "thumbnailUrl", ignore = true)
  @Mapping(target = "basePrice", ignore = true)
  @Mapping(target = "inStock", ignore = true)
  @Mapping(target = "stockAvailable", ignore = true)
  @Mapping(target = "summary", source = "description")
  ProductListItemResponse toListItemResponse(Product product);

  List<ProductListItemResponse> toListItemResponseList(List<Product> products);

  // ── Product Detail ────────────────────────────────────────────────────────
  @Mapping(source = "id", target = "productId")
  @Mapping(target = "imageUrls", ignore = true)
  @Mapping(target = "variants", ignore = true)
  ProductDetailResponse toDetailResponse(Product product);

  // ── Variant ───────────────────────────────────────────────────────────────
  @Mapping(source = "id", target = "variantId")
  @Mapping(source = "productId", target = "productId")
  @Mapping(source = "variantName", target = "name")
  @Mapping(target = "price", ignore = true)
  @Mapping(target = "stockAvailable", ignore = true)
  ProductVariantResponse toVariantResponse(ProductVariant variant);

  List<ProductVariantResponse> toVariantResponseList(List<ProductVariant> variants);

  // ── Money ─────────────────────────────────────────────────────────────────
  @Mapping(source = "amountUnits", target = "units")
  @Mapping(source = "amountNanos", target = "nanos")
  MoneyResponse toMoneyResponse(ChannelPrice price);

  // ── Admin ─────────────────────────────────────────────────────────────────
  @Mapping(source = "id", target = "productId")
  @Mapping(target = "saleStatus", ignore = true)
  @Mapping(target = "saleStatusLabel", ignore = true)
  @Mapping(target = "skuCount", ignore = true)
  @Mapping(target = "onSaleSkuCount", ignore = true)
  @Mapping(target = "basePriceFromVnd", ignore = true)
  @Mapping(target = "foodInformationComplete", ignore = true)
  @Mapping(target = "updatedAt", ignore = true)
  @Mapping(source = "version", target = "revision")
  @Mapping(target = "categoryName", ignore = true)
  AdminProductListItemResponse toAdminListItemResponse(Product product);

  List<AdminProductListItemResponse> toAdminListItemResponseList(List<Product> products);

  @Mapping(source = "id", target = "productId")
  @Mapping(target = "saleStatus", ignore = true)
  @Mapping(target = "saleStatusLabel", ignore = true)
  @Mapping(target = "skuCount", ignore = true)
  @Mapping(target = "onSaleSkuCount", ignore = true)
  @Mapping(target = "basePriceFromVnd", ignore = true)
  @Mapping(target = "foodInformationComplete", ignore = true)
  @Mapping(target = "updatedAt", ignore = true)
  @Mapping(source = "version", target = "revision")
  @Mapping(target = "categoryName", ignore = true)
  @Mapping(target = "shortDescription", source = "description")
  @Mapping(target = "longDescription", ignore = true)
  @Mapping(target = "coverImageUrl", ignore = true)
  @Mapping(target = "coverImageAlt", ignore = true)
  @Mapping(target = "foodInformation", expression = "java(toAdminFoodInformationResponse(product))")
  @Mapping(target = "skus", ignore = true)
  AdminProductDetailResponse toAdminDetailResponse(Product product);

  @Mapping(source = "ingredients", target = "ingredients")
  @Mapping(source = "storageGuide", target = "storageInstructions")
  @Mapping(target = "allergenStatement", constant = "Thông tin cảnh báo dị ứng đang được cập nhật")
  @Mapping(target = "manufacturingDatePolicy", constant = "Ngày sản xuất ghi trên bao bì")
  @Mapping(target = "shelfLifeDescription", constant = "Hạn sử dụng ghi trên bao bì")
  AdminFoodInformationResponse toAdminFoodInformationResponse(Product product);

  @Mapping(source = "id", target = "skuId")
  @Mapping(source = "listingStatus", target = "saleStatus")
  @Mapping(source = "variantName", target = "label")
  @Mapping(source = "weightValue", target = "weightGrams")
  @Mapping(source = "packagingType", target = "packageType")
  @Mapping(target = "basePriceVnd", ignore = true)
  @Mapping(target = "currency", ignore = true)
  @Mapping(target = "saleStatusLabel", ignore = true)
  @Mapping(target = "updatedAt", ignore = true)
  @Mapping(target = "revision", ignore = true)
  AdminProductSkuResponse toAdminSkuResponse(ProductVariant variant);

  List<AdminProductSkuResponse> toAdminSkuResponseList(List<ProductVariant> variants);
}
