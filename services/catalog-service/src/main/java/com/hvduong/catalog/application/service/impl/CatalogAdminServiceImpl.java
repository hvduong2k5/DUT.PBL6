package com.hvduong.catalog.application.service.impl;

import com.hvduong.catalog.application.dto.request.CreateCategoryRequest;
import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductImageRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminPriceResponse;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListItemResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.mapper.AdminCatalogMapper;
import com.hvduong.catalog.application.mapper.CatalogDtoMapper;
import com.hvduong.catalog.application.service.CatalogAdminService;
import com.hvduong.catalog.application.service.CatalogCacheEvictor;
import com.hvduong.catalog.common.constant.CatalogConstants;
import com.hvduong.catalog.common.constant.SaleStatus;
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
import com.hvduong.catalog.repository.mybatis.projection.AdminCategoryRow;
import com.hvduong.catalog.repository.mybatis.projection.AdminStatusCountRow;

import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;
import java.util.function.Consumer;
import java.util.stream.Collectors;

import org.springframework.cache.annotation.CacheEvict;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionSynchronization;
import org.springframework.transaction.support.TransactionSynchronizationManager;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
@RequiredArgsConstructor
public class CatalogAdminServiceImpl implements CatalogAdminService {

  private final ProductMapper productMapper;
  private final ProductVariantMapper variantMapper;
  private final ChannelPriceMapper channelPriceMapper;
  private final ProductImageMapper imageMapper;
  private final CategoryMapper categoryMapper;
  private final CatalogDtoMapper dtoMapper;
  private final CatalogOutboxWriter outboxWriter;
  private final CatalogCacheEvictor cacheEvictor;

