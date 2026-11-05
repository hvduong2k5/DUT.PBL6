package com.hvduong.catalog.application.service.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;

import com.hvduong.catalog.application.dto.request.PriceValidationRequest;
import com.hvduong.catalog.application.dto.request.ProductFilterRequest;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListItemResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.mapper.CatalogDtoMapper;
import com.hvduong.catalog.common.enums.ListingStatus;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;
import com.hvduong.catalog.common.response.PageResponse;
import com.hvduong.catalog.domain.entity.Category;
import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductImage;
import com.hvduong.catalog.domain.entity.ProductVariant;
import com.hvduong.catalog.infrastructure.inventory.StockProjectionService;
import com.hvduong.catalog.repository.mybatis.CategoryMapper;
import com.hvduong.catalog.repository.mybatis.ChannelPriceMapper;
import com.hvduong.catalog.repository.mybatis.ProductImageMapper;
import com.hvduong.catalog.repository.mybatis.ProductMapper;
import com.hvduong.catalog.repository.mybatis.ProductVariantMapper;
import com.hvduong.catalog.repository.mybatis.projection.AdminCategoryRow;
import com.hvduong.catalog.repository.mybatis.projection.AdminStatusCountRow;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

@ExtendWith(MockitoExtension.class)
class CatalogServiceImplTest {

  @Mock private ProductMapper productMapper;
  @Mock private ProductVariantMapper variantMapper;
  @Mock private CategoryMapper categoryMapper;
  @Mock private ChannelPriceMapper channelPriceMapper;
  @Mock private ProductImageMapper imageMapper;
  @Mock private CatalogDtoMapper dtoMapper;
  @Mock private StockProjectionService stockProjectionService;

  private CatalogServiceImpl catalogService;

  private Product mockProduct;
  private ProductVariant mockVariant;
  private ChannelPrice mockPrice;
  private ProductImage mockImage;
  private UUID productId;
  private UUID variantId;
  private UUID categoryId;

  @BeforeEach
  void setUp() {
    catalogService =
        new CatalogServiceImpl(
            productMapper,
            variantMapper,
            channelPriceMapper,
            imageMapper,
            categoryMapper,
            stockProjectionService,
            dtoMapper);
    productId = UUID.randomUUID();
    variantId = UUID.randomUUID();
    categoryId = UUID.randomUUID();

    mockProduct = new Product();
    mockProduct.setId(productId);
    mockProduct.setName("Test Product");
    mockProduct.setSlug("test-product");
    mockProduct.setDescription("Description");
    mockProduct.setCategoryId(categoryId);
    org.mockito.Mockito.lenient()
        .when(productMapper.findPublicById(productId))
        .thenReturn(Optional.of(mockProduct));

    mockVariant = new ProductVariant();
    mockVariant.setId(variantId);
    mockVariant.setProductId(productId);
    mockVariant.setSkuCode("SKU-123");
    mockVariant.setListingStatus(ListingStatus.ACTIVE);

    mockPrice = new ChannelPrice();
    mockPrice.setVariantId(variantId);
    mockPrice.setAmountUnits(100000L);
    mockPrice.setAmountNanos(0);
    mockPrice.setCurrencyCode("VND");

    mockImage = new ProductImage();
    mockImage.setProductId(productId);
    mockImage.setObjectKey("image1.jpg");
    mockImage.setRole("COVER");
  }

