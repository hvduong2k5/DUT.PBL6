package com.hvduong.catalog.api;

import com.hvduong.catalog.application.dto.request.CreateCategoryRequest;
import com.hvduong.catalog.application.dto.request.CreatePriceScheduleRequest;
import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.PatchCategoryRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.ProductTransitionRequest;
import com.hvduong.catalog.application.dto.request.ReviewProductImageRequest;
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
import com.hvduong.catalog.application.service.CatalogAdminService;
import com.hvduong.catalog.common.response.BaseResponse;

import java.util.List;
import java.util.UUID;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
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

  @Operation(
      summary = "Lấy danh sách toàn bộ danh mục sản phẩm phục vụ quản trị CMS",
      description =
          "Hỗ trợ duyệt cây (format=tree, mặc định) hoặc danh sách phẳng (format=flat). Khi tìm kiếm theo q, yêu cầu format=flat. Yêu cầu quyền CATALOG_VIEW.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Lấy danh sách danh mục thành công"),
    @ApiResponse(responseCode = "400", description = "Dữ liệu đầu vào không hợp lệ"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "500", description = "Lỗi hệ thống nội bộ")
  })
  @PreAuthorize("hasAuthority('CATALOG_VIEW')")
  @GetMapping("/categories")
  public ResponseEntity<BaseResponse<AdminCategoryListResponse>> getAdminCategories(
      @Parameter(description = "Chế độ hiển thị: tree hoặc flat (mặc định: tree)")
          @RequestParam(required = false, defaultValue = "tree")
          String format,
      @Parameter(description = "Từ khóa tìm kiếm theo tên hoặc slug (yêu cầu format=flat)")
          @RequestParam(required = false)
          String q,
      @Parameter(description = "Lọc theo trạng thái kích hoạt (true/false)")
          @RequestParam(required = false)
          Boolean isActive,
      @Parameter(description = "ID danh mục cha (UUID)") @RequestParam(required = false)
          UUID parentId) {
    log.info(
        "[REST-ADMIN] getAdminCategories: format={}, q={}, isActive={}, parentId={}",
        format,
        q,
        isActive,
        parentId);
    AdminCategoryListResponse result =
        adminService.getAdminCategories(format, q, isActive, parentId);
    return ResponseEntity.ok(BaseResponse.success("Lấy danh sách danh mục thành công", result));
  }

  @Operation(
      summary = "Tạo danh mục sản phẩm mới",
      description =
          "Tạo một danh mục sản phẩm đang hoạt động. Yêu cầu quyền CATALOG_MANAGE#GLOBAL.")
  @ApiResponses({
    @ApiResponse(responseCode = "201", description = "Tạo danh mục thành công"),
    @ApiResponse(responseCode = "400", description = "Dữ liệu không hợp lệ"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Danh mục cha không tồn tại"),
    @ApiResponse(responseCode = "409", description = "Xung đột Idempotency hoặc trùng lặp slug")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE#GLOBAL')")
  @PostMapping("/categories")
  public ResponseEntity<BaseResponse<CategoryResponse>> createCategory(
      @Valid @RequestBody CreateCategoryRequest request,
      @RequestHeader("Idempotency-Key") String key,
      @AuthenticationPrincipal Jwt jwt) {
    return ResponseEntity.status(201)
        .body(BaseResponse.success(adminService.createCategory(request, jwt.getSubject(), key)));
  }

  @Operation(
      summary = "Cập nhật danh mục sản phẩm",
      description = "Chỉnh sửa tên, slug, mô tả, ảnh đại diện, vị trí và trạng thái của danh mục.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Cập nhật danh mục thành công"),
    @ApiResponse(responseCode = "400", description = "Dữ liệu không hợp lệ"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy danh mục"),
    @ApiResponse(
        responseCode = "409",
        description = "Xung đột phiên bản (optimistic lock) hoặc trùng lặp slug")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PutMapping("/categories/{categoryId}")
  public ResponseEntity<BaseResponse<CategoryResponse>> updateCategory(
      @Parameter(description = "ID danh mục (UUID)", required = true) @PathVariable UUID categoryId,
      @Valid @RequestBody UpdateCategoryRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] updateCategory: categoryId={}, actor={}", categoryId, actorId);
    CategoryResponse result = adminService.updateCategory(categoryId, request, actorId);
    return ResponseEntity.ok(BaseResponse.success("Cập nhật danh mục thành công", result));
  }

  @Operation(
      summary = "Xem chi tiết danh mục sản phẩm",
      description = "Lấy thông tin chi tiết một danh mục theo ID.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Lấy chi tiết danh mục thành công"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy danh mục")
  })
  @PreAuthorize("hasAuthority('CATALOG_VIEW')")
  @GetMapping("/categories/{categoryId}")
  public ResponseEntity<BaseResponse<CategoryResponse>> getCategory(
      @Parameter(description = "ID danh mục (UUID)", required = true) @PathVariable
          UUID categoryId) {
    return ResponseEntity.ok(BaseResponse.success(adminService.getCategory(categoryId)));
  }

  @Operation(
      summary = "Cập nhật một phần danh mục sản phẩm",
      description =
          "Chỉnh sửa một hoặc nhiều trường của danh mục (tên, slug, ảnh, mô tả, vị trí, kích hoạt) mà không cần gửi toàn bộ.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Cập nhật danh mục thành công"),
    @ApiResponse(responseCode = "400", description = "Dữ liệu không hợp lệ"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy danh mục"),
    @ApiResponse(
        responseCode = "409",
        description = "Xung đột phiên bản (optimistic lock) hoặc trùng lặp slug")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PatchMapping("/categories/{categoryId}")
  public ResponseEntity<BaseResponse<CategoryResponse>> patchCategory(
      @Parameter(description = "ID danh mục (UUID)", required = true) @PathVariable UUID categoryId,
      @Valid @RequestBody PatchCategoryRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] patchCategory: categoryId={}, actor={}", categoryId, actorId);
    CategoryResponse result = adminService.patchCategory(categoryId, request, actorId);
    return ResponseEntity.ok(BaseResponse.success("Cập nhật danh mục thành công", result));
  }

  @Operation(
      summary = "Xóa danh mục sản phẩm",
      description = "Xóa một danh mục chưa có sản phẩm và chưa có danh mục con gán vào.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Xóa danh mục thành công"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy danh mục"),
    @ApiResponse(responseCode = "409", description = "Danh mục đang có sản phẩm hoặc danh mục con")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @DeleteMapping("/categories/{categoryId}")
  public ResponseEntity<BaseResponse<Void>> deleteCategory(
      @Parameter(description = "ID danh mục (UUID)", required = true) @PathVariable UUID categoryId,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] deleteCategory: categoryId={}, actor={}", categoryId, actorId);
    adminService.deleteCategory(categoryId, actorId);
    return ResponseEntity.ok(BaseResponse.success("Xóa danh mục thành công", null));
  }

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
  public ResponseEntity<BaseResponse<AdminProductListResponse>> listAdminProducts(
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
    return ResponseEntity.ok(BaseResponse.success(result));
  }

  @Operation(
      summary = "Xem chi tiết sản phẩm CMS",
      description = "Xem chi tiết sản phẩm kèm thông tin kiểm duyệt, phiên bản và các biến thể")
  @PreAuthorize("hasAuthority('CATALOG_VIEW')")
  @GetMapping("/products/{productId}")
  public ResponseEntity<BaseResponse<AdminProductDetailResponse>> getAdminProduct(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable
          String productId) {
    log.info("[REST-ADMIN] getAdminProduct: id={}", productId);
    AdminProductDetailResponse result = adminService.getAdminProductDetail(productId);
    return ResponseEntity.ok(BaseResponse.success(result));
  }

  @Operation(
      summary = "Tạo mới một sản phẩm",
      description = "Khởi tạo sản phẩm mới ở trạng thái DRAFT")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PostMapping("/products")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> createProduct(
      @Valid @RequestBody CreateProductRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] createProduct: actor={}", actorId);
    ProductDetailResponse result = adminService.createProduct(request, actorId);
    return ResponseEntity.ok(BaseResponse.success("Tạo sản phẩm thành công", result));
  }

  @Operation(
      summary = "Cập nhật thông tin sản phẩm",
      description = "Cập nhật tên, mô tả, sao OCOP và các thông tin cơ bản")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PutMapping("/products/{productId}")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> updateProduct(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Valid @RequestBody UpdateProductRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] updateProduct: id={}, actor={}", productId, actorId);
    ProductDetailResponse result = adminService.updateProduct(productId, request, actorId);
    return ResponseEntity.ok(BaseResponse.success("Cập nhật sản phẩm thành công", result));
  }

  @Operation(
      summary = "Cập nhật một phần nội dung sản phẩm",
      description =
          "PATCH cho phép chỉnh sửa mọi thông tin nội dung: tên, slug, danh mục, mô tả, thành phần, "
              + "hướng dẫn bảo quản, sao OCOP, số chứng nhận và câu chuyện thương hiệu. "
              + "Field không gửi lên sẽ giữ nguyên; chuỗi rỗng dùng để xóa field optional.")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PatchMapping("/products/{productId}")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> patchProduct(
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
    return ResponseEntity.ok(BaseResponse.success("Cập nhật nội dung sản phẩm thành công", result));
  }

  @Operation(
      summary = "Chuyển trạng thái vòng đời sản phẩm",
      description = "Hỗ trợ các chuyển đổi: SUBMIT_FOR_REVIEW, APPROVE, REJECT, SUSPEND, ARCHIVE")
  @PreAuthorize("hasAuthority('CATALOG_APPROVE')")
  @PostMapping("/products/{productId}/transition")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> transitionProduct(
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

    return ResponseEntity.ok(BaseResponse.success("Chuyển trạng thái sản phẩm thành công", result));
  }

  @Operation(
      summary = "Thêm SKU mới cho sản phẩm",
      description = "Tạo mã SKU và quy cách đóng gói mới")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PostMapping("/products/{productId}/skus")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> addVariant(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Valid @RequestBody CreateVariantRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] addVariant: productId={}, actor={}", productId, actorId);
    ProductDetailResponse result = adminService.addVariant(productId, request, actorId);
    return ResponseEntity.ok(BaseResponse.success("Thêm biến thể thành công", result));
  }

  @Operation(
      summary = "Cập nhật một phần thông tin SKU",
      description =
          "Chỉnh sửa tên biến thể, khối lượng, đơn vị, hương vị, loại bao bì, hạn sử dụng và "
              + "trạng thái bán. Mã SKU là bất biến, không thể đổi qua API này.")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PatchMapping("/products/{productId}/skus/{skuId}")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> patchVariant(
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
    return ResponseEntity.ok(BaseResponse.success("Cập nhật SKU thành công", result));
  }

  @Operation(
      summary = "Xem chi tiết một SKU",
      description = "Lấy thông tin chi tiết của một biến thể sản phẩm theo ID.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Lấy chi tiết SKU thành công"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy SKU hoặc sản phẩm")
  })
  @PreAuthorize("hasAuthority('CATALOG_VIEW')")
  @GetMapping("/products/{productId}/skus/{skuId}")
  public ResponseEntity<BaseResponse<ProductVariantResponse>> getVariant(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID SKU (UUID)", required = true) @PathVariable UUID skuId) {
    log.info("[REST-ADMIN] getVariant: productId={}, skuId={}", productId, skuId);
    return ResponseEntity.ok(BaseResponse.success(adminService.getVariant(productId, skuId)));
  }

  @Operation(
      summary = "Xem lịch sử giá của SKU",
      description = "Trả toàn bộ bản ghi giá theo mọi kênh bán, gồm ACTIVE, SUPERSEDED và EXPIRED.")
  @PreAuthorize("hasAuthority('CATALOG_VIEW')")
  @GetMapping("/products/{productId}/skus/{skuId}/prices")
  public ResponseEntity<BaseResponse<List<AdminPriceResponse>>> getVariantPrices(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID SKU (UUID)", required = true) @PathVariable UUID skuId) {
    log.info("[REST-ADMIN] getVariantPrices: productId={}, skuId={}", productId, skuId);
    return ResponseEntity.ok(BaseResponse.success(adminService.getVariantPrices(productId, skuId)));
  }

  @Operation(
      summary = "Thêm ảnh cho sản phẩm",
      description =
          "Thêm metadata ảnh (object key MinIO/S3). Nếu role=COVER và mediaStatus=APPROVED, "
              + "ảnh cover APPROVED cũ sẽ tự động được hạ xuống GALLERY.")
  @ApiResponses({
    @ApiResponse(responseCode = "201", description = "Thêm ảnh sản phẩm thành công"),
    @ApiResponse(responseCode = "400", description = "Dữ liệu ảnh không hợp lệ"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy sản phẩm")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PostMapping("/products/{productId}/images")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> addImage(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Valid @RequestBody CreateProductImageRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info("[REST-ADMIN] addImage: productId={}, actor={}", productId, actorId);
    ProductDetailResponse result = adminService.addImage(productId, request, actorId);
    return ResponseEntity.status(HttpStatus.CREATED)
        .body(BaseResponse.success("Thêm ảnh sản phẩm thành công", result));
  }

  @Operation(
      summary = "Cập nhật metadata hiển thị ảnh sản phẩm",
      description =
          "Cập nhật alt text, role, thứ tự hiển thị với optimistic locking. Không cho phép sửa mediaStatus hoặc scanStatus.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Cập nhật ảnh sản phẩm thành công"),
    @ApiResponse(responseCode = "400", description = "Dữ liệu cập nhật không hợp lệ"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy ảnh hoặc sản phẩm"),
    @ApiResponse(responseCode = "409", description = "Xung đột phiên bản (optimistic lock)")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PatchMapping("/products/{productId}/images/{imageId}")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> updateImage(
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
    return ResponseEntity.ok(BaseResponse.success("Cập nhật ảnh sản phẩm thành công", result));
  }

  @Operation(
      summary = "Kiểm duyệt nội dung ảnh sản phẩm",
      description =
          "Duyệt hoặc từ chối ảnh sản phẩm (Content Moderation). Yêu cầu quyền CATALOG_APPROVE. Ảnh phải có scanStatus là CLEAN và mediaStatus là PENDING_REVIEW.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Kiểm duyệt ảnh sản phẩm thành công"),
    @ApiResponse(responseCode = "400", description = "Dữ liệu hoặc trạng thái duyệt không hợp lệ"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện hoặc vi phạm SoD"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy ảnh hoặc sản phẩm"),
    @ApiResponse(responseCode = "409", description = "Xung đột phiên bản (optimistic lock)"),
    @ApiResponse(responseCode = "422", description = "Ảnh chưa được quét an toàn mã độc")
  })
  @PreAuthorize("hasAuthority('CATALOG_APPROVE')")
  @PostMapping("/products/{productId}/images/{imageId}/review")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> reviewImage(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID ảnh (UUID)", required = true) @PathVariable UUID imageId,
      @Valid @RequestBody ReviewProductImageRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info(
        "[REST-ADMIN] reviewImage: productId={}, imageId={}, decision={}, actor={}",
        productId,
        imageId,
        request.getDecision(),
        actorId);
    ProductDetailResponse result = adminService.reviewImage(productId, imageId, request, actorId);
    return ResponseEntity.ok(BaseResponse.success("Kiểm duyệt ảnh sản phẩm thành công", result));
  }

  @Operation(
      summary = "Xóa ảnh sản phẩm",
      description =
          "Xóa metadata ảnh khỏi catalog. Object trên storage cần được dọn bởi job riêng.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Xóa ảnh sản phẩm thành công"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy ảnh hoặc sản phẩm")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @DeleteMapping("/products/{productId}/images/{imageId}")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> deleteImage(
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
    return ResponseEntity.ok(BaseResponse.success("Xóa ảnh sản phẩm thành công", result));
  }

  @Operation(
      summary = "Cập nhật bảng giá bán của biến thể",
      description = "Cập nhật giá bán theo từng kênh (D2C, Shopee, TikTok...)")
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PutMapping("/products/{productId}/skus/{skuId}/prices")
  public ResponseEntity<BaseResponse<ProductDetailResponse>> updateVariantPrices(
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
    return ResponseEntity.ok(BaseResponse.success("Cập nhật bảng giá thành công", result));
  }

  @Operation(
      summary = "Lên lịch hoặc tạo bản ghi giá mới theo khoảng hiệu lực cho SKU",
      description =
          "Tạo bản ghi giá trong khoảng [effectiveFrom, effectiveTo). Nếu bị overlap với khoảng giá đã có cùng kênh, trả về 409 Conflict.")
  @ApiResponses({
    @ApiResponse(responseCode = "201", description = "Đã tạo và lên lịch giá thành công"),
    @ApiResponse(responseCode = "400", description = "Dữ liệu đầu vào không hợp lệ"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(responseCode = "404", description = "Không tìm thấy sản phẩm hoặc biến thể"),
    @ApiResponse(
        responseCode = "409",
        description = "Khoảng hiệu lực trùng lặp (GiST Exclusion Constraint)"),
    @ApiResponse(responseCode = "422", description = "Lỗi nghiệp vụ không thỏa mãn điều kiện")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @PostMapping("/products/{productId}/skus/{skuId}/prices")
  public ResponseEntity<BaseResponse<AdminPriceResponse>> schedulePrice(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID SKU (UUID)", required = true) @PathVariable UUID skuId,
      @Valid @RequestBody CreatePriceScheduleRequest request,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info(
        "[REST-ADMIN] schedulePrice: productId={}, skuId={}, channel={}, actor={}",
        productId,
        skuId,
        request.getChannel(),
        actorId);
    AdminPriceResponse result = adminService.schedulePrice(productId, skuId, request, actorId);
    return ResponseEntity.status(HttpStatus.CREATED)
        .body(BaseResponse.success("Lên lịch giá thành công", result));
  }

  @Operation(
      summary = "Hủy bản ghi giá đã lên lịch trong tương lai",
      description =
          "Chỉ cho phép hủy/xóa bản ghi giá chưa tới thời điểm hiệu lực (effectiveFrom > now). Giá đang hoặc đã từng có hiệu lực không được xóa.")
  @ApiResponses({
    @ApiResponse(responseCode = "200", description = "Hủy bản ghi giá đã lên lịch thành công"),
    @ApiResponse(responseCode = "401", description = "Chưa xác thực"),
    @ApiResponse(responseCode = "403", description = "Không có quyền thực hiện"),
    @ApiResponse(
        responseCode = "404",
        description = "Không tìm thấy bản ghi giá, sản phẩm hoặc biến thể"),
    @ApiResponse(
        responseCode = "422",
        description = "Không thể hủy bản ghi giá đang hoặc đã từng có hiệu lực")
  })
  @PreAuthorize("hasAuthority('CATALOG_MANAGE')")
  @DeleteMapping("/products/{productId}/skus/{skuId}/prices/{priceId}")
  public ResponseEntity<BaseResponse<Void>> deleteScheduledPrice(
      @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
      @Parameter(description = "ID SKU (UUID)", required = true) @PathVariable UUID skuId,
      @Parameter(description = "ID bản ghi giá (UUID)", required = true) @PathVariable UUID priceId,
      @AuthenticationPrincipal Jwt jwt,
      @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader) {
    String actorId = resolveActorId(jwt, actorHeader);
    log.info(
        "[REST-ADMIN] deleteScheduledPrice: productId={}, skuId={}, priceId={}, actor={}",
        productId,
        skuId,
        priceId,
        actorId);
    adminService.deleteScheduledPrice(productId, skuId, priceId, actorId);
    return ResponseEntity.ok(BaseResponse.success("Hủy bản ghi giá thành công", null));
  }
}
