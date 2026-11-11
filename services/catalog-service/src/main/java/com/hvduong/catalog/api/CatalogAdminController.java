package com.hvduong.catalog.api;

import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.ProductTransitionRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductImageRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminPriceResponse;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.service.CatalogAdminService;
import com.hvduong.catalog.common.response.ApiResponse;

import java.util.List;
import java.util.UUID;

import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.*;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

@Slf4j
@RestController
@RequestMapping("/api/v1/admin/catalog")
@RequiredArgsConstructor
@SecurityRequirement(name = "bearerAuth")
@Tag(
    name = "Catalog Admin",
    description =
        "Các API quản trị sản phẩm, biến thể, bảng giá và phê duyệt kinh doanh sản phẩm OCOP")
public class CatalogAdminController {

  private final CatalogAdminService adminService;

  private String resolveActorId(Jwt jwt, String actorHeader) {
    if (actorHeader != null && !actorHeader.isBlank()) {
      return actorHeader;
    }
    if (jwt != null && jwt.getSubject() != null) {
      return jwt.getSubject();
    }
    return "system-admin";
  }

  @Operation(
      summary = "Xem danh sách sản phẩm quản trị",
      description = "Xem toàn bộ sản phẩm trên hệ thống bất kể trạng thái")
  @PreAuthorize("hasAuthority('CATALOG_VIEW')")
  @GetMapping("/products")
  public ResponseEntity<ApiResponse<AdminProductListResponse>> listAdminProducts(
      @RequestParam(required = false, defaultValue = "1") int page,
      @RequestParam(required = false, defaultValue = "20") int pageSize,
      @RequestParam(required = false) String q,
      @RequestParam(required = false) String status) {
    log.info(
        "[REST-ADMIN] listAdminProducts: page={}, pageSize={}, q={}, status={}",
        page,
        pageSize,
        q,
        status);
    AdminProductListResponse result = adminService.getAdminProducts(page, pageSize, q, status);
    return ResponseEntity.ok(ApiResponse.success(result));
  }

  @Operation(
      summary = "Xem chi tiết sản phẩm CMS",
      description = "Xem chi tiết sản phẩm kèm thông tin kiểm duyệt, phiên bản và các biến thể")
  @PreAuthorize("hasAuthority('CATALOG_VIEW')")
  @GetMapping("/products/{productId}")
  public ResponseEntity<ApiResponse<AdminProductDetailResponse>> getAdminProduct(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable
          String productId) {
    log.info("[REST-ADMIN] getAdminProduct: id={}", productId);
    AdminProductDetailResponse result = adminService.getAdminProductDetail(productId);
    return ResponseEntity.ok(ApiResponse.success(result));
  }