  @Test
  void testGetPublicProducts() {
    // Given
    ProductFilterRequest filter = new ProductFilterRequest();
    filter.setPage(1);
    filter.setPageSize(10);
    filter.setCategoryId(categoryId);

    when(productMapper.findPublicProducts(
            anyString(),
            eq(categoryId),
            any(),
            any(),
            any(),
            anyString(),
            anyString(),
            anyInt(),
            anyInt()))
        .thenReturn(List.of(mockProduct));
    when(productMapper.countPublicProducts(
            anyString(), eq(categoryId), any(), any(), any(), anyString()))
        .thenReturn(1L);

    when(variantMapper.findActiveByProductId(productId)).thenReturn(List.of(mockVariant));
    when(channelPriceMapper.findActiveCurrentByVariantIds(any(), anyString()))
        .thenReturn(List.of(mockPrice));
    when(imageMapper.findApprovedByProductId(productId)).thenReturn(List.of(mockImage));

    ProductListItemResponse mockListItemResponse = new ProductListItemResponse();
    when(dtoMapper.toListItemResponse(any(Product.class))).thenReturn(mockListItemResponse);

    // When
    PageResponse<ProductListItemResponse> result = catalogService.getPublicProducts(filter);

    // Then
    assertNotNull(result);
    assertEquals(1, result.getTotal());
    assertEquals(1, result.getItems().size());
    assertEquals(mockListItemResponse, result.getItems().get(0));
  }

  @Test
  void testGetPublicProductDetail_Success() {
    // Given
    when(productMapper.findPublicBySlug("test-product")).thenReturn(Optional.of(mockProduct));
    when(variantMapper.findActiveByProductId(productId)).thenReturn(List.of(mockVariant));
    when(imageMapper.findApprovedByProductId(productId)).thenReturn(List.of(mockImage));

    ProductDetailResponse detailResponse = ProductDetailResponse.builder().build();
    when(dtoMapper.toDetailResponse(any(Product.class))).thenReturn(detailResponse);

    ProductVariantResponse variantResponse = new ProductVariantResponse();
    when(dtoMapper.toVariantResponse(any(ProductVariant.class))).thenReturn(variantResponse);

    when(channelPriceMapper.findActiveByVariantAndChannel(eq(variantId), anyString()))
        .thenReturn(Optional.of(mockPrice));

    MoneyResponse moneyResponse = new MoneyResponse();
    when(dtoMapper.toMoneyResponse(any(ChannelPrice.class))).thenReturn(moneyResponse);

    // When
    ProductDetailResponse result =
        catalogService.getPublicProductDetail("test-product", SalesChannel.D2C_WEB);

    // Then
    assertNotNull(result);
    assertEquals(1, result.getVariants().size());
    assertEquals(1, result.getImageUrls().size());
  }

  @Test
  void testGetPublicProductDetail_NotFound() {
    // Given
    when(productMapper.findPublicBySlug("non-existent")).thenReturn(Optional.empty());

    // When & Then
    CatalogException exception =
        assertThrows(
            CatalogException.class,
            () -> catalogService.getPublicProductDetail("non-existent", SalesChannel.D2C_WEB));
    assertEquals(ErrorCode.CATALOG_PRODUCT_NOT_FOUND, exception.getErrorCode());
  }

  @Test
  void testSearchProducts() {
    // Given
    when(productMapper.findPublicProducts(
            eq("Test"), any(), any(), any(), any(), eq("NEWEST"), eq("D2C_WEB"), eq(0), eq(20)))
        .thenReturn(List.of(mockProduct));
    when(productMapper.countPublicProducts(eq("Test"), any(), any(), any(), any(), eq("D2C_WEB")))
        .thenReturn(1L);

    ProductListItemResponse mockListItemResponse = new ProductListItemResponse();
    when(dtoMapper.toListItemResponse(any(Product.class))).thenReturn(mockListItemResponse);

    when(variantMapper.findActiveByProductId(productId)).thenReturn(List.of(mockVariant));
    when(channelPriceMapper.findActiveCurrentByVariantIds(any(), anyString()))
        .thenReturn(List.of(mockPrice));
    when(imageMapper.findApprovedByProductId(productId)).thenReturn(List.of(mockImage));

    // When
    PageResponse<ProductListItemResponse> result = catalogService.searchProducts("Test", 1, 20);

    // Then
    assertNotNull(result);
    assertEquals(1, result.getItems().size());
    assertEquals(1, result.getTotal());
  }

  @Test
  void testSearchProducts_EmptyQuery() {
    // When
    PageResponse<ProductListItemResponse> result = catalogService.searchProducts("   ", 1, 20);

    // Then
    assertNotNull(result);
    assertTrue(result.getItems().isEmpty());
  }

