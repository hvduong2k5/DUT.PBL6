package com.hvduong.catalog.api;

import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.ProductTransitionRequest;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.service.CatalogAdminService;
import com.hvduong.catalog.common.handler.GlobalRestExceptionHandler;

import java.util.Collections;
import java.util.UUID;

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

import com.fasterxml.jackson.databind.ObjectMapper;

@ExtendWith(MockitoExtension.class)
class CatalogAdminControllerTest {

  private MockMvc mockMvc;

  @Mock private CatalogAdminService adminService;

  @InjectMocks private CatalogAdminController adminController;

  private final ObjectMapper objectMapper = new ObjectMapper();

  @BeforeEach
  void setUp() {
    HandlerMethodArgumentResolver jwtResolver =
        new HandlerMethodArgumentResolver() {
          @Override
          public boolean supportsParameter(MethodParameter parameter) {
            return parameter.getParameterType().equals(Jwt.class);
          }

          @Override
          public Object resolveArgument(
              MethodParameter parameter,
              ModelAndViewContainer mavContainer,
              NativeWebRequest webRequest,
              WebDataBinderFactory binderFactory) {
            return Jwt.withTokenValue("mock-token-xyz")
                .header("alg", "none")
                .subject("test-actor-123")
                .claim("email", "admin@omama.vn")
                .build();
          }
        };

    mockMvc =
        MockMvcBuilders.standaloneSetup(adminController)
            .setCustomArgumentResolvers(jwtResolver)
            .setControllerAdvice(new GlobalRestExceptionHandler())
            .build();
  }

  @Test
  void listAdminProducts_ReturnsSuccess() throws Exception {
    AdminProductListResponse list =
        AdminProductListResponse.builder()
            .items(Collections.emptyList())
            .statusCounts(Collections.emptyList())
            .categories(Collections.emptyList())
            .calculatedAt("2026-10-02T02:30:00.000Z")
            .build();

    when(adminService.getAdminProducts(eq(1), eq(20), isNull(), isNull())).thenReturn(list);

    mockMvc
        .perform(get("/api/v1/admin/catalog/products"))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.success").value(true))
        .andExpect(jsonPath("$.data").exists());
  }

  @Test
  void getAdminProduct_ReturnsProductDetail() throws Exception {
    UUID id = UUID.randomUUID();
    AdminProductDetailResponse detail =
        AdminProductDetailResponse.builder()
            .productId(id.toString())
            .name("Kẹo mè xửng Huế O Mạ")
            .build();

    when(adminService.getAdminProductDetail(id.toString())).thenReturn(detail);

    mockMvc
        .perform(get("/api/v1/admin/catalog/products/" + id))
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

    ProductDetailResponse res =
        ProductDetailResponse.builder()
            .productId(UUID.randomUUID())
            .name("Kẹo Mè Xửng Giòn 250g")
            .slug("keo-me-xung-gion-250g")
            .build();

    when(adminService.createProduct(any(), any())).thenReturn(res);

    mockMvc
        .perform(
            post("/api/v1/admin/catalog/products")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(req)))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.success").value(true))
        .andExpect(jsonPath("$.message").value("Tạo sản phẩm thành công"));
  }

  @Test
  void transitionProduct_Approve_ReturnsUpdatedProduct() throws Exception {
    UUID id = UUID.randomUUID();
    ProductTransitionRequest req =
        ProductTransitionRequest.builder().transition("APPROVE").expectedVersion(1L).build();

    ProductDetailResponse res =
        ProductDetailResponse.builder().productId(id).name("Kẹo Mè Xửng Giòn").build();

    when(adminService.approveProduct(eq(id), any(), eq(1L))).thenReturn(res);

    mockMvc
        .perform(
            post("/api/v1/admin/catalog/products/" + id + "/transition")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(req)))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.success").value(true))
        .andExpect(jsonPath("$.message").value("Chuyển trạng thái sản phẩm thành công"));
  }

  @Test
  void patchProduct_ReturnsSuccess() throws Exception {
    UUID id = UUID.randomUUID();
    PatchProductRequest req = new PatchProductRequest();
    req.setStory("Câu chuyện thương hiệu O Mạ");
    req.setExpectedVersion(1L);

    ProductDetailResponse res =
        ProductDetailResponse.builder().productId(id).story("Câu chuyện thương hiệu O Mạ").build();

    when(adminService.patchProduct(eq(id), any(), any())).thenReturn(res);

    mockMvc
        .perform(
            patch("/api/v1/admin/catalog/products/" + id)
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(req)))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.success").value(true))
        .andExpect(jsonPath("$.message").value("Cập nhật nội dung sản phẩm thành công"));
  }

  @Test
  void patchVariant_ReturnsSuccess() throws Exception {
    UUID productId = UUID.randomUUID();
    UUID skuId = UUID.randomUUID();
    PatchVariantRequest req = new PatchVariantRequest();
    req.setSaleStatus("PAUSED");
    req.setExpectedVersion(2L);

    ProductDetailResponse res = ProductDetailResponse.builder().productId(productId).build();

    when(adminService.patchVariant(eq(productId), eq(skuId), any(), any())).thenReturn(res);

    mockMvc
        .perform(
            patch("/api/v1/admin/catalog/products/" + productId + "/skus/" + skuId)
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(req)))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.success").value(true))
        .andExpect(jsonPath("$.message").value("Cập nhật SKU thành công"));
  }

  @Test
  void addImage_ReturnsSuccess() throws Exception {
    UUID productId = UUID.randomUUID();
    CreateProductImageRequest req = new CreateProductImageRequest();
    req.setObjectKey("products/cover.jpg");
    req.setRole("COVER");

    ProductDetailResponse res = ProductDetailResponse.builder().productId(productId).build();

    when(adminService.addImage(eq(productId), any(), any())).thenReturn(res);

    mockMvc
        .perform(
            post("/api/v1/admin/catalog/products/" + productId + "/images")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(req)))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.success").value(true))
        .andExpect(jsonPath("$.message").value("Thêm ảnh sản phẩm thành công"));
  }
}
