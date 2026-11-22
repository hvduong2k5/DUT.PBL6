package com.hvduong.catalog.application.service.impl;

import com.hvduong.catalog.application.dto.request.CreateCategoryRequest;
import com.hvduong.catalog.application.dto.request.CreatePriceScheduleRequest;
import com.hvduong.catalog.application.dto.request.CreateProductImageRequest;
import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.PatchCategoryRequest;
import com.hvduong.catalog.application.dto.request.PatchProductRequest;
import com.hvduong.catalog.application.dto.request.PatchVariantRequest;
import com.hvduong.catalog.application.dto.request.ReviewProductImageRequest;
import com.hvduong.catalog.application.dto.request.UpdateCategoryRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductImageRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminCategoryItem;
import com.hvduong.catalog.application.dto.response.AdminCategoryListResponse;
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
import com.hvduong.catalog.repository.mybatis.projection.CategoryCountRow;

import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;
import java.util.function.Consumer;
import java.util.function.Function;
import java.util.stream.Collectors;
import java.util.stream.Stream;

import org.springframework.cache.annotation.CacheEvict;
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
  public CategoryResponse createCategory(
      CreateCategoryRequest request, String actorId, String key) {
    if (!key.matches("[!-~]{16,128}")) {
      throw new CatalogException(ErrorCode.INVALID_ARGUMENT);
    }
    UUID categoryId =
        UUID.nameUUIDFromBytes(
            ("catalog:create-category:" + actorId + ":" + key).getBytes(StandardCharsets.UTF_8));
    Category category =
        Category.builder()
            .id(categoryId)
            .parentId(request.parentId())
            .name(request.name())
            .slug(request.slug())
            .description(request.description())
            .imageUrl(request.imageUrl())
            .position(request.position() == null ? 0 : request.position())
            .isActive(true)
            .build();
    if (category.getParentId() != null) {
      validateCategoryParent(categoryId, category.getParentId());
    }

    Category saved = category;
    if (categoryMapper.insert(category) == 0) {
      Category existing =
          categoryMapper
              .findById(categoryId)
              .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_CATEGORY_SLUG_DUPLICATE));
      if (!Objects.equals(existing.getParentId(), category.getParentId())
          || !Objects.equals(existing.getName(), category.getName())
          || !Objects.equals(existing.getSlug(), category.getSlug())
          || !Objects.equals(existing.getDescription(), category.getDescription())
          || !Objects.equals(existing.getImageUrl(), category.getImageUrl())
          || existing.getPosition() != category.getPosition()) {
        throw new CatalogException(ErrorCode.IDEMPOTENCY_CONFLICT);
      }
      saved = existing;
    }
    TransactionSynchronizationManager.registerSynchronization(
        new TransactionSynchronization() {
          @Override
          public void afterCommit() {
            cacheEvictor.evictCategoriesCache();
          }
        });
    return dtoMapper.toCategoryResponse(saved);
  }

  @Override
  @Transactional
  public CategoryResponse updateCategory(
      UUID categoryId, UpdateCategoryRequest request, String actorId) {
    Category category =
        categoryMapper
            .findById(categoryId)
            .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_CATEGORY_NOT_FOUND));

    if (categoryMapper.existsBySlug(request.getSlug(), categoryId)) {
      throw new CatalogException(ErrorCode.CATALOG_CATEGORY_SLUG_DUPLICATE);
    }

    if (request.getParentId() != null) {
      validateCategoryParent(categoryId, request.getParentId());
    }

    category.setName(request.getName().trim());
    category.setSlug(request.getSlug().trim());
    category.setParentId(request.getParentId());
    category.setDescription(trimToNull(request.getDescription()));
    category.setImageUrl(trimToNull(request.getImageUrl()));
    if (request.getPosition() != null) {
      category.setPosition(request.getPosition());
    }
    if (request.getIsActive() != null) {
      category.setActive(request.getIsActive());
    }
    category.setVersion(request.getExpectedVersion());

    if (categoryMapper.update(category) == 0) {
      throw CatalogException.versionConflict(categoryId.toString());
    }

    TransactionSynchronizationManager.registerSynchronization(
        new TransactionSynchronization() {
          @Override
          public void afterCommit() {
            cacheEvictor.evictCategoriesCache();
          }
        });

    return dtoMapper.toCategoryResponse(categoryMapper.findById(categoryId).orElse(category));
  }

  @Override
  @Transactional(readOnly = true)
  public AdminCategoryListResponse getAdminCategories(
      String format, String q, Boolean isActive, UUID parentId) {
    String resolvedFormat =
        (format == null || format.isBlank()) ? "tree" : format.trim().toLowerCase(Locale.ROOT);
    boolean isTree = "tree".equals(resolvedFormat);
    boolean isFlat = "flat".equals(resolvedFormat);

    if (!isTree && !isFlat) {
      throw new CatalogException(
          ErrorCode.INVALID_ARGUMENT, "format chỉ chấp nhận giá trị 'tree' hoặc 'flat'");
    }

    boolean hasQuery = (q != null && !q.isBlank());
    if (isTree && hasQuery) {
      throw new CatalogException(ErrorCode.CATALOG_CATEGORY_SEARCH_REQUIRES_FLAT_FORMAT);
    }

    List<Category> allCategories = categoryMapper.findAllAdmin();
    Map<UUID, Long> productCountMap =
        categoryMapper.countProductsGroupedByCategoryId().stream()
            .collect(Collectors.toMap(CategoryCountRow::getCategoryId, CategoryCountRow::getCount));

    Map<UUID, Long> childrenCountMap =
        allCategories.stream()
            .filter(c -> c.getParentId() != null)
            .collect(Collectors.groupingBy(Category::getParentId, Collectors.counting()));

    Map<UUID, Category> categoryById =
        allCategories.stream().collect(Collectors.toMap(Category::getId, c -> c));

    Function<Category, AdminCategoryItem> toItem =
        c -> {
          String parentName =
              (c.getParentId() != null && categoryById.containsKey(c.getParentId()))
                  ? categoryById.get(c.getParentId()).getName()
                  : null;
          return AdminCategoryItem.builder()
              .categoryId(c.getId())
              .parentId(c.getParentId())
              .parentName(parentName)
              .name(c.getName())
              .slug(c.getSlug())
              .description(c.getDescription())
              .imageUrl(c.getImageUrl())
              .position(c.getPosition())
              .isActive(c.isActive())
              .productCount(productCountMap.getOrDefault(c.getId(), 0L))
              .childrenCount(childrenCountMap.getOrDefault(c.getId(), 0L))
              .version(c.getVersion())
              .createdAt(c.getCreatedAt())
              .updatedAt(c.getUpdatedAt())
              .children(new ArrayList<>())
              .build();
        };

    if (isFlat) {
      Stream<Category> stream = allCategories.stream();
      if (isActive != null) {
        stream = stream.filter(c -> c.isActive() == isActive);
      }
      if (parentId != null) {
        stream = stream.filter(c -> parentId.equals(c.getParentId()));
      }
      if (hasQuery) {
        String query = q.trim().toLowerCase(Locale.ROOT);
        stream =
            stream.filter(
                c ->
                    (c.getName() != null && c.getName().toLowerCase(Locale.ROOT).contains(query))
                        || (c.getSlug() != null
                            && c.getSlug().toLowerCase(Locale.ROOT).contains(query)));
      }
      List<AdminCategoryItem> items =
          stream
              .map(toItem)
              .sorted(
                  Comparator.comparingInt(AdminCategoryItem::getPosition)
                      .thenComparing(AdminCategoryItem::getName))
              .collect(Collectors.toList());
      return AdminCategoryListResponse.builder().categories(items).build();
    }

    // Tree Mode:
    Stream<Category> stream = allCategories.stream();
    if (isActive != null) {
      stream = stream.filter(c -> c.isActive() == isActive);
    }
    List<Category> filteredCategories = stream.toList();
    Map<UUID, AdminCategoryItem> itemMap =
        filteredCategories.stream()
            .map(toItem)
            .collect(Collectors.toMap(AdminCategoryItem::getCategoryId, item -> item));

    for (AdminCategoryItem item : itemMap.values()) {
      if (item.getParentId() != null && itemMap.containsKey(item.getParentId())) {
        itemMap.get(item.getParentId()).getChildren().add(item);
      }
    }

    for (AdminCategoryItem item : itemMap.values()) {
      item.getChildren()
          .sort(
              Comparator.comparingInt(AdminCategoryItem::getPosition)
                  .thenComparing(AdminCategoryItem::getName));
    }

    List<AdminCategoryItem> rootItems;
    if (parentId != null) {
      if (itemMap.containsKey(parentId)) {
        rootItems = itemMap.get(parentId).getChildren();
      } else {
        rootItems =
            itemMap.values().stream()
                .filter(item -> parentId.equals(item.getParentId()))
                .sorted(
                    Comparator.comparingInt(AdminCategoryItem::getPosition)
                        .thenComparing(AdminCategoryItem::getName))
                .collect(Collectors.toList());
      }
    } else {
      rootItems =
          itemMap.values().stream()
              .filter(
                  item -> item.getParentId() == null || !itemMap.containsKey(item.getParentId()))
              .sorted(
                  Comparator.comparingInt(AdminCategoryItem::getPosition)
                      .thenComparing(AdminCategoryItem::getName))
              .collect(Collectors.toList());
    }

    return AdminCategoryListResponse.builder().categories(rootItems).build();
  }

  @Override
  @Transactional(readOnly = true)
  public CategoryResponse getCategory(UUID categoryId) {
    Category category =
        categoryMapper
            .findById(categoryId)
            .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_CATEGORY_NOT_FOUND));
    return dtoMapper.toCategoryResponse(category);
  }

  @Override
  @Transactional
  public CategoryResponse patchCategory(
      UUID categoryId, PatchCategoryRequest request, String actorId) {
    Category category =
        categoryMapper
            .findById(categoryId)
            .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_CATEGORY_NOT_FOUND));

    if (request.getSlug() != null && !request.getSlug().isBlank()) {
      String newSlug = request.getSlug().trim();
      if (!newSlug.equals(category.getSlug()) && categoryMapper.existsBySlug(newSlug, categoryId)) {
        throw new CatalogException(ErrorCode.CATALOG_CATEGORY_SLUG_DUPLICATE);
      }
      category.setSlug(newSlug);
    }

    if (request.getName() != null && !request.getName().isBlank()) {
      category.setName(request.getName().trim());
    }

    if (request.getParentId() != null) {
      validateCategoryParent(categoryId, request.getParentId());
      category.setParentId(request.getParentId());
    }

    if (request.getDescription() != null) {
      category.setDescription(trimToNull(request.getDescription()));
    }
    if (request.getImageUrl() != null) {
      category.setImageUrl(trimToNull(request.getImageUrl()));
    }
    if (request.getPosition() != null) {
      category.setPosition(request.getPosition());
    }
    if (request.getIsActive() != null) {
      category.setActive(request.getIsActive());
    }

    category.setVersion(request.getExpectedVersion());

    if (categoryMapper.update(category) == 0) {
      throw CatalogException.versionConflict(categoryId.toString());
    }

    TransactionSynchronizationManager.registerSynchronization(
        new TransactionSynchronization() {
          @Override
          public void afterCommit() {
            cacheEvictor.evictCategoriesCache();
          }
        });

    return dtoMapper.toCategoryResponse(categoryMapper.findById(categoryId).orElse(category));
  }

  @Override
  @Transactional
  public void deleteCategory(UUID categoryId, String actorId) {
    categoryMapper
        .findById(categoryId)
        .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_CATEGORY_NOT_FOUND));

    if (categoryMapper.countChildrenByParentId(categoryId) > 0) {
      throw new CatalogException(
          ErrorCode.CATALOG_CATEGORY_IN_USE, "Không thể xóa danh mục đang có danh mục con");
    }

    if (categoryMapper.countProductsByCategoryId(categoryId) > 0) {
      throw new CatalogException(
          ErrorCode.CATALOG_CATEGORY_IN_USE, "Không thể xóa danh mục đang gắn với sản phẩm");
    }

    categoryMapper.deleteById(categoryId);

    TransactionSynchronizationManager.registerSynchronization(
        new TransactionSynchronization() {
          @Override
          public void afterCommit() {
            cacheEvictor.evictCategoriesCache();
          }
        });
  }

  @Override
  @Transactional(readOnly = true)
  public ProductVariantResponse getVariant(UUID productId, UUID variantId) {
    findProductOrThrow(productId);
    ProductVariant variant = findVariantOfProduct(productId, variantId);
    ProductVariantResponse response = dtoMapper.toVariantResponse(variant);
    channelPriceMapper
        .findActiveByVariantAndChannel(variant.getId(), SalesChannel.D2C_WEB.getValue())
        .ifPresent(price -> response.setPrice(dtoMapper.toMoneyResponse(price)));
    return response;
  }

  // ── Admin Read ──────────────────────────────────────────────────────

  @Override
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
    if (images == null) {
      images = List.of();
    }
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
    if (variants == null) {
      variants = List.of();
    }
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
  public AdminProductDetailResponse createProduct(CreateProductRequest request, String actorId) {
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
    return getAdminProductDetail(product.getId().toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse updateProduct(
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
    Long expectedVersion =
        request.getExpectedVersion() != null ? request.getExpectedVersion() : request.getVersion();
    if (expectedVersion != null) {
      existing.setVersion(expectedVersion);
    }

    int updated = productMapper.update(existing);
    if (updated == 0) throw CatalogException.versionConflict(productId.toString());

    cacheEvictor.evictCategoriesCache();
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse patchProduct(
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
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  public AdminProductDetailResponse submitProductForReview(
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
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse approveProduct(
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
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  public AdminProductDetailResponse rejectProduct(
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
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse suspendProduct(
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
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse archiveProduct(
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
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  public AdminProductDetailResponse addVariant(
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
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse patchVariant(
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
    return getAdminProductDetail(productId.toString());
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
  public AdminProductDetailResponse addImage(
      UUID productId, CreateProductImageRequest request, String actorId) {
    findProductOrThrow(productId);

    String role = normalizeEnum("GALLERY", request.getRole(), CatalogConstants.IMAGE_ROLES, "role");
    int position = request.getPosition() == null ? 0 : request.getPosition();

    imageMapper.insert(
        ProductImage.builder()
            .id(UUID.randomUUID())
            .productId(productId)
            .objectKey(requireText(request.getObjectKey(), "objectKey"))
            .altText(trimToNull(request.getAltText()))
            .role(role)
            .position(position)
            .mediaStatus("PENDING_SCAN")
            .scanStatus("PENDING")
            .uploadedBy(actorId)
            .version(0L)
            .build());

    cacheEvictor.evictCategoriesCache();
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse updateImage(
      UUID productId, UUID imageId, UpdateProductImageRequest request, String actorId) {
    findProductOrThrow(productId);
    ProductImage image = findImageOfProduct(productId, imageId);

    if (request.getAltText() != null) {
      image.setAltText(trimToNull(request.getAltText()));
    }
    if (request.getRole() != null) {
      String newRole =
          normalizeEnum(image.getRole(), request.getRole(), CatalogConstants.IMAGE_ROLES, "role");
      if ("APPROVED".equals(image.getMediaStatus())
          && "COVER".equals(newRole)
          && !"COVER".equals(image.getRole())) {
        throw new CatalogException(
            ErrorCode.CATALOG_IMAGE_APPROVED_ROLE_CHANGE_FORBIDDEN,
            "Không thể chuyển ảnh đã duyệt thành COVER qua PATCH. Thao tác cần quyền CATALOG_APPROVE.");
      }
      image.setRole(newRole);
    }
    if (request.getPosition() != null) {
      image.setPosition(request.getPosition());
    }

    image.setVersion(request.getExpectedVersion());
    if (imageMapper.update(image) == 0) {
      throw CatalogException.versionConflict(imageId.toString());
    }

    cacheEvictor.evictCategoriesCache();
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse reviewImage(
      UUID productId, UUID imageId, ReviewProductImageRequest request, String actorId) {
    // 1. Khóa dòng Product bằng FOR UPDATE để serialize concurrent cover reviews
    productMapper
        .findByIdForUpdate(productId)
        .orElseThrow(
            () ->
                new CatalogException(
                    ErrorCode.CATALOG_PRODUCT_NOT_FOUND, "Không tìm thấy sản phẩm."));

    ProductImage image = findImageOfProduct(productId, imageId);

    // 2. Kiểm tra scan_status phải là CLEAN
    if (!"CLEAN".equals(image.getScanStatus())) {
      throw new CatalogException(
          ErrorCode.CATALOG_IMAGE_NOT_SCANNED,
          "Ảnh chưa được Media Service quét an toàn mã độc hoặc quét không đạt (scan_status != CLEAN).");
    }

    // 3. Kiểm tra media_status phải là PENDING_REVIEW
    if (!"PENDING_REVIEW".equals(image.getMediaStatus())) {
      throw new CatalogException(
          ErrorCode.CATALOG_IMAGE_NOT_PENDING_REVIEW,
          "Ảnh không ở trạng thái chờ duyệt (PENDING_REVIEW). Trạng thái hiện tại: "
              + image.getMediaStatus());
    }

    // 4. Separation of Duties: Người upload không được tự phê duyệt
    if (actorId != null && actorId.equals(image.getUploadedBy())) {
      throw new CatalogException(
          ErrorCode.CATALOG_IMAGE_CANNOT_SELF_REVIEW,
          "Người tải ảnh lên không được phép tự phê duyệt ảnh (Separation of Duties).");
    }

    // 5. Xử lý quyết định
    String decision = request.getDecision().toUpperCase();
    if ("APPROVED".equals(decision)) {
      if ("COVER".equals(image.getRole())) {
        imageMapper.demoteOtherApprovedCovers(productId, imageId);
      }
      image.setMediaStatus("APPROVED");
      image.setRejectedReason(null);
    } else if ("REJECTED".equals(decision)) {
      requireText(request.getReason(), "reason");
      image.setMediaStatus("REJECTED");
      image.setRejectedReason(request.getReason().trim());
    } else {
      throw new CatalogException(
          ErrorCode.CATALOG_IMAGE_INVALID, "decision phải là APPROVED hoặc REJECTED.");
    }

    image.setReviewedBy(actorId);
    image.setReviewedAt(Instant.now());
    image.setVersion(request.getExpectedVersion());

    if (imageMapper.update(image) == 0) {
      throw CatalogException.versionConflict(imageId.toString());
    }

    cacheEvictor.evictCategoriesCache();
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  public void handleScanResult(UUID imageId, String scanStatus, Instant scannedAt) {
    ProductImage image =
        imageMapper
            .findById(imageId)
            .orElseThrow(
                () ->
                    new CatalogException(
                        ErrorCode.CATALOG_IMAGE_NOT_FOUND, "Không tìm thấy ảnh: " + imageId));

    String normalizedScanStatus =
        normalizeEnum("PENDING", scanStatus, CatalogConstants.IMAGE_SCAN_STATUSES, "scanStatus");
    Instant scanTime = scannedAt != null ? scannedAt : Instant.now();
    String newMediaStatus = image.getMediaStatus();

    if ("CLEAN".equals(normalizedScanStatus)) {
      if ("PENDING_SCAN".equals(image.getMediaStatus())) {
        newMediaStatus = "PENDING_REVIEW";
      }
    } else if ("INFECTED".equals(normalizedScanStatus) || "FAILED".equals(normalizedScanStatus)) {
      newMediaStatus = "REJECTED";
      image.setRejectedReason(
          "Quét an toàn thất bại hoặc phát hiện mã độc: " + normalizedScanStatus);
    }

    imageMapper.updateScanResult(imageId, normalizedScanStatus, newMediaStatus, scanTime);
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse deleteImage(UUID productId, UUID imageId, String actorId) {
    findProductOrThrow(productId);
    ProductImage image = findImageOfProduct(productId, imageId);
    imageMapper.deleteById(image.getId());
    cacheEvictor.evictCategoriesCache();
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminProductDetailResponse updateVariantPrices(
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

    Long expectedVersion =
        request.getExpectedVersion() != null
            ? request.getExpectedVersion()
            : request.getExpectedVariantVersion();
    if (expectedVersion != null && variant.getVersion() != expectedVersion) {
      throw CatalogException.versionConflict(variantId.toString());
    }

    List<Map<String, Object>> changedPrices = new ArrayList<>();
    Instant now = Instant.now();

    for (UpdatePriceRequest.PriceEntry entry : request.getPrices()) {
      Instant effectiveFrom = entry.getEffectiveFrom() != null ? entry.getEffectiveFrom() : now;
      if (entry.getEffectiveTo() != null && !entry.getEffectiveTo().isAfter(effectiveFrom)) {
        throw new CatalogException(
            ErrorCode.INVALID_ARGUMENT, "Thời điểm kết thúc phải sau thời điểm bắt đầu");
      }
      boolean isImmediate = !effectiveFrom.isAfter(now);
      if (isImmediate) {
        channelPriceMapper.supersedePreviousPrice(variantId, entry.getChannel().getValue());
      }
      int nanos = entry.getAmountNanos() != null ? entry.getAmountNanos() : 0;
      String currency = entry.getCurrencyCode() != null ? entry.getCurrencyCode() : "VND";

      ChannelPrice priceEntity =
          ChannelPrice.builder()
              .id(UUID.randomUUID())
              .variantId(variantId)
              .channel(entry.getChannel())
              .currencyCode(currency)
              .amountUnits(entry.getAmountUnits())
              .amountNanos(nanos)
              .status("ACTIVE")
              .effectiveFrom(effectiveFrom)
              .effectiveTo(entry.getEffectiveTo())
              .createdBy(actorId)
              .build();

      try {
        channelPriceMapper.insert(priceEntity);
      } catch (org.springframework.dao.DataIntegrityViolationException ex) {
        String msg = ex.getMessage();
        if (msg != null
            && (msg.contains("excl_channel_prices_no_overlap") || msg.contains("23P01"))) {
          throw new CatalogException(ErrorCode.CATALOG_PRICE_OVERLAP_CONFLICT);
        }
        throw ex;
      }

      if (isImmediate) {
        Map<String, Object> price = new LinkedHashMap<>();
        price.put("channel", entry.getChannel().name());
        price.put("currency_code", currency);
        price.put("amount_units", entry.getAmountUnits());
        price.put("amount_nanos", nanos);
        changedPrices.add(price);
      }
    }

    if (!changedPrices.isEmpty()) {
      variantMapper.incrementPriceRevision(variantId);
      variant = variantMapper.findById(variantId).orElseThrow();
      long currentPriceRevision = variant.getPriceRevision();
      for (Map<String, Object> p : changedPrices) {
        p.put("price_revision", currentPriceRevision);
      }
      Map<String, Object> eventData = new LinkedHashMap<>();
      eventData.put("product_id", productId.toString());
      eventData.put("variant_id", variantId.toString());
      eventData.put("sku_code", variant.getSkuCode());
      eventData.put("price_revision", currentPriceRevision);
      eventData.put("prices", changedPrices);
      eventData.put("changed_by", actorId);
      outboxWriter.enqueue(
          "Product", productId.toString(), "vn.omama.catalog.product.price.changed.v1", eventData);
    }

    cacheEvictor.evictCategoriesCache();
    return getAdminProductDetail(productId.toString());
  }

  @Override
  @Transactional
  @CacheEvict(value = "catalog:product", allEntries = true)
  public AdminPriceResponse schedulePrice(
      UUID productId, UUID skuId, CreatePriceScheduleRequest request, String actorId) {
    log.info(
        "[CATALOG-ADMIN] schedulePrice: productId={}, skuId={}, channel={}",
        productId,
        skuId,
        request.getChannel());

    findProductOrThrow(productId);
    ProductVariant variant = findVariantOfProduct(productId, skuId);

    if (request.getEffectiveTo() != null
        && !request.getEffectiveTo().isAfter(request.getEffectiveFrom())) {
      throw new CatalogException(
          ErrorCode.INVALID_ARGUMENT, "Thời điểm kết thúc phải sau thời điểm bắt đầu");
    }

    Instant effectiveFrom = request.getEffectiveFrom();
    Instant now = Instant.now();
    boolean isImmediate = !effectiveFrom.isAfter(now);

    if (isImmediate) {
      channelPriceMapper.supersedePreviousPrice(skuId, request.getChannel().getValue());
    }

    int nanos = request.getAmountNanos() != null ? request.getAmountNanos() : 0;
    String currency = request.getCurrencyCode() != null ? request.getCurrencyCode() : "VND";

    ChannelPrice newPrice =
        ChannelPrice.builder()
            .id(UUID.randomUUID())
            .variantId(skuId)
            .channel(request.getChannel())
            .currencyCode(currency)
            .amountUnits(request.getAmountUnits())
            .amountNanos(nanos)
            .status("ACTIVE")
            .effectiveFrom(effectiveFrom)
            .effectiveTo(request.getEffectiveTo())
            .createdBy(actorId)
            .build();

    try {
      channelPriceMapper.insert(newPrice);
    } catch (org.springframework.dao.DataIntegrityViolationException ex) {
      String msg = ex.getMessage();
      if (msg != null
          && (msg.contains("excl_channel_prices_no_overlap") || msg.contains("23P01"))) {
        throw new CatalogException(ErrorCode.CATALOG_PRICE_OVERLAP_CONFLICT);
      }
      throw ex;
    }

    if (isImmediate) {
      variantMapper.incrementPriceRevision(skuId);
      variant = variantMapper.findById(skuId).orElseThrow();
      long currentPriceRevision = variant.getPriceRevision();

      Map<String, Object> price = new LinkedHashMap<>();
      price.put("channel", request.getChannel().name());
      price.put("currency_code", currency);
      price.put("amount_units", request.getAmountUnits());
      price.put("amount_nanos", nanos);
      price.put("price_revision", currentPriceRevision);

      Map<String, Object> eventData = new LinkedHashMap<>();
      eventData.put("product_id", productId.toString());
      eventData.put("variant_id", skuId.toString());
      eventData.put("sku_code", variant.getSkuCode());
      eventData.put("price_revision", currentPriceRevision);
      eventData.put("prices", List.of(price));
      eventData.put("changed_by", actorId);
      outboxWriter.enqueue(
          "Product", productId.toString(), "vn.omama.catalog.product.price.changed.v1", eventData);

      cacheEvictor.evictCategoriesCache();
    }

    return AdminPriceResponse.builder()
        .priceId(newPrice.getId().toString())
        .channel(newPrice.getChannel().name())
        .currencyCode(newPrice.getCurrencyCode())
        .amountUnits(newPrice.getAmountUnits())
        .amountNanos(newPrice.getAmountNanos())
        .status(newPrice.getStatus())
        .effectiveFrom(formatInstant(newPrice.getEffectiveFrom()))
        .effectiveTo(formatInstant(newPrice.getEffectiveTo()))
        .revision(newPrice.getVersion())
        .build();
  }

  @Override
  @Transactional
  public void deleteScheduledPrice(UUID productId, UUID skuId, UUID priceId, String actorId) {
    log.info(
        "[CATALOG-ADMIN] deleteScheduledPrice: productId={}, skuId={}, priceId={}, actor={}",
        productId,
        skuId,
        priceId,
        actorId);

    findProductOrThrow(productId);
    findVariantOfProduct(productId, skuId);

    ChannelPrice price =
        channelPriceMapper
            .findById(priceId)
            .orElseThrow(() -> CatalogException.notFound("ChannelPrice", priceId.toString()));

    if (!price.getVariantId().equals(skuId)) {
      throw new CatalogException(
          ErrorCode.RESOURCE_NOT_FOUND, "Bản ghi giá không thuộc SKU: " + skuId);
    }

    Instant now = Instant.now();
    if (price.getEffectiveFrom() != null && !price.getEffectiveFrom().isAfter(now)) {
      throw new CatalogException(ErrorCode.CATALOG_PRICE_CANNOT_DELETE_ACTIVE);
    }

    int deleted = channelPriceMapper.deleteFuturePriceById(priceId, skuId);
    if (deleted == 0) {
      throw new CatalogException(ErrorCode.CATALOG_PRICE_CANNOT_DELETE_ACTIVE);
    }
  }

  // ── Private helpers ───────────────────────────────────────────────────────

  private void validateCategoryParent(UUID categoryId, UUID newParentId) {
    if (newParentId == null) {
      return;
    }
    if (newParentId.equals(categoryId)) {
      throw new CatalogException(
          ErrorCode.INVALID_ARGUMENT, "Danh mục cha không thể là chính danh mục này");
    }
    if (categoryMapper.findById(newParentId).isEmpty()) {
      throw new CatalogException(ErrorCode.CATALOG_CATEGORY_NOT_FOUND);
    }
    if (categoryId != null) {
      UUID currentParentId = newParentId;
      Set<UUID> visited = new HashSet<>();
      int depth = 0;
      while (currentParentId != null) {
        if (currentParentId.equals(categoryId)) {
          throw new CatalogException(
              ErrorCode.INVALID_ARGUMENT,
              "Không thể chọn danh mục con hoặc hậu duệ làm danh mục cha (tạo chu trình phân cấp)");
        }
        visited.add(currentParentId);
        depth++;
        if (depth > 100) {
          throw new CatalogException(
              ErrorCode.INVALID_ARGUMENT,
              "Độ sâu phân cấp danh mục vượt quá giới hạn cho phép (100)");
        }
        Category parent = categoryMapper.findById(currentParentId).orElse(null);
        currentParentId = (parent != null) ? parent.getParentId() : null;
        if (currentParentId != null && visited.contains(currentParentId)) {
          break;
        }
      }
    }
  }

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
    List<ProductImage> allImages = imageMapper.findAllByProductId(product.getId());
    if (allImages == null || allImages.isEmpty()) {
      allImages = imageMapper.findApprovedByProductId(product.getId());
    }
    if (allImages != null) {
      response.setImages(
          allImages.stream()
              .map(AdminCatalogMapper::toAdminImageResponse)
              .collect(Collectors.toList()));
      response.setImageUrls(
          allImages.stream()
              .filter(img -> "APPROVED".equalsIgnoreCase(img.getMediaStatus()))
              .map(image -> AdminCatalogMapper.imageUrl(image.getObjectKey()))
              .filter(Objects::nonNull)
              .collect(Collectors.toList()));
    }
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