  @Test
  void testGetAllCategories() {
    // Given
    Category parent = new Category();
    parent.setId(categoryId);
    parent.setParentId(null);

    Category child = new Category();
    child.setId(UUID.randomUUID());
    child.setParentId(categoryId);

    when(categoryMapper.findAllActive()).thenReturn(List.of(parent, child));

    CategoryResponse parentResponse = CategoryResponse.builder().build();
    when(dtoMapper.toCategoryResponse(parent)).thenReturn(parentResponse);

    CategoryResponse childResponse = CategoryResponse.builder().build();
    when(dtoMapper.toCategoryResponse(child)).thenReturn(childResponse);

    // When
    List<CategoryResponse> result = catalogService.getAllCategories();

    // Then
    assertNotNull(result);
    assertEquals(2, result.size());
    assertEquals(parentResponse, result.get(0));
    assertEquals(childResponse, result.get(1));
  }

  @Test
  void testGetVariantBySkuCode_Success() {
    // Given
    when(variantMapper.findBySkuCode("SKU-123")).thenReturn(Optional.of(mockVariant));

    ProductVariantResponse response = new ProductVariantResponse();
    when(dtoMapper.toVariantResponse(mockVariant)).thenReturn(response);

    when(channelPriceMapper.findActiveByVariantAndChannel(eq(variantId), anyString()))
        .thenReturn(Optional.of(mockPrice));

    MoneyResponse moneyResponse = new MoneyResponse();
    when(dtoMapper.toMoneyResponse(mockPrice)).thenReturn(moneyResponse);

    // When
    ProductVariantResponse result = catalogService.getVariantBySkuCode("SKU-123");

    // Then
    assertNotNull(result);
    assertEquals(response, result);
  }

  @Test
  void testGetVariantBySkuCode_NotFound() {
    // Given
    when(variantMapper.findBySkuCode("INVALID-SKU")).thenReturn(Optional.empty());

    // When & Then
    CatalogException exception =
        assertThrows(
            CatalogException.class, () -> catalogService.getVariantBySkuCode("INVALID-SKU"));
    assertEquals(ErrorCode.CATALOG_VARIANT_NOT_FOUND, exception.getErrorCode());
  }

  @Test
  void testValidatePrices_Valid() {
    // Given
    PriceValidationRequest request =
        PriceValidationRequest.builder()
            .channel(SalesChannel.D2C_WEB)
            .items(
                List.of(
                    PriceValidationRequest.Item.builder()
                        .skuCode("SKU-123")
                        .quantity(2)
                        .clientUnitPrice(100000L)
                        .nanos(0)
                        .currencyCode("VND")
                        .build()))
            .build();

    when(variantMapper.findBySkuCode("SKU-123")).thenReturn(Optional.of(mockVariant));
    when(channelPriceMapper.findActiveByVariantAndChannel(
            variantId, SalesChannel.D2C_WEB.getValue()))
        .thenReturn(Optional.of(mockPrice));

    // When
    PriceValidationResponse response = catalogService.validatePrices(request);

    // Then
    assertTrue(response.isValid());
    assertEquals(200000L, response.getCanonicalSubtotalUnits());
    assertTrue(response.getDiscrepancies().isEmpty());
  }

  @Test
  void testValidatePrices_InvalidPrice() {
    // Given
    PriceValidationRequest request =
        PriceValidationRequest.builder()
            .channel(SalesChannel.D2C_WEB)
            .items(
                List.of(
                    PriceValidationRequest.Item.builder()
                        .skuCode("SKU-123")
                        .quantity(1)
                        .clientUnitPrice(50000L) // Wrong price (should be 100000)
                        .nanos(0)
                        .currencyCode("VND")
                        .build()))
            .build();

    when(variantMapper.findBySkuCode("SKU-123")).thenReturn(Optional.of(mockVariant));
    when(channelPriceMapper.findActiveByVariantAndChannel(
            variantId, SalesChannel.D2C_WEB.getValue()))
        .thenReturn(Optional.of(mockPrice));

    // When
    PriceValidationResponse response = catalogService.validatePrices(request);

    // Then
    assertFalse(response.isValid());
    assertEquals(
        100000L,
        response.getCanonicalSubtotalUnits()); // still calculates correct subtotal based on DB
    assertEquals(1, response.getDiscrepancies().size());
    assertEquals(
        "Giá hoặc tiền tệ do client gửi không khớp giá niêm yết",
        response.getDiscrepancies().get(0).getReason());
  }

