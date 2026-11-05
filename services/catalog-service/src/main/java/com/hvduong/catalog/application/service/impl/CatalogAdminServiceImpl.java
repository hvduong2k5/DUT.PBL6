package com.hvduong.catalog.application.service.impl;

import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListItemResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
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

import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.stream.Collectors;

import org.springframework.cache.annotation.CacheEvict;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

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

  // ── Admin Read ──────────────────────────────────────────────────────

  @Override
  @Cacheable(value = "catalog:categories", key = "'all'")
  public AdminProductListResponse getAdminProducts(int page, int pageSize, String q, String status) {
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

    List<ProductImage> images = imageMapper.findApprovedByProductId(id);
    product.setImages(images);

    AdminProductDetailResponse response = dtoMapper.toAdminDetailResponse(product);
    response.setCategoryName(product.getCategoryName());
    SaleStatus saleStatus = SaleStatus.fromListingStatus(product.getListingStatus());
    response.setSaleStatus(saleStatus.getCode());
    response.setSaleStatusLabel(saleStatus.getLabelVn());
    response.setLongDescription(product.getDescription());
    response.setCoverImageUrl(AdminCatalogMapper.coverImageUrl(images));
    response.setCoverImageAlt(AdminCatalogMapper.coverImageAlt(images, product.getName()));

    List<ProductVariant> variants = variantMapper.findAllByProductId(id);
    List<UUID> variantIds = variants.stream().map(ProductVariant::getId).collect(Collectors.toList());
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
    response.setBasePriceFromVnd(AdminCatalogMapper.skuBasePriceFromVnd(variants, currentPriceByVariant));
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

    if (request.getCategoryId() != null) existing.setCategoryId(request.getCategoryId());
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
    return buildDetailResponse(existing);
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
    return buildDetailResponse(product);
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
    return buildDetailResponse(product);
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
    return buildDetailResponse(product);
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
    return buildDetailResponse(product);
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
    return buildDetailResponse(product);
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
                    ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                    "Product không tồn tại: " + productId));

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
    return response;
  }
}
