package com.hvduong.catalog.application.mapper;

import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.common.enums.ListingStatus;
import com.hvduong.catalog.domain.entity.Category;
import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductVariant;
import org.mapstruct.Mapper;
import org.mapstruct.Mapping;
import org.mapstruct.Named;

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
    @Mapping(source = "listingStatus", target = "inStock", qualifiedByName = "listingToInStock")
    @Mapping(target = "thumbnailUrl",  ignore = true)
    @Mapping(target = "basePrice",     ignore = true)
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
    ProductVariantResponse toVariantResponse(ProductVariant variant);

    List<ProductVariantResponse> toVariantResponseList(List<ProductVariant> variants);

    // ── Money ─────────────────────────────────────────────────────────────────
    @Mapping(source = "amountUnits", target = "units")
    @Mapping(source = "amountNanos", target = "nanos")
    MoneyResponse toMoneyResponse(ChannelPrice price);

    // ── Helpers ───────────────────────────────────────────────────────────────
    @Named("listingToInStock")
    default boolean listingStatusToInStock(ListingStatus status) {
        return status == ListingStatus.ACTIVE;
    }
}