  @Override
  @Transactional
  public CategoryResponse createCategory(CreateCategoryRequest request, String actorId, String key) {
    if (!key.matches("[!-~]{16,128}")) {
      throw new CatalogException(ErrorCode.INVALID_ARGUMENT);
    }
    UUID categoryId = UUID.nameUUIDFromBytes(
        ("catalog:create-category:" + actorId + ":" + key).getBytes(StandardCharsets.UTF_8));
    Category category = Category.builder()
        .id(categoryId)
        .parentId(request.parentId())
        .name(request.name())
        .slug(request.slug())
        .description(request.description())
        .position(request.position() == null ? 0 : request.position())
        .isActive(true)
        .build();
    if (category.getParentId() != null
        && categoryMapper.findById(category.getParentId()).isEmpty()) {
      throw new CatalogException(ErrorCode.CATALOG_CATEGORY_NOT_FOUND);
    }

    Category saved = category;
    if (categoryMapper.insert(category) == 0) {
      Category existing = categoryMapper.findById(categoryId)
          .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_CATEGORY_SLUG_DUPLICATE));
      if (!Objects.equals(existing.getParentId(), category.getParentId())
          || !Objects.equals(existing.getName(), category.getName())
          || !Objects.equals(existing.getSlug(), category.getSlug())
          || !Objects.equals(existing.getDescription(), category.getDescription())
          || existing.getPosition() != category.getPosition()) {
        throw new CatalogException(ErrorCode.IDEMPOTENCY_CONFLICT);
      }
      saved = existing;
    }
    TransactionSynchronizationManager.registerSynchronization(new TransactionSynchronization() {
      @Override
      public void afterCommit() {
        cacheEvictor.evictCategoriesCache();
      }
    });
    return dtoMapper.toCategoryResponse(saved);
  }

  // ── Admin Read ──────────────────────────────────────────────────────

  @Override
  @Cacheable(value = "catalog:categories", key = "'all'")
  public AdminProductListResponse getAdminProducts(
      int page, int pageSize, String q, String status) {
    log.debug(
        "[CATALOG-ADMIN] getAdminProducts: page={}, pageSize={}, q={}, status={}",
        page,
        pageSize,
        q,
        status);

    int offset = (page - 1) * pageSize;
    List<Product> products = productMapper.findAdminProducts(q, status, offset, pageSize);

    List<UUID> productIds = products.stream().map(Product::getId).collect(Collectors.toList());

    Map<UUID, List<ProductVariant>> variantsByProduct = new HashMap<>();
    if (!productIds.isEmpty()) {
      List<ProductVariant> allVariants = variantMapper.findByProductIds(productIds);
      for (ProductVariant v : allVariants) {
        variantsByProduct.computeIfAbsent(v.getProductId(), k -> new ArrayList<>()).add(v);
      }
    }

    List<UUID> variantIds =
        variantsByProduct.values().stream()
            .flatMap(List::stream)
            .map(ProductVariant::getId)
            .collect(Collectors.toList());
    Map<UUID, ChannelPrice> currentPriceByVariant = new HashMap<>();
    if (!variantIds.isEmpty()) {
      String channel = CatalogConstants.CHANNEL_D2C_WEB;
      for (ChannelPrice price :
          channelPriceMapper.findActiveCurrentByVariantIds(variantIds, channel)) {
        currentPriceByVariant.putIfAbsent(price.getVariantId(), price);
      }
    }

    List<AdminStatusCountRow> statusRows = productMapper.countAdminProductsByStatus(q);
    Map<String, Long> statusCountMap = new HashMap<>();
    for (AdminStatusCountRow row : statusRows) {
      statusCountMap.merge(row.getStatus(), row.getCount(), Long::sum);
    }

    List<AdminCategoryRow> categoryRows = productMapper.findAdminCategories(q);

    String calculatedAt = Instant.now().toString();

    List<AdminProductListItemResponse> items =
        products.stream()
            .map(
                product ->
                    AdminCatalogMapper.toListItemResponse(
                        product,
                        variantsByProduct.getOrDefault(product.getId(), List.of()),
                        currentPriceByVariant))
            .collect(Collectors.toList());

    List<AdminProductListResponse.StatusCount> statusCounts =
        AdminCatalogMapper.toStatusCounts(statusCountMap);

    List<AdminProductListResponse.CategoryOption> categories =
        AdminCatalogMapper.toCategoryOptions(categoryRows);

    return AdminProductListResponse.builder()
        .items(items)
        .statusCounts(statusCounts)
        .categories(categories)
        .calculatedAt(calculatedAt)
        .build();
  }

  @Override
  public AdminProductDetailResponse getAdminProductDetail(String productId) {
    log.debug("[CATALOG-ADMIN] getAdminProductDetail: productId={}", productId);

    UUID id;
    try {
      id = UUID.fromString(productId);
    } catch (IllegalArgumentException e) {
      throw new CatalogException(
          ErrorCode.CATALOG_PRODUCT_NOT_FOUND, "ID không hợp lệ: " + productId);
    }

    Product product =
        productMapper
            .findById(id)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Không tìm thấy sản phẩm: " + productId));

    List<ProductImage> images = imageMapper.findAllByProductId(id);
    product.setImages(images);

    AdminProductDetailResponse response = dtoMapper.toAdminDetailResponse(product);
    response.setCategoryName(product.getCategoryName());
    SaleStatus saleStatus = SaleStatus.fromListingStatus(product.getListingStatus());
    response.setSaleStatus(saleStatus.getCode());
    response.setSaleStatusLabel(saleStatus.getLabelVn());
    response.setSlug(product.getSlug());
    response.setDescription(product.getDescription());
    response.setLongDescription(product.getDescription());
    response.setOcopStar(product.getOcopStar());
    response.setOcopCertificateNo(product.getOcopCertificateNo());
    response.setStory(product.getStory());
    response.setApprovalStatus(
        product.getApprovalStatus() == null ? null : product.getApprovalStatus().name());
    response.setListingStatus(
        product.getListingStatus() == null ? null : product.getListingStatus().getValue());
    response.setImages(
        images.stream().map(AdminCatalogMapper::toAdminImageResponse).collect(Collectors.toList()));
    response.setCoverImageUrl(AdminCatalogMapper.coverImageUrl(images));
    response.setCoverImageAlt(AdminCatalogMapper.coverImageAlt(images, product.getName()));

    List<ProductVariant> variants = variantMapper.findAllByProductId(id);
    List<UUID> variantIds =
        variants.stream().map(ProductVariant::getId).collect(Collectors.toList());
    Map<UUID, ChannelPrice> currentPriceByVariant = new HashMap<>();
    if (!variantIds.isEmpty()) {
      String channel = CatalogConstants.CHANNEL_D2C_WEB;
      for (ChannelPrice price :
          channelPriceMapper.findActiveCurrentByVariantIds(variantIds, channel)) {
        currentPriceByVariant.putIfAbsent(price.getVariantId(), price);
      }
    }

    List<AdminProductDetailResponse.AdminProductSkuResponse> skus =
        variants.stream()
            .map(
                variant ->
                    AdminCatalogMapper.toSkuResponse(
                        variant, currentPriceByVariant.get(variant.getId())))
            .collect(Collectors.toList());
    response.setSkus(skus);

    response.setSkuCount(variants.size());
    response.setOnSaleSkuCount(
        variants.stream()
            .filter(v -> v.getListingStatus() == ListingStatus.ACTIVE)
            .mapToInt(v -> 1)
            .sum());
    response.setBasePriceFromVnd(
        AdminCatalogMapper.skuBasePriceFromVnd(variants, currentPriceByVariant));
    response.setFoodInformationComplete(AdminCatalogMapper.foodInformationComplete(product));
    response.setUpdatedAt(AdminCatalogMapper.formatInstant(product.getUpdatedAt()));
    response.setRevision(product.getVersion());

    return response;
  }

  // ── Admin Write ─────────────────────────────────────────────────────

  @Override
  @Transactional
  public ProductDetailResponse createProduct(CreateProductRequest request, String actorId) {
    log.info("[CATALOG-ADMIN] createProduct: slug={}, actor={}", request.getSlug(), actorId);

    if (productMapper.existsBySlug(request.getSlug(), null)) {
      throw new CatalogException(
          ErrorCode.CATALOG_SLUG_DUPLICATE, "Slug đã được sử dụng: " + request.getSlug());
    }
    categoryMapper
        .findById(request.getCategoryId())
        .orElseThrow(
            () ->
                new CatalogException(
                    ErrorCode.CATALOG_CATEGORY_NOT_FOUND,
                    "Category không tồn tại: " + request.getCategoryId()));

    Product product =
        Product.builder()
            .id(UUID.randomUUID())
            .categoryId(request.getCategoryId())
            .name(request.getName())
            .slug(request.getSlug())
            .description(request.getDescription())
            .ingredients(request.getIngredients())
            .storageGuide(request.getStorageGuide())
            .ocopStar(request.getOcopStar())
            .ocopCertificateNo(request.getOcopCertificateNo())
            .story(request.getStory())
            .approvalStatus(ApprovalStatus.DRAFT)
            .listingStatus(ListingStatus.ACTIVE)
            .version(0L)
            .createdBy(actorId)
            .build();

    productMapper.insert(product);
    outboxWriter.enqueue(
        "Product",
        product.getId().toString(),
        "vn.omama.catalog.product.created.v1",
        Map.of(
            "product_id", product.getId().toString(),
            "category_id", product.getCategoryId().toString(),
            "name", product.getName(),
            "slug", product.getSlug(),
            "approval_status", product.getApprovalStatus().name(),
            "listing_status", product.getListingStatus().name(),
            "version", product.getVersion()));
    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(product);
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse updateProduct(
      UUID productId, UpdateProductRequest request, String actorId) {
    log.info("[CATALOG-ADMIN] updateProduct: id={}, version={}", productId, request.getVersion());

    Product existing =
        productMapper
            .findById(productId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Product không tồn tại: " + productId));

    if (request.getSlug() != null && !request.getSlug().equals(existing.getSlug())) {
      if (productMapper.existsBySlug(request.getSlug(), productId)) {
        throw new CatalogException(
            ErrorCode.CATALOG_SLUG_DUPLICATE, "Slug đã được sử dụng: " + request.getSlug());
      }
    }

    if (request.getCategoryId() != null) {
      requireCategory(request.getCategoryId());
      existing.setCategoryId(request.getCategoryId());
    }
    if (request.getName() != null) existing.setName(request.getName());
    if (request.getSlug() != null) existing.setSlug(request.getSlug());
    if (request.getDescription() != null) existing.setDescription(request.getDescription());
    if (request.getIngredients() != null) existing.setIngredients(request.getIngredients());
    if (request.getStorageGuide() != null) existing.setStorageGuide(request.getStorageGuide());
    if (request.getOcopStar() != null) existing.setOcopStar(request.getOcopStar());
    if (request.getOcopCertificateNo() != null)
      existing.setOcopCertificateNo(request.getOcopCertificateNo());
    if (request.getStory() != null) existing.setStory(request.getStory());
    existing.setVersion(request.getVersion());

    int updated = productMapper.update(existing);
    if (updated == 0) throw CatalogException.versionConflict(productId.toString());

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse patchProduct(
      UUID productId, PatchProductRequest request, String actorId) {
    log.info(
        "[CATALOG-ADMIN] patchProduct: id={}, expectedVersion={}",
        productId,
        request.getExpectedVersion());

    Product existing = findProductOrThrow(productId);

    if (request.getCategoryId() != null) {
      requireCategory(request.getCategoryId());
      existing.setCategoryId(request.getCategoryId());
    }
    if (request.getName() != null) {
      existing.setName(requireText(request.getName(), "name"));
    }
    if (request.getSlug() != null) {
      String slug = requireText(request.getSlug(), "slug");
      if (!slug.equals(existing.getSlug()) && productMapper.existsBySlug(slug, productId)) {
        throw new CatalogException(
            ErrorCode.CATALOG_SLUG_DUPLICATE, "Slug đã được sử dụng: " + slug);
      }
      existing.setSlug(slug);
    }
    applyText(request.getDescription(), existing::setDescription);
    applyText(request.getIngredients(), existing::setIngredients);
    applyText(request.getStorageGuide(), existing::setStorageGuide);
    applyText(request.getOcopCertificateNo(), existing::setOcopCertificateNo);
    applyText(request.getStory(), existing::setStory);
    if (request.getOcopStar() != null) {
      existing.setOcopStar(request.getOcopStar());
    }

    existing.setVersion(request.getExpectedVersion());
    if (productMapper.update(existing) == 0) {
      throw CatalogException.versionConflict(productId.toString());
    }

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  public ProductDetailResponse submitProductForReview(
      UUID productId, String actorId, long expectedVersion) {
    log.info("[CATALOG-ADMIN] submitForReview: id={}", productId);
    Product product =
        productMapper
            .findById(productId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Product không tồn tại: " + productId));

    if (product.getApprovalStatus() != ApprovalStatus.DRAFT
        && product.getApprovalStatus() != ApprovalStatus.REJECTED) {
      throw new CatalogException(
          ErrorCode.CATALOG_PRODUCT_INVALID_STATE,
          "Chỉ có thể submit product ở trạng thái DRAFT hoặc REJECTED");
    }
    int updated =
        productMapper.updateApprovalStatus(
            productId, ApprovalStatus.PENDING.getValue(), actorId, expectedVersion);
    if (updated == 0) throw CatalogException.versionConflict(productId.toString());

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse approveProduct(
      UUID productId, String approverActorId, long expectedVersion) {
    log.info("[CATALOG-ADMIN] approveProduct: id={}, approver={}", productId, approverActorId);
    Product product =
        productMapper
            .findById(productId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Product không tồn tại: " + productId));

    if (product.getApprovalStatus() != ApprovalStatus.PENDING) {
      throw new CatalogException(
          ErrorCode.CATALOG_PRODUCT_INVALID_STATE,
          "Chỉ có thể approve product ở trạng thái PENDING");
    }
    int updated =
        productMapper.updateApprovalStatus(
            productId, ApprovalStatus.APPROVED.getValue(), approverActorId, expectedVersion);
    if (updated == 0) throw CatalogException.versionConflict(productId.toString());

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  public ProductDetailResponse rejectProduct(
      UUID productId, String approverActorId, long expectedVersion) {
    Product product =
        productMapper
            .findById(productId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Product không tồn tại: " + productId));
    if (product.getApprovalStatus() != ApprovalStatus.PENDING) {
      throw new CatalogException(
          ErrorCode.CATALOG_PRODUCT_INVALID_STATE,
          "Chỉ có thể reject product ở trạng thái PENDING");
    }
    int updated =
        productMapper.updateApprovalStatus(
            productId, ApprovalStatus.REJECTED.getValue(), approverActorId, expectedVersion);
    if (updated == 0) throw CatalogException.versionConflict(productId.toString());

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse suspendProduct(
      UUID productId, String actorId, long expectedVersion) {
    Product product =
        productMapper
            .findById(productId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Product không tồn tại: " + productId));
    if (product.getListingStatus() != ListingStatus.ACTIVE) {
      throw new CatalogException(
          ErrorCode.CATALOG_PRODUCT_INVALID_STATE, "Chỉ có thể suspend product đang ACTIVE");
    }
    int updated =
        productMapper.updateListingStatus(
            productId, ListingStatus.SUSPENDED.getValue(), expectedVersion);
    if (updated == 0) throw CatalogException.versionConflict(productId.toString());

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse archiveProduct(
      UUID productId, String actorId, long expectedVersion) {
    Product product =
        productMapper
            .findById(productId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Product không tồn tại: " + productId));
    if (product.getListingStatus() == ListingStatus.ARCHIVED) {
      throw new CatalogException(
          ErrorCode.CATALOG_PRODUCT_INVALID_STATE, "Product đã ở trạng thái ARCHIVED");
    }
    int updated =
        productMapper.updateListingStatus(
            productId, ListingStatus.ARCHIVED.getValue(), expectedVersion);
    if (updated == 0) throw CatalogException.versionConflict(productId.toString());

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  public ProductDetailResponse addVariant(
      UUID productId, CreateVariantRequest request, String actorId) {
    log.info("[CATALOG-ADMIN] addVariant: productId={}, sku={}", productId, request.getSkuCode());

    productMapper
        .findById(productId)
        .orElseThrow(
            () ->
                new CatalogException(
                    ErrorCode.CATALOG_PRODUCT_NOT_FOUND, "Product không tồn tại: " + productId));

    if (variantMapper.existsBySkuCode(request.getSkuCode(), null)) {
      throw new CatalogException(
          ErrorCode.CATALOG_SKU_DUPLICATE, "SKU đã tồn tại: " + request.getSkuCode());
    }

    UUID variantId = UUID.randomUUID();
    ProductVariant variant =
        ProductVariant.builder()
            .id(variantId)
            .productId(productId)
            .skuCode(request.getSkuCode())
            .variantName(request.getVariantName())
            .weightValue(request.getWeightValue())
            .weightUnit(request.getWeightUnit())
            .flavor(request.getFlavor())
            .packagingType(request.getPackagingType())
            .shelfLifeDays(request.getShelfLifeDays())
            .listingStatus(ListingStatus.ACTIVE)
            .version(0L)
            .build();
    variantMapper.insert(variant);

    if (request.getBasePriceUnits() != null) {
      channelPriceMapper.insert(
          ChannelPrice.builder()
              .id(UUID.randomUUID())
              .variantId(variantId)
              .channel(SalesChannel.D2C_WEB)
              .currencyCode("VND")
              .amountUnits(request.getBasePriceUnits())
              .amountNanos(0)
              .status("ACTIVE")
              .effectiveFrom(Instant.now())
              .createdBy(actorId)
              .build());
    }

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(productMapper.findById(productId).orElseThrow());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse patchVariant(
      UUID productId, UUID variantId, PatchVariantRequest request, String actorId) {
    log.info(
        "[CATALOG-ADMIN] patchVariant: productId={}, variantId={}, expectedVersion={}",
        productId,
        variantId,
        request.getExpectedVersion());

    findProductOrThrow(productId);
    ProductVariant variant = findVariantOfProduct(productId, variantId);

    if (request.getVariantName() != null) {
      variant.setVariantName(requireText(request.getVariantName(), "variantName"));
    }
    if (request.getWeightValue() != null) {
      variant.setWeightValue(request.getWeightValue());
    }
    if (request.getWeightUnit() != null) {
      String unit = requireText(request.getWeightUnit(), "weightUnit").toUpperCase(Locale.ROOT);
      if (!CatalogConstants.WEIGHT_UNITS.contains(unit)) {
        throw new CatalogException(
            ErrorCode.CATALOG_VARIANT_INVALID,
            "weightUnit không hợp lệ: " + request.getWeightUnit());
      }
      variant.setWeightUnit(unit);
    }
    applyText(request.getFlavor(), variant::setFlavor);
    if (request.getPackagingType() != null) {
      String packaging =
          requireText(request.getPackagingType(), "packagingType").toUpperCase(Locale.ROOT);
      if (!CatalogConstants.PACKAGING_TYPES.contains(packaging)) {
        throw new CatalogException(
            ErrorCode.CATALOG_VARIANT_INVALID,
            "packagingType không hợp lệ: " + request.getPackagingType());
      }
      variant.setPackagingType(packaging);
    }
    if (request.getShelfLifeDays() != null) {
      variant.setShelfLifeDays(request.getShelfLifeDays());
    }
    if (request.getSaleStatus() != null) {
      ListingStatus status = SaleStatus.toListingStatus(request.getSaleStatus());
      if (status == null) {
        throw new CatalogException(
            ErrorCode.CATALOG_VARIANT_INVALID,
            "saleStatus không hợp lệ: " + request.getSaleStatus());
      }
      variant.setListingStatus(status);
    }

    variant.setVersion(request.getExpectedVersion());
    if (variantMapper.update(variant) == 0) {
      throw CatalogException.versionConflict(variantId.toString());
    }

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional(readOnly = true)
  public List<AdminPriceResponse> getVariantPrices(UUID productId, UUID variantId) {
    findProductOrThrow(productId);
    findVariantOfProduct(productId, variantId);

    return channelPriceMapper.findAllByVariantId(variantId).stream()
        .map(
            price ->
                AdminPriceResponse.builder()
                    .priceId(price.getId() == null ? null : price.getId().toString())
                    .channel(price.getChannel() == null ? null : price.getChannel().name())
                    .currencyCode(price.getCurrencyCode())
                    .amountUnits(price.getAmountUnits())
                    .amountNanos(price.getAmountNanos())
                    .status(price.getStatus())
                    .effectiveFrom(formatInstant(price.getEffectiveFrom()))
                    .effectiveTo(formatInstant(price.getEffectiveTo()))
                    .revision(price.getVersion())
                    .build())
        .collect(Collectors.toList());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse addImage(
      UUID productId, CreateProductImageRequest request, String actorId) {
    findProductOrThrow(productId);

    String role = normalizeEnum("GALLERY", request.getRole(), CatalogConstants.IMAGE_ROLES, "role");
    String mediaStatus =
        normalizeEnum(
            "PENDING_SCAN",
            request.getMediaStatus(),
            CatalogConstants.IMAGE_STATUSES,
            "mediaStatus");
    int position = request.getPosition() == null ? 0 : request.getPosition();

    if (isApprovedCover(role, mediaStatus)) {
      imageMapper.demoteOtherApprovedCovers(productId, null);
    }

    imageMapper.insert(
        ProductImage.builder()
            .id(UUID.randomUUID())
            .productId(productId)
            .objectKey(requireText(request.getObjectKey(), "objectKey"))
            .altText(trimToNull(request.getAltText()))
            .role(role)
            .position(position)
            .mediaStatus(mediaStatus)
            .version(0L)
            .build());

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse updateImage(
      UUID productId, UUID imageId, UpdateProductImageRequest request, String actorId) {
    findProductOrThrow(productId);
    ProductImage image = findImageOfProduct(productId, imageId);

    if (request.getAltText() != null) {
      image.setAltText(trimToNull(request.getAltText()));
    }
    if (request.getRole() != null) {
      image.setRole(
          normalizeEnum(image.getRole(), request.getRole(), CatalogConstants.IMAGE_ROLES, "role"));
    }
    if (request.getMediaStatus() != null) {
      image.setMediaStatus(
          normalizeEnum(
              image.getMediaStatus(),
              request.getMediaStatus(),
              CatalogConstants.IMAGE_STATUSES,
              "mediaStatus"));
    }
    if (request.getPosition() != null) {
      image.setPosition(request.getPosition());
    }

    if (isApprovedCover(image.getRole(), image.getMediaStatus())) {
      imageMapper.demoteOtherApprovedCovers(productId, imageId);
    }

    image.setVersion(request.getExpectedVersion());
    if (imageMapper.update(image) == 0) {
      throw CatalogException.versionConflict(imageId.toString());
    }

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse deleteImage(UUID productId, UUID imageId, String actorId) {
    findProductOrThrow(productId);
    ProductImage image = findImageOfProduct(productId, imageId);
    imageMapper.deleteById(image.getId());
    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(findProductOrThrow(productId));
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public ProductDetailResponse updateVariantPrices(
      UUID productId, UUID variantId, UpdatePriceRequest request, String actorId) {
    log.info("[CATALOG-ADMIN] updateVariantPrices: variantId={}", variantId);

    ProductVariant variant =
        variantMapper
            .findById(variantId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_VARIANT_NOT_FOUND,
                        "Variant không tồn tại: " + variantId));

    if (!variant.getProductId().equals(productId)) {
      throw new CatalogException(
          ErrorCode.CATALOG_VARIANT_MISMATCH,
          "Variant " + variantId + " không thuộc Product " + productId);
    }

    List<Map<String, Object>> changedPrices =
        request.getPrices().stream()
            .map(
                entry -> {
                  channelPriceMapper.supersedePreviousPrice(
                      variantId, entry.getChannel().getValue());
                  channelPriceMapper.insert(
                      ChannelPrice.builder()
                          .id(UUID.randomUUID())
                          .variantId(variantId)
                          .channel(entry.getChannel())
                          .currencyCode(entry.getCurrencyCode())
                          .amountUnits(entry.getAmountUnits())
                          .amountNanos(0)
                          .status("ACTIVE")
                          .effectiveFrom(Instant.now())
                          .createdBy(actorId)
                          .build());
                  Map<String, Object> price = new LinkedHashMap<>();
                  price.put("channel", entry.getChannel().name());
                  price.put("currency_code", entry.getCurrencyCode());
                  price.put("amount_units", entry.getAmountUnits());
                  price.put("amount_nanos", 0);
                  return price;
                })
            .toList();
    Map<String, Object> eventData = new LinkedHashMap<>();
    eventData.put("product_id", productId.toString());
    eventData.put("variant_id", variantId.toString());
    eventData.put("sku_code", variant.getSkuCode());
    eventData.put("prices", changedPrices);
    eventData.put("changed_by", actorId);
    outboxWriter.enqueue(
        "Product", productId.toString(), "vn.omama.catalog.product.price.changed.v1", eventData);

    cacheEvictor.evictCategoriesCache();
    return buildDetailResponse(productMapper.findById(productId).orElseThrow());
  }

  // ── Private helpers ───────────────────────────────────────────────────────

  private ProductDetailResponse buildDetailResponse(Product product) {
    ProductDetailResponse response = dtoMapper.toDetailResponse(product);
    List<ProductVariant> variants = variantMapper.findAllByProductId(product.getId());
    List<ProductVariantResponse> variantResponses =
        variants.stream()
            .map(
                v -> {
                  ProductVariantResponse vr = dtoMapper.toVariantResponse(v);
                  channelPriceMapper
                      .findActiveByVariantAndChannel(v.getId(), SalesChannel.D2C_WEB.getValue())
                      .ifPresent(price -> vr.setPrice(dtoMapper.toMoneyResponse(price)));
                  return vr;
                })
            .collect(Collectors.toList());
    response.setVariants(variantResponses);
    response.setImageUrls(
        imageMapper.findApprovedByProductId(product.getId()).stream()
            .map(image -> AdminCatalogMapper.imageUrl(image.getObjectKey()))
            .filter(Objects::nonNull)
            .collect(Collectors.toList()));
    return response;
  }

  private Product findProductOrThrow(UUID productId) {
    return productMapper
        .findById(productId)
        .orElseThrow(
            () ->
                new CatalogException(
                    ErrorCode.CATALOG_PRODUCT_NOT_FOUND, "Product không tồn tại: " + productId));
  }

  private ProductVariant findVariantOfProduct(UUID productId, UUID variantId) {
    ProductVariant variant =
        variantMapper
            .findById(variantId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_VARIANT_NOT_FOUND,
                        "Variant không tồn tại: " + variantId));
    if (!variant.getProductId().equals(productId)) {
      throw new CatalogException(
          ErrorCode.CATALOG_VARIANT_MISMATCH,
          "Variant " + variantId + " không thuộc Product " + productId);
    }
    return variant;
  }

  private ProductImage findImageOfProduct(UUID productId, UUID imageId) {
    ProductImage image =
        imageMapper
            .findById(imageId)
            .orElseThrow(
                () -> new CatalogException(ErrorCode.CATALOG_IMAGE_NOT_FOUND, "Ảnh không tồn tại"));
    if (!image.getProductId().equals(productId)) {
      throw new CatalogException(
          ErrorCode.CATALOG_IMAGE_NOT_FOUND,
          "Ảnh " + imageId + " không thuộc Product " + productId);
    }
    return image;
  }

  private void requireCategory(UUID categoryId) {
    categoryMapper
        .findById(categoryId)
        .orElseThrow(
            () ->
                new CatalogException(
                    ErrorCode.CATALOG_CATEGORY_NOT_FOUND, "Category không tồn tại: " + categoryId));
  }

  private static String requireText(String value, String field) {
    if (value == null || value.isBlank()) {
      throw new CatalogException(ErrorCode.INVALID_ARGUMENT, field + " không được để trống");
    }
    return value.trim();
  }

  private static void applyText(String value, Consumer<String> setter) {
    if (value != null) {
      setter.accept(trimToNull(value));
    }
  }

  private static String trimToNull(String value) {
    if (value == null) {
      return null;
    }
    String trimmed = value.trim();
    return trimmed.isEmpty() ? null : trimmed;
  }

  private static String normalizeEnum(
      String defaultValue, String value, java.util.Set<String> allowed, String field) {
    if (value == null) {
      return defaultValue;
    }
    String normalized = value.trim().toUpperCase(Locale.ROOT);
    if (!allowed.contains(normalized)) {
      throw new CatalogException(
          ErrorCode.CATALOG_IMAGE_INVALID, field + " không hợp lệ: " + value);
    }
    return normalized;
  }

  private static boolean isApprovedCover(String role, String mediaStatus) {
    return "COVER".equals(role) && "APPROVED".equals(mediaStatus);
  }

  private static String formatInstant(Instant instant) {
    return instant == null ? null : instant.toString();
  }
}
