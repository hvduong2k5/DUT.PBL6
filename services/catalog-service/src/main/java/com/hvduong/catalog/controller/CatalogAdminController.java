package com.hvduong.catalog.controller;

import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.ProductTransitionRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.service.CatalogAdminService;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.common.response.ApiResponse;
import com.hvduong.catalog.common.response.PageResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@Slf4j
@RestController
@RequestMapping("/api/v1/admin/catalog")
@RequiredArgsConstructor
@SecurityRequirement(name = "bearerAuth")
@Tag(name = "Catalog Admin", description = "Các API quản trị sản phẩm, biến thể, bảng giá và phê duyệt kinh doanh sản phẩm OCOP")
public class CatalogAdminController {

    private final CatalogAdminService adminService;
    private final CatalogService catalogService;

    private String resolveActorId(Jwt jwt, String actorHeader) {
        if (actorHeader != null && !actorHeader.isBlank()) {
            return actorHeader;
        }
        if (jwt != null && jwt.getSubject() != null) {
            return jwt.getSubject();
        }
        return "system-admin";
    }

    @Operation(summary = "Xem danh sách sản phẩm quản trị", description = "Xem toàn bộ sản phẩm trên hệ thống bất kể trạng thái")
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('CATALOG_VIEW')")
    @GetMapping("/products")
    public ResponseEntity<ApiResponse<PageResponse<AdminProductListItemResponse>>> listAdminProducts(
            @RequestParam(required = false, defaultValue = "1") int page,
            @RequestParam(required = false, defaultValue = "20") int pageSize
    ) {
        log.info("[REST-ADMIN] listAdminProducts: page={}, pageSize={}", page, pageSize);
        PageResponse<AdminProductListItemResponse> result = catalogService.getAdminProducts(page, pageSize);
        return ResponseEntity.ok(ApiResponse.success(result));
    }

    @Operation(summary = "Xem chi tiết sản phẩm CMS", description = "Xem chi tiết sản phẩm kèm thông tin kiểm duyệt, phiên bản và các biến thể")
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('CATALOG_VIEW')")
    @GetMapping("/products/{productId}")
    public ResponseEntity<ApiResponse<AdminProductDetailResponse>> getAdminProduct(
            @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable String productId
    ) {
        log.info("[REST-ADMIN] getAdminProduct: id={}", productId);
        AdminProductDetailResponse result = catalogService.getAdminProductDetail(productId);
        return ResponseEntity.ok(ApiResponse.success(result));
    }

    @Operation(summary = "Tạo mới một sản phẩm", description = "Khởi tạo sản phẩm mới ở trạng thái DRAFT")
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('CATALOG_MANAGE')")
    @PostMapping("/products")
    public ResponseEntity<ApiResponse<ProductDetailResponse>> createProduct(
            @Valid @RequestBody CreateProductRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        String actorId = resolveActorId(jwt, actorHeader);
        log.info("[REST-ADMIN] createProduct: actor={}", actorId);
        ProductDetailResponse result = adminService.createProduct(request, actorId);
        return ResponseEntity.ok(ApiResponse.success("Tạo sản phẩm thành công", result));
    }

    @Operation(summary = "Cập nhật thông tin sản phẩm", description = "Cập nhật tên, mô tả, sao OCOP và các thông tin cơ bản")
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('CATALOG_MANAGE')")
    @PutMapping("/products/{productId}")
    public ResponseEntity<ApiResponse<ProductDetailResponse>> updateProduct(
            @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
            @Valid @RequestBody UpdateProductRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        String actorId = resolveActorId(jwt, actorHeader);
        log.info("[REST-ADMIN] updateProduct: id={}, actor={}", productId, actorId);
        ProductDetailResponse result = adminService.updateProduct(productId, request, actorId);
        return ResponseEntity.ok(ApiResponse.success("Cập nhật sản phẩm thành công", result));
    }

    @Operation(summary = "Chuyển trạng thái vòng đời sản phẩm", description = "Hỗ trợ các chuyển đổi: SUBMIT_FOR_REVIEW, APPROVE, REJECT, SUSPEND, ARCHIVE")
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('CATALOG_APPROVE')")
    @PostMapping("/products/{productId}/transition")
    public ResponseEntity<ApiResponse<ProductDetailResponse>> transitionProduct(
            @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
            @Valid @RequestBody ProductTransitionRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        String actorId = resolveActorId(jwt, actorHeader);
        long version = request.getExpectedVersion();
        String transition = request.getTransition().toUpperCase();

        log.info("[REST-ADMIN] transitionProduct: id={}, transition={}, actor={}", productId, transition, actorId);

        ProductDetailResponse result = switch (transition) {
            case "SUBMIT_FOR_REVIEW" -> adminService.submitProductForReview(productId, actorId, version);
            case "APPROVE" -> adminService.approveProduct(productId, actorId, version);
            case "REJECT" -> adminService.rejectProduct(productId, actorId, version);
            case "SUSPEND" -> adminService.suspendProduct(productId, actorId, version);
            case "ARCHIVE" -> adminService.archiveProduct(productId, actorId, version);
            default -> throw new IllegalArgumentException("Hành động chuyển đổi không được hỗ trợ: " + transition);
        };

        return ResponseEntity.ok(ApiResponse.success("Chuyển trạng thái sản phẩm thành công", result));
    }

    @Operation(summary = "Thêm biến thể mới cho sản phẩm", description = "Tạo mã SKU và quy cách đóng gói mới")
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('CATALOG_MANAGE')")
    @PostMapping("/products/{productId}/variants")
    public ResponseEntity<ApiResponse<ProductDetailResponse>> addVariant(
            @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
            @Valid @RequestBody CreateVariantRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        String actorId = resolveActorId(jwt, actorHeader);
        log.info("[REST-ADMIN] addVariant: productId={}, actor={}", productId, actorId);
        ProductDetailResponse result = adminService.addVariant(productId, request, actorId);
        return ResponseEntity.ok(ApiResponse.success("Thêm biến thể thành công", result));
    }

    @Operation(summary = "Cập nhật bảng giá bán của biến thể", description = "Cập nhật giá bán theo từng kênh (D2C, Shopee, TikTok...)")
    @PreAuthorize("hasRole('ADMIN') or hasAuthority('CATALOG_MANAGE')")
    @PutMapping("/products/{productId}/variants/{variantId}/prices")
    public ResponseEntity<ApiResponse<ProductDetailResponse>> updateVariantPrices(
            @Parameter(description = "ID sản phẩm (UUID)", required = true) @PathVariable UUID productId,
            @Parameter(description = "ID biến thể (UUID)", required = true) @PathVariable UUID variantId,
            @Valid @RequestBody UpdatePriceRequest request,
            @AuthenticationPrincipal Jwt jwt,
            @RequestHeader(value = "X-Actor-Principal-Id", required = false) String actorHeader
    ) {
        String actorId = resolveActorId(jwt, actorHeader);
        log.info("[REST-ADMIN] updateVariantPrices: productId={}, variantId={}, actor={}", productId, variantId, actorId);
        ProductDetailResponse result = adminService.updateVariantPrices(productId, variantId, request, actorId);
        return ResponseEntity.ok(ApiResponse.success("Cập nhật bảng giá thành công", result));
    }
}

