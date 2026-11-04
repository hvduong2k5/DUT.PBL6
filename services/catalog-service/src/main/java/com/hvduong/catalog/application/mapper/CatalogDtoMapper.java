package com.hvduong.catalog.application.mapper;

import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.domain.entity.Category;
import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductVariant;
import org.mapstruct.Mapper;
import org.mapstruct.Mapping;

import java.util.List;

/**
 * MapStruct mapper: Entity → Application DTO Response.
 */
@Mapper(componentModel = "spring")
public interface CatalogDtoMapper {

    // ── Category ──────────────────────────────────────────────────────────────
    @Mapping(source = "id",       target = "categoryId")
    
    @Mapping(target = "children", ignore = true)
    CategoryResponse toCategoryResponse(Category category);

    List<CategoryResponse> toCategoryResponseList(List<Category> categories);

    // ── Product List Item ─────────────────────────────────────────────────────
    @Mapping(source = "id",            target = "productId")
    @Mapping(target = "thumbnailUrl",  ignore = true)
    @Mapping(target = "basePrice",     ignore = true)
    @Mapping(target = "inStock",       ignore = true)
    @Mapping(target = "stockAvailable", ignore = true)
    @Mapping(target = "summary",       source = "description")
    ProductListItemResponse toListItemResponse(Product product);

    List<ProductListItemResponse> toListItemResponseList(List<Product> products);

    // ── Product Detail ────────────────────────────────────────────────────────
    @Mapping(source = "id",       target = "productId")
    @Mapping(target = "imageUrls", ignore = true)
    @Mapping(target = "variants",  ignore = true)
    ProductDetailResponse toDetailResponse(Product product);

    // ── Variant ───────────────────────────────────────────────────────────────
    @Mapping(source = "id",          target = "variantId")
    @Mapping(source = "variantName", target = "name")
    @Mapping(target = "price",       ignore = true)
    @Mapping(target = "stockAvailable", ignore = true)
    ProductVariantResponse toVariantResponse(ProductVariant variant);

    List<ProductVariantResponse> toVariantResponseList(List<ProductVariant> variants);

    // ── Money ─────────────────────────────────────────────────────────────────
    @Mapping(source = "amountUnits", target = "units")
    @Mapping(source = "amountNanos", target = "nanos")
    MoneyResponse toMoneyResponse(ChannelPrice price);

    // ── Admin ─────────────────────────────────────────────────────────────────
    @Mapping(source = "id",            target = "productId")
    @Mapping(source = "listingStatus", target = "saleStatus")
    @Mapping(target = "saleStatusLabel", ignore = true)
    @Mapping(target = "skuCount",      ignore = true)
    @Mapping(target = "onSaleSkuCount",ignore = true)
    @Mapping(target = "basePriceFromVnd", ignore = true)
    @Mapping(target = "foodInformationComplete", ignore = true)
    @Mapping(target = "updatedAt",     ignore = true)
    @Mapping(source = "version",       target = "revision")
    @Mapping(target = "categoryName",  ignore = true)
    com.hvduong.catalog.application.dto.response.AdminProductListItemResponse toAdminListItemResponse(Product product);

    List<com.hvduong.catalog.application.dto.response.AdminProductListItemResponse> toAdminListItemResponseList(List<Product> products);

    @Mapping(source = "id",            target = "productId")
    @Mapping(source = "listingStatus", target = "saleStatus")
    @Mapping(target = "saleStatusLabel", ignore = true)
    @Mapping(target = "skuCount",      ignore = true)
    @Mapping(target = "onSaleSkuCount",ignore = true)
    @Mapping(target = "basePriceFromVnd", ignore = true)
    @Mapping(target = "foodInformationComplete", ignore = true)
    @Mapping(target = "updatedAt",     ignore = true)
    @Mapping(source = "version",       target = "revision")
    @Mapping(target = "categoryName",  ignore = true)
    @Mapping(target = "shortDescription", source = "description")
    @Mapping(target = "longDescription",  ignore = true)
    @Mapping(target = "coverImageUrl",    ignore = true)
    @Mapping(target = "coverImageAlt",    ignore = true)
    @Mapping(target = "foodInformation",  ignore = true)
    @Mapping(target = "skus",             ignore = true)
    com.hvduong.catalog.application.dto.response.AdminProductDetailResponse toAdminDetailResponse(Product product);

    @Mapping(source = "id",            target = "skuId")
    @Mapping(source = "listingStatus", target = "saleStatus")
    @Mapping(source = "variantName",   target = "label")
    @Mapping(source = "weightValue",   target = "weightGrams")
    @Mapping(source = "packagingType", target = "packageType")
    @Mapping(target = "basePriceVnd",  ignore = true)
    @Mapping(target = "currency",      ignore = true)
    @Mapping(target = "saleStatusLabel", ignore = true)
    @Mapping(target = "updatedAt",     ignore = true)
    @Mapping(target = "revision",      ignore = true)
    com.hvduong.catalog.application.dto.response.AdminProductDetailResponse.AdminProductSkuResponse toAdminSkuResponse(ProductVariant variant);

    List<com.hvduong.catalog.application.dto.response.AdminProductDetailResponse.AdminProductSkuResponse> toAdminSkuResponseList(List<ProductVariant> variants);

}