  @Test
  void testValidatePrices_SkuNotFound() {
    // Given
    PriceValidationRequest request =
        PriceValidationRequest.builder()
            .channel(SalesChannel.D2C_WEB)
            .items(
                List.of(
                    PriceValidationRequest.Item.builder()
                        .skuCode("SKU-404")
                        .quantity(1)
                        .clientUnitPrice(100000L)
                        .nanos(0)
                        .currencyCode("VND")
                        .build()))
            .build();

    when(variantMapper.findBySkuCode("SKU-404")).thenReturn(Optional.empty());

    // When
    PriceValidationResponse response = catalogService.validatePrices(request);

    // Then
    assertFalse(response.isValid());
    assertEquals(1, response.getDiscrepancies().size());
    assertTrue(response.getDiscrepancies().get(0).getReason().contains("SKU không tồn tại"));
  }

  @Test
  void testSearchProducts_PushesQueryToSql() {
    // Given
    when(productMapper.findPublicProducts(
            eq("mè xùng"), any(), any(), any(), any(), eq("NEWEST"), eq("D2C_WEB"), eq(0), eq(20)))
        .thenReturn(List.of(mockProduct));
    when(productMapper.countPublicProducts(
            eq("mè xùng"), any(), any(), any(), any(), eq("D2C_WEB")))
        .thenReturn(5L);

    when(variantMapper.findActiveByProductId(productId)).thenReturn(List.of(mockVariant));
    when(channelPriceMapper.findActiveCurrentByVariantIds(any(), anyString()))
        .thenReturn(List.of(mockPrice));
    when(imageMapper.findApprovedByProductId(productId)).thenReturn(List.of(mockImage));

    ProductListItemResponse mockListItemResponse = new ProductListItemResponse();
    when(dtoMapper.toListItemResponse(any(Product.class))).thenReturn(mockListItemResponse);

    // When
    PageResponse<ProductListItemResponse> result = catalogService.searchProducts("mè xùng", 1, 20);

    // Then
    assertNotNull(result);
    assertEquals(5, result.getTotal());
    assertEquals(1, result.getItems().size());
  }

  @Test
  void testSearchProducts_EmptyQueryReturnsEmpty() {
    // When
    PageResponse<ProductListItemResponse> result = catalogService.searchProducts("   ", 1, 20);

    // Then
    assertNotNull(result);
    assertTrue(result.getItems().isEmpty());
    assertEquals(0, result.getTotal());
  }

