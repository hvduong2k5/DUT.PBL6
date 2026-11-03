package com.hvduong.catalog.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.catalog.application.dto.request.PriceValidationRequest;
import com.hvduong.catalog.application.dto.response.*;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.handler.GlobalRestExceptionHandler;
import com.hvduong.catalog.common.response.PageResponse;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import java.util.Collections;
import java.util.List;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@ExtendWith(MockitoExtension.class)
class CatalogPublicControllerTest {

    private MockMvc mockMvc;

    @Mock
    private CatalogService catalogService;

    @InjectMocks
    private CatalogPublicController publicController;

    private final ObjectMapper objectMapper = new ObjectMapper();

    @BeforeEach
    void setUp() {
        mockMvc = MockMvcBuilders.standaloneSetup(publicController)
                .setControllerAdvice(new GlobalRestExceptionHandler())
                .build();
    }

    @Test
    void listCategories_ReturnsSuccess() throws Exception {
        CategoryResponse cat = CategoryResponse.builder()
                .categoryId(UUID.randomUUID())
                .name("Kẹo mè xửng giòn")
                .slug("keo-me-xung-gion")
                .build();
        when(catalogService.getAllCategories()).thenReturn(List.of(cat));

        mockMvc.perform(get("/api/v1/categories"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data[0].name").value("Kẹo mè xửng giòn"));
    }

    @Test
    void listProducts_ReturnsPaginatedProducts() throws Exception {
        ProductListItemResponse item = ProductListItemResponse.builder()
                .productId(UUID.randomUUID())
                .name("Hộp Mè Xửng O Mạ 500g")
                .slug("hop-me-xung-o-ma-500g")
                .build();
        PageResponse<ProductListItemResponse> page = PageResponse.<ProductListItemResponse>builder()
                .items(List.of(item))
                .page(1)
                .pageSize(20)
                .total(1L)
                .totalPages(1)
                .build();

        when(catalogService.getPublicProducts(any())).thenReturn(page);

        mockMvc.perform(get("/api/v1/products"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.items[0].name").value("Hộp Mè Xửng O Mạ 500g"));
    }

    @Test
    void getProduct_ReturnsProductDetail() throws Exception {
        ProductDetailResponse detail = ProductDetailResponse.builder()
                .productId(UUID.randomUUID())
                .name("Kẹo Mè Xửng Dẻo Truyền Thống")
                .slug("keo-me-xung-deo")
                .build();

        when(catalogService.getPublicProductDetail(eq("keo-me-xung-deo"), any())).thenReturn(detail);

        mockMvc.perform(get("/api/v1/products/keo-me-xung-deo"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.slug").value("keo-me-xung-deo"));
    }

    @Test
    void searchProducts_ReturnsSearchResults() throws Exception {
        PageResponse<ProductListItemResponse> page = PageResponse.<ProductListItemResponse>builder()
                .items(Collections.emptyList())
                .page(1)
                .pageSize(20)
                .total(0L)
                .totalPages(0)
                .build();

        when(catalogService.searchProducts(eq("mè xửng"), eq(1), eq(20))).thenReturn(page);

        mockMvc.perform(get("/api/v1/products/search?q=mè xửng"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true));
    }

    @Test
    void getProductVariant_ReturnsVariant() throws Exception {
        ProductVariantResponse variant = ProductVariantResponse.builder()
                .skuCode("MX-GION-500G")
                .name("Hộp 500g")
                .build();

        when(catalogService.getVariantBySkuCode("MX-GION-500G")).thenReturn(variant);

        mockMvc.perform(get("/api/v1/products/variants/MX-GION-500G"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.skuCode").value("MX-GION-500G"));
    }

    @Test
    void validatePriceAndSku_ReturnsValidationResult() throws Exception {
        PriceValidationRequest req = PriceValidationRequest.builder()
                .idempotencyKey("idem-123")
                .channel(SalesChannel.D2C_WEB)
                .items(Collections.emptyList())
                .build();

        PriceValidationResponse resp = PriceValidationResponse.builder()
                .valid(true)
                .canonicalSubtotalUnits(150000L)
                .currencyCode("VND")
                .build();

        when(catalogService.validatePrices(any())).thenReturn(resp);

        mockMvc.perform(post("/api/v1/internal/prices/validate")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(req)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.valid").value(true))
                .andExpect(jsonPath("$.data.canonicalSubtotalUnits").value(150000));
    }
}
