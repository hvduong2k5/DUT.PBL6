package com.hvduong.catalog.controller;

import com.hvduong.catalog.application.dto.request.PriceValidationRequest;
import com.hvduong.catalog.application.dto.request.ProductFilterRequest;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.response.ApiResponse;
import com.hvduong.catalog.common.response.PageResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@Slf4j
@RestController
@RequestMapping("/api/v1")
@RequiredArgsConstructor
@Tag(name = "Catalog Public", description = "Các API tra cứu danh mục, sản phẩm và biến thể cho khách hàng và sàn TMĐT")
public class CatalogPublicController {

    private final CatalogService catalogService;

    @Operation(summary = "Lấy danh mục sản phẩm OCOP", description = "Trả về cây danh mục sản phẩm mè xửng và đặc sản Huế")
    @GetMapping("/categories")
    public ResponseEntity<ApiResponse<List<CategoryResponse>>> listCategories() {
        log.debug("[REST] listCategories");
        List<CategoryResponse> categories = catalogService.getAllCategories();
        return ResponseEntity.ok(ApiResponse.success(categories));
    }

    @Operation(summary = "Duyệt danh sách sản phẩm công khai", description = "Hỗ trợ lọc theo danh mục, giá, sao OCOP, sắp xếp và phân trang")
    @GetMapping("/products")
    public ResponseEntity<ApiResponse<PageResponse<ProductListItemResponse>>> listProducts(
            @Valid @ModelAttribute ProductFilterRequest filter
    ) {
        log.debug("[REST] listProducts: filter={}", filter);
        PageResponse<ProductListItemResponse> page = catalogService.getPublicProducts(filter);
        return ResponseEntity.ok(ApiResponse.success(page));
    }

    @Operation(summary = "Xem chi tiết một sản phẩm", description = "Truy vấn theo UUID sản phẩm hoặc slug SEO")
    @GetMapping("/products/{idOrSlug}")
    public ResponseEntity<ApiResponse<ProductDetailResponse>> getProduct(
            @Parameter(description = "ID hoặc Slug của sản phẩm", required = true) @PathVariable String idOrSlug,
            @Parameter(description = "Kênh bán hàng") @RequestParam(required = false, defaultValue = "D2C_WEB") SalesChannel channel
    ) {
        log.debug("[REST] getProduct: idOrSlug={}, channel={}", idOrSlug, channel);
        ProductDetailResponse detail = catalogService.getPublicProductDetail(idOrSlug, channel);
        return ResponseEntity.ok(ApiResponse.success(detail));
    }

    @Operation(summary = "Tìm kiếm sản phẩm", description = "Tìm kiếm theo từ khóa tên sản phẩm hoặc mô tả")
    @GetMapping("/products/search")
    public ResponseEntity<ApiResponse<PageResponse<ProductListItemResponse>>> searchProducts(
            @Parameter(description = "Từ khóa tìm kiếm", required = true) @RequestParam String q,
            @RequestParam(required = false, defaultValue = "1") int page,
            @RequestParam(required = false, defaultValue = "20") int pageSize
    ) {
        log.debug("[REST] searchProducts: query={}, page={}, pageSize={}", q, page, pageSize);
        PageResponse<ProductListItemResponse> result = catalogService.searchProducts(q, page, pageSize);
        return ResponseEntity.ok(ApiResponse.success(result));
    }

    @Operation(summary = "Lấy thông tin biến thể theo SKU", description = "Truy vấn quy cách đóng gói, khối lượng tịnh và giá niêm yết theo mã SKU")
    @GetMapping("/products/variants/{skuCode}")
    public ResponseEntity<ApiResponse<ProductVariantResponse>> getProductVariant(
            @Parameter(description = "Mã SKU biến thể", required = true) @PathVariable String skuCode
    ) {
        log.debug("[REST] getProductVariant: skuCode={}", skuCode);
        ProductVariantResponse variant = catalogService.getVariantBySkuCode(skuCode);
        return ResponseEntity.ok(ApiResponse.success(variant));
    }

    @Operation(summary = "Thẩm định giá bán và SKU (Nội bộ / Gateway)", description = "Đối chiếu giá Client gửi lên với giá niêm yết trong Database")
    @PostMapping("/internal/prices/validate")
    public ResponseEntity<ApiResponse<PriceValidationResponse>> validatePriceAndSku(
            @Valid @RequestBody PriceValidationRequest request
    ) {
        log.debug("[REST] validatePriceAndSku: items={}", request.getItems() != null ? request.getItems().size() : 0);
        PriceValidationResponse response = catalogService.validatePrices(request);
        return ResponseEntity.ok(ApiResponse.success(response));
    }
}