  @Test
  void testGetAdminProducts_WithFilter() {
    // Given
    when(productMapper.findAdminProducts(eq("mè"), eq("ACTIVE"), eq(0), eq(20)))
        .thenReturn(List.of(mockProduct));
    when(productMapper.countAdminProducts(eq("mè"), eq("ACTIVE"))).thenReturn(1L);

    AdminStatusCountRow activeRow = new AdminStatusCountRow();
    activeRow.setStatus("ACTIVE");
    activeRow.setCount(1L);
    AdminStatusCountRow suspendedRow = new AdminStatusCountRow();
    suspendedRow.setStatus("SUSPENDED");
    suspendedRow.setCount(0L);
    when(productMapper.countAdminProductsByStatus(eq("mè")))
        .thenReturn(List.of(activeRow, suspendedRow));

    AdminCategoryRow catRow = new AdminCategoryRow();
    catRow.setCategoryId(categoryId);
    catRow.setCategoryName("Mè xửng Huế");
    when(productMapper.findAdminCategories(eq("mè"))).thenReturn(List.of(catRow));

    when(variantMapper.findByProductIds(any())).thenReturn(List.of(mockVariant));
    when(channelPriceMapper.findActiveCurrentByVariantIds(any(), anyString()))
        .thenReturn(List.of(mockPrice));
    org.mockito.Mockito.lenient()
        .when(imageMapper.findApprovedByProductId(productId))
        .thenReturn(List.of(mockImage));

    AdminProductListItemResponse mockListItem = new AdminProductListItemResponse();
    org.mockito.Mockito.lenient()
        .when(dtoMapper.toAdminListItemResponse(any(Product.class)))
        .thenReturn(mockListItem);

    // When
    AdminProductListResponse result = catalogService.getAdminProducts(1, 20, "mè", "ACTIVE");

    // Then
    assertNotNull(result);
    assertEquals(1, result.getItems().size());
    assertEquals(3, result.getStatusCounts().size());
    assertEquals("ACTIVE", result.getStatusCounts().get(0).getStatus());
    assertEquals("Đang bán", result.getStatusCounts().get(0).getLabel());
    assertEquals(1L, result.getStatusCounts().get(0).getCount());
    assertEquals(1, result.getCategories().size());
    assertEquals(categoryId.toString(), result.getCategories().get(0).getCategoryId());
    assertEquals("Mè xửng Huế", result.getCategories().get(0).getCategoryName());
    assertNotNull(result.getCalculatedAt());
  }

  @Test
  void testGetAdminProducts_ReturnsAllStatusCounts() {
    // Given
    org.mockito.Mockito.lenient()
        .when(productMapper.findAdminProducts(anyString(), anyString(), anyInt(), anyInt()))
        .thenReturn(List.of());
    org.mockito.Mockito.lenient()
        .when(productMapper.countAdminProducts(anyString(), anyString()))
        .thenReturn(0L);
    org.mockito.Mockito.lenient()
        .when(productMapper.countAdminProductsByStatus(anyString()))
        .thenReturn(List.of());
    org.mockito.Mockito.lenient()
        .when(productMapper.findAdminCategories(anyString()))
        .thenReturn(List.of());

    // When
    AdminProductListResponse result = catalogService.getAdminProducts(1, 20, null, null);

    // Then
    assertNotNull(result);
    assertEquals(3, result.getStatusCounts().size()); // ACTIVE, SUSPENDED, ARCHIVED
    assertEquals("ACTIVE", result.getStatusCounts().get(0).getStatus());
    assertEquals("Đang bán", result.getStatusCounts().get(0).getLabel());
    assertEquals("SUSPENDED", result.getStatusCounts().get(1).getStatus());
    assertEquals("Tạm ngừng bán", result.getStatusCounts().get(1).getLabel());
    assertEquals("ARCHIVED", result.getStatusCounts().get(2).getStatus());
    assertEquals("Ngừng kinh doanh", result.getStatusCounts().get(2).getLabel());
  }

  @Test
  void testGetAdminProductDetail_ReturnsAllVariantsWithPrices() {
    // Given
    UUID id = UUID.randomUUID();
    when(productMapper.findById(eq(id))).thenReturn(Optional.of(mockProduct));
    when(imageMapper.findApprovedByProductId(id)).thenReturn(List.of(mockImage));

    when(variantMapper.findAllByProductId(id)).thenReturn(List.of(mockVariant));
    when(channelPriceMapper.findActiveCurrentByVariantIds(any(), anyString()))
        .thenReturn(List.of(mockPrice));

    AdminProductDetailResponse mockDetail = new AdminProductDetailResponse();
    when(dtoMapper.toAdminDetailResponse(any(Product.class))).thenReturn(mockDetail);

    // When
    AdminProductDetailResponse result = catalogService.getAdminProductDetail(id.toString());

    // Then
    assertNotNull(result);
    assertEquals(1, result.getSkus().size());
  }
}
