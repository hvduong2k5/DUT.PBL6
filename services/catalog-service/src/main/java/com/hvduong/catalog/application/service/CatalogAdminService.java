package com.hvduong.catalog.application.service;

import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;

import java.util.UUID;

/**
 * Application Service interface cho các chức năng admin catalog.
 * Không phụ thuộc gRPC/protobuf.
 */
public interface CatalogAdminService {

    ProductDetailResponse createProduct(CreateProductRequest request, String actorId);

    ProductDetailResponse updateProduct(UUID productId, UpdateProductRequest request, String actorId);

    ProductDetailResponse submitProductForReview(UUID productId, String actorId, long expectedVersion);

    ProductDetailResponse approveProduct(UUID productId, String approverActorId, long expectedVersion);

    ProductDetailResponse rejectProduct(UUID productId, String approverActorId, long expectedVersion);

    ProductDetailResponse suspendProduct(UUID productId, String actorId, long expectedVersion);

    ProductDetailResponse archiveProduct(UUID productId, String actorId, long expectedVersion);

    ProductDetailResponse addVariant(UUID productId, CreateVariantRequest request, String actorId);

    ProductDetailResponse updateVariantPrices(UUID productId, UUID variantId,
                                              UpdatePriceRequest request, String actorId);
}
