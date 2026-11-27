package com.hvduong.catalog.api;

import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.handler.GlobalRestExceptionHandler;

import java.math.BigDecimal;
import java.util.List;
import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.core.MethodParameter;
import org.springframework.security.authentication.TestingAuthenticationToken;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.bind.support.WebDataBinderFactory;
import org.springframework.web.context.request.NativeWebRequest;
import org.springframework.web.method.support.HandlerMethodArgumentResolver;
import org.springframework.web.method.support.ModelAndViewContainer;

import com.fasterxml.jackson.databind.ObjectMapper;

@ExtendWith(MockitoExtension.class)
class CatalogHttpControllerTest {

  private MockMvc mockMvc;

  @Mock private CatalogService catalogService;

  @InjectMocks private CatalogHttpController controller;

  private final ObjectMapper objectMapper = new ObjectMapper();

  private Jwt currentJwt;
  private TestingAuthenticationToken currentAuth;

  @BeforeEach
  void setUp() {
    controller = new CatalogHttpController(catalogService, objectMapper);

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
            return currentJwt;
          }
        };

    HandlerMethodArgumentResolver authResolver =
        new HandlerMethodArgumentResolver() {
          @Override
          public boolean supportsParameter(MethodParameter parameter) {
            return parameter.getParameterType().isAssignableFrom(TestingAuthenticationToken.class)
                || parameter.getParameterType().getSimpleName().equals("Authentication");
          }

          @Override
          public Object resolveArgument(
              MethodParameter parameter,
              ModelAndViewContainer mavContainer,
              NativeWebRequest webRequest,
              WebDataBinderFactory binderFactory) {
            return currentAuth;
          }
        };

    mockMvc =
        MockMvcBuilders.standaloneSetup(controller)
            .setCustomArgumentResolvers(jwtResolver, authResolver)
            .setControllerAdvice(new CatalogHttpErrors(), new GlobalRestExceptionHandler())
            .build();

    currentJwt = null;
    currentAuth = null;
  }

  @Test
  void getVariantBySku_DefaultChannel_Success() throws Exception {
    String sku = "GAO-ST25-1KG";
    ProductVariantResponse variant =
        ProductVariantResponse.builder()
            .variantId(UUID.randomUUID())
            .skuCode(sku)
            .name("Gạo ST25 1KG")
            .channel("D2C_WEB")
            .weightValue(new BigDecimal("1000"))
            .weightUnit("GRAM")
            .price(MoneyResponse.ofVnd(55000L))
            .priceVersion(4L)
            .stockStatus("IN_STOCK")
            .stockAvailable(null)
            .build();

    when(catalogService.getVariantBySkuCode(eq(sku), eq(SalesChannel.D2C_WEB))).thenReturn(variant);

    mockMvc
        .perform(get("/api/v1/products/variants/" + sku))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.variant.sku_code").value(sku))
        .andExpect(jsonPath("$.variant.channel").value("D2C_WEB"))
        .andExpect(jsonPath("$.variant.price.units").value(55000))
        .andExpect(jsonPath("$.variant.price_version").value(4))
        .andExpect(jsonPath("$.variant.stock_status").value("IN_STOCK"))
        .andExpect(jsonPath("$.variant.stock_available").doesNotExist())
        .andExpect(jsonPath("$.variant.original_price").doesNotExist());
  }

  @Test
  void getVariantBySku_B2bChannel_Unauthenticated_Returns403() throws Exception {
    mockMvc
        .perform(get("/api/v1/products/variants/GAO-ST25-1KG").param("channel", "B2B_WHOLESALE"))
        .andExpect(status().isForbidden())
        .andExpect(jsonPath("$.code").value("CHANNEL_ACCESS_DENIED"));
  }

  @Test
  void getVariantBySku_B2bChannel_AuthorizedB2b_Success() throws Exception {
    String sku = "GAO-ST25-1KG";
    currentJwt =
        Jwt.withTokenValue("mock-b2b-token")
            .header("alg", "none")
            .subject("b2b-user-1")
            .claim("roles", List.of("B2B_BUYER"))
            .build();
    currentAuth =
        new TestingAuthenticationToken(
            "b2b-user-1", "mock-b2b-token", List.of(new SimpleGrantedAuthority("ROLE_B2B")));

    ProductVariantResponse variant =
        ProductVariantResponse.builder()
            .variantId(UUID.randomUUID())
            .skuCode(sku)
            .name("Gạo ST25 1KG")
            .channel("B2B_WHOLESALE")
            .weightValue(new BigDecimal("1000"))
            .weightUnit("GRAM")
            .price(MoneyResponse.ofVnd(45000L))
            .priceVersion(2L)
            .stockStatus("IN_STOCK")
            .stockAvailable(null)
            .build();

    when(catalogService.getVariantBySkuCode(eq(sku), eq(SalesChannel.B2B_WHOLESALE)))
        .thenReturn(variant);

    mockMvc
        .perform(
            get("/api/v1/products/variants/" + sku)
                .param("channel", "B2B_WHOLESALE")
                .principal(currentAuth))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.variant.channel").value("B2B_WHOLESALE"))
        .andExpect(jsonPath("$.variant.price.units").value(45000))
        .andExpect(jsonPath("$.variant.price_version").value(2))
        .andExpect(jsonPath("$.variant.stock_status").value("IN_STOCK"));
  }

  @Test
  void getVariantBySku_PosChannel_StaffAuthorized_Success() throws Exception {
    String sku = "GAO-ST25-1KG";
    currentJwt =
        Jwt.withTokenValue("mock-staff-token")
            .header("alg", "none")
            .subject("staff-user-1")
            .claim("roles", List.of("STAFF"))
            .build();
    currentAuth =
        new TestingAuthenticationToken(
            "staff-user-1",
            "mock-staff-token",
            List.of(new SimpleGrantedAuthority("CATALOG_VIEW")));

    ProductVariantResponse variant =
        ProductVariantResponse.builder()
            .variantId(UUID.randomUUID())
            .skuCode(sku)
            .name("Gạo ST25 1KG")
            .channel("POS_QUAY")
            .weightValue(new BigDecimal("1000"))
            .weightUnit("GRAM")
            .price(MoneyResponse.ofVnd(50000L))
            .priceVersion(1L)
            .stockStatus("IN_STOCK")
            .stockAvailable(null)
            .build();

    when(catalogService.getVariantBySkuCode(eq(sku), eq(SalesChannel.POS_QUAY)))
        .thenReturn(variant);

    mockMvc
        .perform(
            get("/api/v1/products/variants/" + sku)
                .param("channel", "POS_QUAY")
                .principal(currentAuth))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.variant.channel").value("POS_QUAY"))
        .andExpect(jsonPath("$.variant.price.units").value(50000));
  }

  @Test
  void getVariantBySku_InvalidChannel_Returns400() throws Exception {
    mockMvc
        .perform(get("/api/v1/products/variants/GAO-ST25-1KG").param("channel", "UNKNOWN_CHANNEL"))
        .andExpect(status().isBadRequest())
        .andExpect(jsonPath("$.code").value("INVALID_ARGUMENT"));
  }

  @Test
  void searchProducts_WithFilters_Success() throws Exception {
    UUID categoryId = UUID.randomUUID();
    var item =
        com.hvduong.catalog.application.dto.response.ProductListItemResponse.builder()
            .productId(UUID.randomUUID())
            .name("Kẹo Mè Xửng Giòn 500g")
            .slug("keo-me-xung-gion-500g")
            .summary("Đặc sản Huế")
            .thumbnailUrl("https://cdn.omama.vn/products/mx_gion.jpg")
            .basePrice(MoneyResponse.ofVnd(35000L))
            .ocopStar(4)
            .categoryName("Mè xửng")
            .stockAvailable(100L)
            .build();
    var pageResponse =
        com.hvduong.catalog.common.response.PageResponse.of(List.of(item), 1, 10, 1L);

    when(catalogService.searchProducts(
            eq("mè"),
            eq(categoryId),
            eq(10000L),
            eq(50000L),
            eq(4),
            eq(com.hvduong.catalog.common.enums.SortBy.PRICE_ASC),
            eq(SalesChannel.D2C_WEB),
            eq(1),
            eq(10)))
        .thenReturn(pageResponse);

    mockMvc
        .perform(
            get("/api/v1/products/search")
                .param("query", "mè")
                .param("category_id", categoryId.toString())
                .param("min_price", "10000")
                .param("max_price", "50000")
                .param("ocop_star", "4")
                .param("sort_by", "PRICE_ASC")
                .param("channel", "D2C_WEB")
                .param("page", "1")
                .param("page_size", "10"))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$.query").value("mè"))
        .andExpect(jsonPath("$.total").value(1))
        .andExpect(jsonPath("$.items[0].name").value("Kẹo Mè Xửng Giòn 500g"))
        .andExpect(jsonPath("$.items[0].base_price.units").value(35000));
  }
}
