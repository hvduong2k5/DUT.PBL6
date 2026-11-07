package com.hvduong.catalog.application.service.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminPriceResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.mapper.CatalogDtoMapper;
import com.hvduong.catalog.application.service.CatalogCacheEvictor;
import com.hvduong.catalog.common.enums.ListingStatus;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.domain.entity.Category;
import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductImage;
import com.hvduong.catalog.domain.entity.ProductVariant;
import com.hvduong.catalog.infrastructure.outbox.CatalogOutboxWriter;
import com.hvduong.catalog.repository.mybatis.CategoryMapper;
import com.hvduong.catalog.repository.mybatis.ChannelPriceMapper;
import com.hvduong.catalog.repository.mybatis.ProductImageMapper;
import com.hvduong.catalog.repository.mybatis.ProductMapper;
import com.hvduong.catalog.repository.mybatis.ProductVariantMapper;

import java.time.Instant;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

@ExtendWith(MockitoExtension.class)
class CatalogAdminServiceImplTest {

  @Mock private ProductMapper productMapper;
  @Mock private CategoryMapper categoryMapper;
  @Mock private ProductVariantMapper variantMapper;
  @Mock private ChannelPriceMapper channelPriceMapper;
  @Mock private ProductImageMapper imageMapper;
  @Mock private CatalogDtoMapper dtoMapper;
  @Mock private CatalogCacheEvictor cacheEvictor;
  @Mock private CatalogOutboxWriter outboxWriter;

  private CatalogAdminServiceImpl adminService;

  private UUID productId;

  @BeforeEach
  void setUp() {
    adminService =
        new CatalogAdminServiceImpl(
            productMapper,
            variantMapper,
            channelPriceMapper,
            imageMapper,
            categoryMapper,
            dtoMapper,
            outboxWriter,
            cacheEvictor);
    productId = UUID.randomUUID();
  }

  @Test
  void createProduct_Success() {
    CreateProductRequest req = new CreateProductRequest();
    req.setSlug("test-slug");
    req.setCategoryId(UUID.randomUUID());
    req.setName("Test Product");

    when(productMapper.existsBySlug(req.getSlug(), null)).thenReturn(false);
    when(categoryMapper.findById(req.getCategoryId())).thenReturn(Optional.of(new Category()));

    ProductDetailResponse mockResponse = ProductDetailResponse.builder().build();
    mockResponse.setProductId(productId);
    when(dtoMapper.toDetailResponse(any())).thenReturn(mockResponse);

    ProductDetailResponse result = adminService.createProduct(req, "admin-1");

    assertNotNull(result);
    verify(productMapper, times(1)).insert(any(Product.class));
  }

  @Test
  void createProduct_SlugExists_ThrowsException() {
    CreateProductRequest req = new CreateProductRequest();
    req.setSlug("existing-slug");

    when(productMapper.existsBySlug(req.getSlug(), null)).thenReturn(true);

    assertThrows(CatalogException.class, () -> adminService.createProduct(req, "admin-1"));
    verify(productMapper, never()).insert(any());
  }

  @Test
  void updateProduct_Success() {
    UpdateProductRequest req = new UpdateProductRequest();
    req.setSlug("new-slug");
    req.setName("Updated Name");

    Product existingProduct = new Product();
    existingProduct.setId(productId);
    existingProduct.setVersion(1L);

    when(productMapper.findById(productId)).thenReturn(Optional.of(existingProduct));
    when(productMapper.existsBySlug("new-slug", productId)).thenReturn(false);
    when(productMapper.update(any(Product.class))).thenReturn(1);

    ProductDetailResponse mockResponse = ProductDetailResponse.builder().build();
    when(dtoMapper.toDetailResponse(any())).thenReturn(mockResponse);

    ProductDetailResponse result = adminService.updateProduct(productId, req, "admin-1");

    assertNotNull(result);
    verify(productMapper, times(1)).update(any(Product.class));
  }

  @Test
  void patchProduct_UpdatesStoryAndClearsOptionalField() {
    PatchProductRequest req = new PatchProductRequest();
    req.setStory("Câu chuyện thương hiệu mới");
    req.setDescription("");
    req.setOcopStar(5);
    req.setExpectedVersion(1L);

    Product existing = new Product();
    existing.setId(productId);
    existing.setVersion(1L);
    existing.setName("Kẹo mè xửng");
    existing.setSlug("keo-me-xung");
    existing.setDescription("Mô tả cũ");

    when(productMapper.findById(productId)).thenReturn(Optional.of(existing));
    when(productMapper.update(any(Product.class))).thenReturn(1);
    stubDetailMapping();

    ProductDetailResponse result = adminService.patchProduct(productId, req, "admin-1");

    assertNotNull(result);
    assertEquals("Câu chuyện thương hiệu mới", existing.getStory());
    assertNull(existing.getDescription());
    assertEquals(5, existing.getOcopStar());
    verify(cacheEvictor).evictCategoriesCache();
  }

