package com.hvduong.catalog.application.service.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

import com.hvduong.catalog.application.dto.request.CreateCategoryRequest;
import com.hvduong.catalog.application.dto.request.CreatePriceScheduleRequest;
import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.PatchCategoryRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.UpdateCategoryRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminPriceResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.mapper.CatalogDtoMapper;
import com.hvduong.catalog.application.service.CatalogCacheEvictor;
import com.hvduong.catalog.common.enums.ApprovalStatus;
import com.hvduong.catalog.common.enums.ListingStatus;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;
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
import org.springframework.cache.annotation.Cacheable;
import org.springframework.transaction.support.TransactionSynchronizationManager;

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

  // F-22: các transition cập nhật trạng thái bằng SQL trực tiếp nên object đã nạp
  // trước đó vẫn giữ trạng thái cũ. Response phải được dựng từ lần đọc mới nhất —
  // trước bản sửa, submit trả về DRAFT v0 trong khi DB đã là PENDING v1.
  @Test
  void submitProductForReview_ResponseReflectsUpdatedState() {
    Product draft = new Product();
    draft.setId(productId);
    draft.setApprovalStatus(ApprovalStatus.DRAFT);
    draft.setVersion(0L);
    Product pending = new Product();
    pending.setId(productId);
    pending.setApprovalStatus(ApprovalStatus.PENDING);
    pending.setVersion(1L);

    when(productMapper.findById(productId)).thenReturn(Optional.of(draft), Optional.of(pending));
    when(productMapper.updateApprovalStatus(
            productId, ApprovalStatus.PENDING.getValue(), "admin-1", 0L))
        .thenReturn(1);
    stubDetailMappingFromEntity();

    ProductDetailResponse result = adminService.submitProductForReview(productId, "admin-1", 0L);

    assertEquals("PENDING", result.getApprovalStatus());
    assertEquals(1L, result.getVersion());
  }

  @Test
  void approveProduct_ResponseReflectsUpdatedState() {
    Product pending = new Product();
    pending.setId(productId);
    pending.setApprovalStatus(ApprovalStatus.PENDING);
    pending.setVersion(1L);
    Product approved = new Product();
    approved.setId(productId);
    approved.setApprovalStatus(ApprovalStatus.APPROVED);
    approved.setVersion(2L);

    when(productMapper.findById(productId)).thenReturn(Optional.of(pending), Optional.of(approved));
    when(productMapper.updateApprovalStatus(
            productId, ApprovalStatus.APPROVED.getValue(), "admin-1", 1L))
        .thenReturn(1);
    stubDetailMappingFromEntity();

    ProductDetailResponse result = adminService.approveProduct(productId, "admin-1", 1L);

    assertEquals("APPROVED", result.getApprovalStatus());
    assertEquals(2L, result.getVersion());
  }

  private void stubDetailMappingFromEntity() {
    when(dtoMapper.toDetailResponse(any()))
        .thenAnswer(
            invocation -> {
              Product p = invocation.getArgument(0);
              return ProductDetailResponse.builder()
                  .productId(p.getId())
                  .approvalStatus(p.getApprovalStatus().name())
                  .version(p.getVersion())
                  .build();
            });
    when(variantMapper.findAllByProductId(productId)).thenReturn(List.of());
    when(imageMapper.findApprovedByProductId(productId)).thenReturn(List.of());
  }

  @Test
  void getAdminProductsMustNotShareTheCategoriesCache() throws Exception {
    // F-43: getAdminProducts once carried the categories list's own
    // @Cacheable("catalog:categories", 'all'); a cache hit cast the
    // cached ArrayList of categories to AdminProductListResponse and
    // the admin products endpoint returned HTTP 500.
    var method =
        CatalogAdminServiceImpl.class.getMethod(
            "getAdminProducts", int.class, int.class, String.class, String.class);
    assertNull(method.getAnnotation(Cacheable.class));
    // The categories list itself keeps its cache.
    var categories = CatalogServiceImpl.class.getMethod("getAllCategories");
    assertNotNull(categories.getAnnotation(Cacheable.class));
  }

  private void stubDetailMapping() {
    when(dtoMapper.toDetailResponse(any()))
        .thenReturn(ProductDetailResponse.builder().productId(productId).build());
    when(variantMapper.findAllByProductId(productId)).thenReturn(List.of());
    when(imageMapper.findApprovedByProductId(productId)).thenReturn(List.of());
  }

  @Test
  void categoryCreationEvictsCacheAfterCommit() {
    var request = new CreateCategoryRequest("Mè xửng", "me-xung", null, null, 0);
    when(categoryMapper.insert(any())).thenReturn(1);
    TransactionSynchronizationManager.initSynchronization();
    try {
      adminService.createCategory(request, "employee", "category-create-01");
      verifyNoInteractions(cacheEvictor);
      TransactionSynchronizationManager.getSynchronizations().forEach(sync -> sync.afterCommit());
      verify(cacheEvictor).evictCategoriesCache();
    } finally {
      TransactionSynchronizationManager.clearSynchronization();
    }
  }

  @Test
  void updateCategoryThrowsWhenParentIsSelf() {
    UUID catId = UUID.randomUUID();
    Category existing = Category.builder().id(catId).name("Mè xửng").slug("me-xung").build();
    when(categoryMapper.findById(catId)).thenReturn(Optional.of(existing));

    var updateReq =
        UpdateCategoryRequest.builder()
            .name("Mè xửng")
            .slug("me-xung")
            .parentId(catId)
            .expectedVersion(1L)
            .build();

    CatalogException ex =
        assertThrows(
            CatalogException.class, () -> adminService.updateCategory(catId, updateReq, "admin-1"));
    assertEquals(ErrorCode.INVALID_ARGUMENT, ex.getErrorCode());
  }

  @Test
  void updateCategoryThrowsWhenCircularHierarchyDetected() {
    UUID catA = UUID.randomUUID();
    UUID catB = UUID.randomUUID();
    UUID catC = UUID.randomUUID();

    Category existingA = Category.builder().id(catA).name("Cat A").slug("cat-a").build();
    Category existingB =
        Category.builder().id(catB).name("Cat B").slug("cat-b").parentId(catA).build();
    Category existingC =
        Category.builder().id(catC).name("Cat C").slug("cat-c").parentId(catB).build();

    when(categoryMapper.findById(catA)).thenReturn(Optional.of(existingA));
    when(categoryMapper.findById(catB)).thenReturn(Optional.of(existingB));
    when(categoryMapper.findById(catC)).thenReturn(Optional.of(existingC));
    when(categoryMapper.existsBySlug("cat-a", catA)).thenReturn(false);

    // Attempt to make C parent of A (cycle: A -> B -> C -> A)
    var updateReq =
        UpdateCategoryRequest.builder()
            .name("Cat A")
            .slug("cat-a")
            .parentId(catC)
            .expectedVersion(1L)
            .build();

    CatalogException ex =
        assertThrows(
            CatalogException.class, () -> adminService.updateCategory(catA, updateReq, "admin-1"));
    assertEquals(ErrorCode.INVALID_ARGUMENT, ex.getErrorCode());
  }

  @Test
  void patchCategoryThrowsWhenCircularHierarchyDetected() {
    UUID catA = UUID.randomUUID();
    UUID catB = UUID.randomUUID();

    Category existingA = Category.builder().id(catA).name("Cat A").slug("cat-a").build();
    Category existingB =
        Category.builder().id(catB).name("Cat B").slug("cat-b").parentId(catA).build();

    when(categoryMapper.findById(catA)).thenReturn(Optional.of(existingA));
    when(categoryMapper.findById(catB)).thenReturn(Optional.of(existingB));

    var patchReq = PatchCategoryRequest.builder().parentId(catB).expectedVersion(1L).build();

    CatalogException ex =
        assertThrows(
            CatalogException.class, () -> adminService.patchCategory(catA, patchReq, "admin-1"));
    assertEquals(ErrorCode.INVALID_ARGUMENT, ex.getErrorCode());
  }

  @Test
  void updateVariantPrices_ThrowsWhenVersionConflict() {
    UUID variantId = UUID.randomUUID();
    ProductVariant variant =
        ProductVariant.builder().id(variantId).productId(productId).version(2L).build();
    when(variantMapper.findById(variantId)).thenReturn(Optional.of(variant));

    UpdatePriceRequest req =
        UpdatePriceRequest.builder()
            .expectedVariantVersion(1L)
            .prices(
                List.of(
                    UpdatePriceRequest.PriceEntry.builder()
                        .channel(SalesChannel.D2C_WEB)
                        .amountUnits(50000L)
                        .currencyCode("VND")
                        .build()))
            .build();

    CatalogException ex =
        assertThrows(
            CatalogException.class,
            () -> adminService.updateVariantPrices(productId, variantId, req, "admin-1"));
    assertEquals(ErrorCode.CATALOG_VERSION_CONFLICT, ex.getErrorCode());
  }

  @Test
  void schedulePrice_Future_DoesNotIncrementPriceRevisionOrPublishOutbox() {
    UUID skuId = UUID.randomUUID();
    Product product = new Product();
    product.setId(productId);
    ProductVariant variant =
        ProductVariant.builder()
            .id(skuId)
            .productId(productId)
            .skuCode("MX-TEST")
            .version(1L)
            .priceRevision(1L)
            .build();

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(variantMapper.findById(skuId)).thenReturn(Optional.of(variant));

    Instant futureFrom = Instant.now().plusSeconds(3600);
    CreatePriceScheduleRequest req =
        CreatePriceScheduleRequest.builder()
            .channel(SalesChannel.D2C_WEB)
            .amountUnits(70000L)
            .amountNanos(0)
            .currencyCode("VND")
            .effectiveFrom(futureFrom)
            .build();

    AdminPriceResponse res = adminService.schedulePrice(productId, skuId, req, "admin-1");

    assertNotNull(res);
    assertEquals("D2C_WEB", res.getChannel());
    assertEquals(70000L, res.getAmountUnits());
    verify(channelPriceMapper, never()).supersedePreviousPrice(any(), any());
    verify(variantMapper, never()).incrementPriceRevision(any());
    verify(outboxWriter, never()).enqueue(any(), any(), any(), any());
    verify(channelPriceMapper, times(1)).insert(any());
  }

  @Test
  void schedulePrice_Immediate_IncrementsRevisionAndPublishesOutbox() {
    UUID skuId = UUID.randomUUID();
    Product product = new Product();
    product.setId(productId);
    ProductVariant variant =
        ProductVariant.builder()
            .id(skuId)
            .productId(productId)
            .skuCode("MX-TEST")
            .version(1L)
            .priceRevision(1L)
            .build();
    ProductVariant updatedVariant =
        ProductVariant.builder()
            .id(skuId)
            .productId(productId)
            .skuCode("MX-TEST")
            .version(1L)
            .priceRevision(2L)
            .build();

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(variantMapper.findById(skuId))
        .thenReturn(Optional.of(variant), Optional.of(updatedVariant));

    Instant pastFrom = Instant.now().minusSeconds(10);
    CreatePriceScheduleRequest req =
        CreatePriceScheduleRequest.builder()
            .channel(SalesChannel.D2C_WEB)
            .amountUnits(80000L)
            .amountNanos(0)
            .currencyCode("VND")
            .effectiveFrom(pastFrom)
            .build();

    AdminPriceResponse res = adminService.schedulePrice(productId, skuId, req, "admin-1");

    assertNotNull(res);
    assertEquals(80000L, res.getAmountUnits());
    verify(channelPriceMapper).supersedePreviousPrice(skuId, SalesChannel.D2C_WEB.getValue());
    verify(variantMapper).incrementPriceRevision(skuId);
    verify(outboxWriter)
        .enqueue(
            eq("Product"),
            eq(productId.toString()),
            eq("vn.omama.catalog.product.price.changed.v1"),
            any());
  }

  @Test
  void schedulePrice_ThrowsWhenEffectiveToBeforeEffectiveFrom() {
    UUID skuId = UUID.randomUUID();
    Product product = new Product();
    product.setId(productId);
    ProductVariant variant =
        ProductVariant.builder().id(skuId).productId(productId).version(1L).build();

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(variantMapper.findById(skuId)).thenReturn(Optional.of(variant));

    Instant from = Instant.now().plusSeconds(3600);
    Instant to = from.minusSeconds(60);
    CreatePriceScheduleRequest req =
        CreatePriceScheduleRequest.builder()
            .channel(SalesChannel.D2C_WEB)
            .amountUnits(80000L)
            .effectiveFrom(from)
            .effectiveTo(to)
            .build();

    CatalogException ex =
        assertThrows(
            CatalogException.class,
            () -> adminService.schedulePrice(productId, skuId, req, "admin-1"));
    assertEquals(ErrorCode.INVALID_ARGUMENT, ex.getErrorCode());
  }

  @Test
  void deleteScheduledPrice_FuturePrice_Success() {
    UUID skuId = UUID.randomUUID();
    UUID priceId = UUID.randomUUID();
    Product product = new Product();
    product.setId(productId);
    ProductVariant variant =
        ProductVariant.builder().id(skuId).productId(productId).version(1L).build();

    ChannelPrice price =
        ChannelPrice.builder()
            .id(priceId)
            .variantId(skuId)
            .channel(SalesChannel.D2C_WEB)
            .effectiveFrom(Instant.now().plusSeconds(7200))
            .build();

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(variantMapper.findById(skuId)).thenReturn(Optional.of(variant));
    when(channelPriceMapper.findById(priceId)).thenReturn(Optional.of(price));
    when(channelPriceMapper.deleteFuturePriceById(priceId, skuId)).thenReturn(1);

    assertDoesNotThrow(
        () -> adminService.deleteScheduledPrice(productId, skuId, priceId, "admin-1"));
    verify(channelPriceMapper).deleteFuturePriceById(priceId, skuId);
  }

  @Test
  void deleteScheduledPrice_ActivePrice_ThrowsCannotDelete() {
    UUID skuId = UUID.randomUUID();
    UUID priceId = UUID.randomUUID();
    Product product = new Product();
    product.setId(productId);
    ProductVariant variant =
        ProductVariant.builder().id(skuId).productId(productId).version(1L).build();

    ChannelPrice price =
        ChannelPrice.builder()
            .id(priceId)
            .variantId(skuId)
            .channel(SalesChannel.D2C_WEB)
            .effectiveFrom(Instant.now().minusSeconds(100))
            .build();

    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(variantMapper.findById(skuId)).thenReturn(Optional.of(variant));
    when(channelPriceMapper.findById(priceId)).thenReturn(Optional.of(price));

    CatalogException ex =
        assertThrows(
            CatalogException.class,
            () -> adminService.deleteScheduledPrice(productId, skuId, priceId, "admin-1"));
    assertEquals(ErrorCode.CATALOG_PRICE_CANNOT_DELETE_ACTIVE, ex.getErrorCode());
    verify(channelPriceMapper, never()).deleteFuturePriceById(any(), any());
  }
}