  @Operation(
      summary = "Tạo mới một sản phẩm",
      description = "Khởi tạo sản phẩm mới ở trạng thái DRAFT")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PostMapping("/products")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> createProduct(
      @Valid @RequestBody CreateProductRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] createProduct: actor={}", actorId);
    ProductDetailResponse result = adminService.createProduct(request, actorId);
    return ResponseEntity.ok(ApiResponse.success("Tạo sản phẩm thành công", result));
  }

  @Operation(
      summary = "Cập nhật thông tin sản phẩm",
      description = "Cập nhật tên, mô tả, sao OCOP và các thông tin cơ bản")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PutMapping("/products/{productId}")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> updateProduct(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Valid @RequestBody UpdateProductRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] updateProduct: id={}, actor={}", productId, actorId);
    ProductDetailResponse result = adminService.updateProduct(productId, request, actorId);
    return ResponseEntity.ok(ApiResponse.success("Cập nhật sản phẩm thành công", result));
  }

  @Operation(
      summary = "Cập nhật một phần nội dung sản phẩm",
      description =
          "PATCH cho phép chỉnh sửa mọi thông tin nội dung: tên, slug, danh mục, mô tả, thành phần, "
              + "hướng dẫn bảo quản, sao OCOP, số chứng nhận và câu chuyện thương hiệu. "
              + "Field không gửi lên sẽ giữ nguyên; chuỗi rỗng dùng để xóa field optional.")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PatchMapping("/products/{productId}")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> patchProduct(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Valid @RequestBody PatchProductRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info(
        "[REST-ADMIN] patchProduct: id={}, expectedVersion={}, actor={}",
        productId,
        request.getExpectedVersion(),
        actorId);
    ProductDetailResponse result = adminService.patchProduct(productId, request, actorId);
    return ResponseEntity.ok(ApiResponse.success("Cập nhật nội dung sản phẩm thành công", result));
  }

  @Operation(
      summary = "Chuyển trạng thái vòng đời sản phẩm",
      description = "Hỗ trợ các chuyển đổi: SUBMIT_FOR_REVIEW, APPROVE, REJECT, SUSPEND, ARCHIVE")
  @PreAuthorize("hasAuthority('CATALOG_APPROVE')")
  @PostMapping("/products/{productId}/transition")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> transitionProduct(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Valid @RequestBody ProductTransitionRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    long version = request.getExpectedVersion();
    String transition = request.getTransition().toUpperCase();

    log.info(
        "[REST-ADMIN] transitionProduct: id={}, transition={}, actor={}",
        productId,
        transition,
        actorId);

    ProductDetailResponse result =
        switch (transition) {
          case "SUBMIT_FOR_REVIEW" -> adminService.submitProductForReview(
              productId, actorId, version);
          case "APPROVE" -> adminService.approveProduct(productId, actorId, version);
          case "REJECT" -> adminService.rejectProduct(productId, actorId, version);
          case "SUSPEND" -> adminService.suspendProduct(productId, actorId, version);
          case "ARCHIVE" -> adminService.archiveProduct(productId, actorId, version);
          default -> throw new IllegalArgumentException(
              "Hành động chuyển đổi không được hỗ trợ: " + transition);
        };

    return ResponseEntity.ok(ApiResponse.success("Chuyển trạng thái sản phẩm thành công", result));
  }

  @Operation(
      summary = "Thêm SKU mới cho sản phẩm",
      description = "Tạo mã SKU và quy cách đóng gói mới")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PostMapping("/products/{productId}/skus")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> addVariant(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Valid @RequestBody CreateVariantRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] addVariant: productId={}, actor={}", productId, actorId);
    ProductDetailResponse result = adminService.addVariant(productId, request, actorId);
    return ResponseEntity.ok(ApiResponse.success("Thêm biến thể thành công", result));
  }

  @Operation(
      summary = "Cập nhật một phần thông tin SKU",
      description =
          "Chỉnh sửa tên biến thể, khối lượng, đơn vị, hương vị, loại bao bì, hạn sử dụng và "
              + "trạng thái bán. Mã SKU là bất biến, không thể đổi qua API này.")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PatchMapping("/products/{productId}/skus/{skuId}")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> patchVariant(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID SKU (UUID)", required = true) @PathVariable UUID skuId,
      @Valid @RequestBody PatchVariantRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info(
        "[REST-ADMIN] patchVariant: productId={}, skuId={}, expectedVersion={}, actor={}",
        productId,
        skuId,
        request.getExpectedVersion(),
        actorId);
    ProductDetailResponse result = adminService.patchVariant(productId, skuId, request, actorId);
    return ResponseEntity.ok(ApiResponse.success("Cập nhật SKU thành công", result));
  }

  @Operation(
      summary = "Xem lịch sử giá của SKU",
      description = "Trả toàn bộ bản ghi giá theo mọi kênh bán, gồm ACTIVE, SUPERSEDED và EXPIRED.")
  @PreAuthorize("hasAuthority('CATALOG_VIEW')")
  @GetMapping("/products/{productId}/skus/{skuId}/prices")
  public ResponseEntity<ApiResponse<List<AdminPriceResponse>>> getVariantPrices(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID SKU (UUID)", required = true) @PathVariable UUID skuId) {
    log.info("[REST-ADMIN] getVariantPrices: productId={}, skuId={}", productId, skuId);
    return ResponseEntity.ok(ApiResponse.success(adminService.getVariantPrices(productId, skuId)));
  }

  @Operation(
      summary = "Thêm ảnh cho sản phẩm",
      description =
          "Thêm metadata ảnh (object key MinIO/S3). Nếu role=COVER và mediaStatus=APPROVED, "
              + "ảnh cover APPROVED cũ sẽ tự động được hạ xuống GALLERY.")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PostMapping("/products/{productId}/images")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> addImage(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Valid @RequestBody CreateProductImageRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] addImage: productId={}, actor={}", productId, actorId);
    ProductDetailResponse result = adminService.addImage(productId, request, actorId);
    return ResponseEntity.ok(ApiResponse.success("Thêm ảnh sản phẩm thành công", result));
  }

  @Operation(
      summary = "Cập nhật metadata/trạng thái ảnh sản phẩm",
      description =
          "Cập nhật alt text, role, thứ tự và trạng thái kiểm duyệt ảnh với optimistic locking.")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PatchMapping("/products/{productId}/images/{imageId}")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> updateImage(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID ảnh (UUID)", required = true) @PathVariable UUID imageId,
      @Valid @RequestBody UpdateProductImageRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info(
        "[REST-ADMIN] updateImage: productId={}, imageId={}, actor={}",
        productId,
        imageId,
        actorId);
    ProductDetailResponse result = adminService.updateImage(productId, imageId, request, actorId);
    return ResponseEntity.ok(ApiResponse.success("Cập nhật ảnh sản phẩm thành công", result));
  }

  @Operation(
      summary = "Xóa ảnh sản phẩm",
      description =
          "Xóa metadata ảnh khỏi catalog. Object trên storage cần được dọn bởi job riêng.")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @DeleteMapping("/products/{productId}/images/{imageId}")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> deleteImage(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID ảnh (UUID)", required = true) @PathVariable UUID imageId,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info(
        "[REST-ADMIN] deleteImage: productId={}, imageId={}, actor={}",
        productId,
        imageId,
        actorId);
    ProductDetailResponse result = adminService.deleteImage(productId, imageId, actorId);
    return ResponseEntity.ok(ApiResponse.success("Xóa ảnh sản phẩm thành công", result));
  }

  @Operation(
      summary = "Cập nhật bảng giá bán của biến thể",
      description = "Cập nhật giá bán theo từng kênh (D2C, Shopee, TikTok...)")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PutMapping("/products/{productId}/skus/{skuId}/prices")
  public ResponseEntity<ApiResponse<ProductDetailResponse>> updateVariantPrices(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID SKU (UUID)", required = true) @PathVariable UUID skuId,
      @Valid @RequestBody UpdatePriceRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info(
        "[REST-ADMIN] updateVariantPrices: productId={}, skuId={}, actor={}",
        productId,
        skuId,
        actorId);
    ProductDetailResponse result =
        adminService.updateVariantPrices(productId, skuId, request, actorId);
    return ResponseEntity.ok(ApiResponse.success("Cập nhật bảng giá thành công", result));
  }
}