  @Test
  void patchProduct_VersionConflict_ThrowsException() {
    PatchProductRequest req = new PatchProductRequest();
    req.setStory("Câu chuyện mới");
    req.setExpectedVersion(9L);

    Product existing = new Product();
    existing.setId(productId);
    existing.setVersion(1L);

    when(productMapper.findById(productId)).thenReturn(Optional.of(existing));
    when(productMapper.update(any(Product.class))).thenReturn(0);

    assertThrows(
        CatalogException.class, () -> adminService.patchProduct(productId, req, "admin-1"));
    verify(cacheEvictor, never()).evictCategoriesCache();
  }

  @Test
  void patchVariant_UpdatesAttributesAndSaleStatus() {
    UUID variantId = UUID.randomUUID();
    PatchVariantRequest req = new PatchVariantRequest();
    req.setVariantName("Hộp 500g");
    req.setWeightUnit("KILOGRAM");
    req.setPackagingType("HOP_THIEC");
    req.setSaleStatus("PAUSED");
    req.setExpectedVersion(2L);

    Product product = new Product();
    product.setId(productId);
    product.setVersion(1L);

    ProductVariant variant =
        ProductVariant.builder()
            .id(variantId)
            .productId(productId)
            .skuCode("MX-500")
            .variantName("Hộp 250g")
            .weightValue(new java.math.BigDecimal("0.5"))
            .weightUnit("GRAM")
            .listingStatus(ListingStatus.ACTIVE)
            .version(2L)
            .build();

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(variantMapper.findById(variantId)).thenReturn(Optional.of(variant));
    when(variantMapper.update(any(ProductVariant.class))).thenReturn(1);
    stubDetailMapping();

    ProductDetailResponse result = adminService.patchVariant(productId, variantId, req, "admin-1");

    assertNotNull(result);
    assertEquals("Hộp 500g", variant.getVariantName());
    assertEquals("KILOGRAM", variant.getWeightUnit());
    assertEquals("HOP_THIEC", variant.getPackagingType());
    assertEquals(ListingStatus.SUSPENDED, variant.getListingStatus());
  }

  @Test
  void patchVariant_NotBelongToProduct_ThrowsException() {
    UUID variantId = UUID.randomUUID();
    PatchVariantRequest req = new PatchVariantRequest();
    req.setExpectedVersion(1L);

    Product product = new Product();
    product.setId(productId);

    ProductVariant variant =
        ProductVariant.builder().id(variantId).productId(UUID.randomUUID()).version(1L).build();

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(variantMapper.findById(variantId)).thenReturn(Optional.of(variant));

    assertThrows(
        CatalogException.class,
        () -> adminService.patchVariant(productId, variantId, req, "admin-1"));
    verify(variantMapper, never()).update(any());
  }

  @Test
  void addImage_ApprovedCover_DemotesPreviousCover() {
    CreateProductImageRequest req = new CreateProductImageRequest();
    req.setObjectKey("products/cover-new.jpg");
    req.setRole("COVER");
    req.setMediaStatus("APPROVED");

    Product product = new Product();
    product.setId(productId);
    product.setVersion(1L);

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(imageMapper.demoteOtherApprovedCovers(productId, null)).thenReturn(1);
    stubDetailMapping();

    ProductDetailResponse result = adminService.addImage(productId, req, "admin-1");

    assertNotNull(result);
    verify(imageMapper).demoteOtherApprovedCovers(productId, null);
    verify(imageMapper).insert(any(ProductImage.class));
  }

  @Test
  void getVariantPrices_ReturnsHistory() {
    UUID variantId = UUID.randomUUID();
    Product product = new Product();
    product.setId(productId);
    ProductVariant variant =
        ProductVariant.builder().id(variantId).productId(productId).version(1L).build();
    ChannelPrice price =
        ChannelPrice.builder()
            .id(UUID.randomUUID())
            .variantId(variantId)
            .channel(SalesChannel.D2C_WEB)
            .currencyCode("VND")
            .amountUnits(345000L)
            .amountNanos(0)
            .status("ACTIVE")
            .effectiveFrom(Instant.now())
            .version(1L)
            .build();

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(variantMapper.findById(variantId)).thenReturn(Optional.of(variant));
    when(channelPriceMapper.findAllByVariantId(variantId)).thenReturn(List.of(price));

    List<AdminPriceResponse> result = adminService.getVariantPrices(productId, variantId);

    assertEquals(1, result.size());
    assertEquals("D2C_WEB", result.get(0).getChannel());
    assertEquals(345000L, result.get(0).getAmountUnits());
  }

  private void stubDetailMapping() {
    when(dtoMapper.toDetailResponse(any()))
        .thenReturn(ProductDetailResponse.builder().productId(productId).build());
    when(variantMapper.findAllByProductId(productId)).thenReturn(List.of());
    when(imageMapper.findApprovedByProductId(productId)).thenReturn(List.of());
  }
}
