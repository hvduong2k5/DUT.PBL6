package com.hvduong.catalog.application.service;

import com.hvduong.catalog.application.dto.request.CreateCategoryRequest;
import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductImageRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminPriceResponse;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;

import java.util.List;
import java.util.UUID;

/** Application Service interface cho các chức năng admin catalog. Không phụ thuộc gRPC/protobuf. */
public interface CatalogAdminService {

  // ── Read ────────────────────────────────────────────────────────
  AdminProductListResponse getAdminProducts(int page, int pageSize, String q, String status);

  AdminProductDetailResponse getAdminProductDetail(String productId);

  // ── Write ───────────────────────────────────────────────────────
  CategoryResponse createCategory(CreateCategoryRequest request, String actorId, String key);

  ProductDetailResponse createProduct(CreateProductRequest request, String actorId);

  ProductDetailResponse updateProduct(UUID productId, UpdateProductRequest request, String actorId);

  /** PATCH: cập nhật một phần nội dung sản phẩm (name, slug, story, OCOP, mô tả...). */
  ProductDetailResponse patchProduct(UUID productId, PatchProductRequest request, String actorId);

  ProductDetailResponse submitProductForReview(
      UUID productId, String actorId, long expectedVersion);

  ProductDetailResponse approveProduct(
      UUID productId, String approverActorId, long expectedVersion);

  ProductDetailResponse rejectProduct(UUID productId, String approverActorId, long expectedVersion);

  ProductDetailResponse suspendProduct(UUID productId, String actorId, long expectedVersion);

  ProductDetailResponse archiveProduct(UUID productId, String actorId, long expectedVersion);

  ProductDetailResponse addVariant(UUID productId, CreateVariantRequest request, String actorId);

  /** PATCH: cập nhật một phần thông tin SKU (tên, khối lượng, bao bì, hạn dùng, trạng thái). */
  ProductDetailResponse patchVariant(
      UUID productId, UUID variantId, PatchVariantRequest request, String actorId);

  /** Lịch sử giá của một SKU trên mọi kênh bán. */
  List<AdminPriceResponse> getVariantPrices(UUID productId, UUID variantId);

  // ── Images ──────────────────────────────────────────────────────
  ProductDetailResponse addImage(UUID productId, CreateProductImageRequest request, String actorId);

  ProductDetailResponse updateImage(
      UUID productId, UUID imageId, UpdateProductImageRequest request, String actorId);

  ProductDetailResponse deleteImage(UUID productId, UUID imageId, String actorId);

  ProductDetailResponse updateVariantPrices(
      UUID productId, UUID variantId, UpdatePriceRequest request, String actorId);
}
