package com.hvduong.catalog.application.service;

import com.hvduong.catalog.application.dto.request.CreateCategoryRequest;
import com.hvduong.catalog.application.dto.request.CreatePriceScheduleRequest;
import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.PatchCategoryRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.UpdateCategoryRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductImageRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminCategoryListResponse;
import com.hvduong.catalog.application.dto.response.AdminPriceResponse;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;

import java.util.List;
import java.util.UUID;

/** Application Service interface cho các chức năng admin catalog. Không phụ thuộc gRPC/protobuf. */
public interface CatalogAdminService {

  // ── Read ────────────────────────────────────────────────────────
  AdminProductListResponse getAdminProducts(int page, int pageSize, String q, String status);

  AdminProductDetailResponse getAdminProductDetail(String productId);

  /** Tra cứu danh sách danh mục phân cấp (tree) hoặc phẳng (flat) cho Admin CMS. */
  AdminCategoryListResponse getAdminCategories(
      String format, String q, Boolean isActive, UUID parentId);

  CategoryResponse getCategory(UUID categoryId);

  ProductVariantResponse getVariant(UUID productId, UUID variantId);

  // ── Write ───────────────────────────────────────────────────────
  CategoryResponse createCategory(CreateCategoryRequest request, String actorId, String key);

  CategoryResponse updateCategory(UUID categoryId, UpdateCategoryRequest request, String actorId);

  CategoryResponse patchCategory(UUID categoryId, PatchCategoryRequest request, String actorId);

  void deleteCategory(UUID categoryId, String actorId);

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

  /** Lên lịch hoặc tạo bản ghi giá mới theo khoảng thời gian hiệu lực cho SKU. */
  AdminPriceResponse schedulePrice(
      UUID productId, UUID skuId, CreatePriceScheduleRequest request, String actorId);

  /** Hủy bản ghi giá đã lên lịch trong tương lai (chưa tới effective_from). */
  void deleteScheduledPrice(UUID productId, UUID skuId, UUID priceId, String actorId);
}
