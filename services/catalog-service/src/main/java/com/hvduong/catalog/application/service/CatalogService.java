package com.hvduong.catalog.application.service;

import com.hvduong.catalog.application.dto.request.ProductFilterRequest;
import com.hvduong.catalog.application.dto.request.PriceValidationRequest;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.response.PageResponse;

import java.util.List;

/**
 * Application Service interface cho các chức năng public catalog.
 * Chỉ trả Product APPROVED + ACTIVE. Không phụ thuộc gRPC/protobuf.
 */
public interface CatalogService {

    PageResponse<ProductListItemResponse> getPublicProducts(ProductFilterRequest filter);

    ProductDetailResponse getPublicProductDetail(String idOrSlug, SalesChannel channel);

    PageResponse<ProductListItemResponse> searchProducts(String query, int page, int pageSize);

    List<CategoryResponse> getAllCategories();

    ProductVariantResponse getVariantBySkuCode(String skuCode);

    PriceValidationResponse validatePrices(PriceValidationRequest request);

    PageResponse<com.hvduong.catalog.application.dto.response.AdminProductListItemResponse> getAdminProducts(int page, int pageSize);

    com.hvduong.catalog.application.dto.response.AdminProductDetailResponse getAdminProductDetail(String productId);
}
