package com.hvduong.catalog.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.catalog.application.dto.request.*;
import com.hvduong.catalog.application.dto.response.*;
import com.hvduong.catalog.application.service.CatalogAdminService;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.common.handler.GlobalRestExceptionHandler;
import com.hvduong.catalog.common.response.PageResponse;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.core.MethodParameter;
import org.springframework.http.MediaType;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.bind.support.WebDataBinderFactory;
import org.springframework.web.context.request.NativeWebRequest;
import org.springframework.web.method.support.HandlerMethodArgumentResolver;
import org.springframework.web.method.support.ModelAndViewContainer;

import java.util.Collections;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@ExtendWith(MockitoExtension.class)
class CatalogAdminControllerTest {

    private MockMvc mockMvc;

    @Mock
    private CatalogAdminService adminService;

    @Mock
    private CatalogService catalogService;

    @InjectMocks
    private CatalogAdminController adminController;

    private final ObjectMapper objectMapper = new ObjectMapper();

    @BeforeEach
    void setUp() {
        HandlerMethodArgumentResolver jwtResolver = new HandlerMethodArgumentResolver() {
            @Override
            public boolean supportsParameter(MethodParameter parameter) {
                return parameter.getParameterType().equals(Jwt.class);
            }

            @Override
            public Object resolveArgument(MethodParameter parameter, ModelAndViewContainer mavContainer,
                                          NativeWebRequest webRequest, WebDataBinderFactory binderFactory) {
                return Jwt.withTokenValue("mock-token-xyz")
                        .header("alg", "none")
                        .subject("test-actor-123")
                        .claim("email", "admin@omama.vn")
                        .build();
            }
        };

        mockMvc = MockMvcBuilders.standaloneSetup(adminController)
                .setCustomArgumentResolvers(jwtResolver)
                .setControllerAdvice(new GlobalRestExceptionHandler())
                .build();
    }

    @Test
    void listAdminProducts_ReturnsSuccess() throws Exception {
        PageResponse<AdminProductListItemResponse> page = PageResponse.<AdminProductListItemResponse>builder()
                .items(Collections.emptyList())
                .page(1)
                .pageSize(20)
                .total(0L)
                .totalPages(0)
                .build();

        when(catalogService.getAdminProducts(eq(1), eq(20))).thenReturn(page);

        mockMvc.perform(get("/api/v1/admin/catalog/products"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true));
    }

    @Test
    void getAdminProduct_ReturnsProductDetail() throws Exception {
        UUID id = UUID.randomUUID();
        AdminProductDetailResponse detail = AdminProductDetailResponse.builder()
                .productId(id.toString())
                .name("Kẹo mè xửng Huế O Mạ")
                .build();

        when(catalogService.getAdminProductDetail(id.toString())).thenReturn(detail);

        mockMvc.perform(get("/api/v1/admin/catalog/products/" + id))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.data.name").value("Kẹo mè xửng Huế O Mạ"));
    }

    @Test
    void createProduct_ReturnsCreatedProduct() throws Exception {
        UUID catId = UUID.randomUUID();
        CreateProductRequest req = new CreateProductRequest();
        req.setCategoryId(catId);
        req.setName("Kẹo Mè Xửng Giòn 250g");
        req.setSlug("keo-me-xung-gion-250g");

        ProductDetailResponse res = ProductDetailResponse.builder()
                .productId(UUID.randomUUID())
                .name("Kẹo Mè Xửng Giòn 250g")
                .slug("keo-me-xung-gion-250g")
                .build();

        when(adminService.createProduct(any(), any())).thenReturn(res);

        mockMvc.perform(post("/api/v1/admin/catalog/products")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(req)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.message").value("Tạo sản phẩm thành công"));
    }

    @Test
    void transitionProduct_Approve_ReturnsUpdatedProduct() throws Exception {
        UUID id = UUID.randomUUID();
        ProductTransitionRequest req = ProductTransitionRequest.builder()
                .transition("APPROVE")
                .expectedVersion(1L)
                .build();

        ProductDetailResponse res = ProductDetailResponse.builder()
                .productId(id)
                .name("Kẹo Mè Xửng Giòn")
                .build();

        when(adminService.approveProduct(eq(id), any(), eq(1L))).thenReturn(res);

        mockMvc.perform(post("/api/v1/admin/catalog/products/" + id + "/transition")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(req)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.success").value(true))
                .andExpect(jsonPath("$.message").value("Chuyển trạng thái sản phẩm thành công"));
    }
}
