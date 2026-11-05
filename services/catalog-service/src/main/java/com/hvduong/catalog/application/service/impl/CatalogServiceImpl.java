package com.hvduong.catalog.application.service.impl;

import java.math.BigDecimal;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import java.util.stream.Collectors;

import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.StringUtils;

import com.hvduong.catalog.application.dto.request.PriceValidationRequest;
import com.hvduong.catalog.application.dto.request.ProductFilterRequest;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse.AdminProductSkuResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListItemResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListResponse;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.mapper.CatalogDtoMapper;
import com.hvduong.catalog.application.service.CatalogService;
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

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
@RequiredArgsConstructor
@Transactional(readOnly = true)
public class CatalogServiceImpl implements CatalogService {

    private final ProductMapper productMapper;
    private final ProductVariantMapper variantMapper;
    private final ChannelPriceMapper channelPriceMapper;
    private final ProductImageMapper imageMapper;
    private final CategoryMapper categoryMapper;
    private final StockProjectionService stockProjectionService;
    private final CatalogDtoMapper dtoMapper;

    @Override
    public PageResponse<ProductListItemResponse> getPublicProducts(ProductFilterRequest filter) {
        log.debug("[CATALOG] getPublicProducts: filter={}", filter);

        List<Product> products = productMapper.findPublicProducts(
                "", // non-search: no query filter
                filter.getCategoryId(),
                filter.getOcopStar(),
                filter.getMinPrice(),
                filter.getMaxPrice(),
                filter.getSortBy().name(),
                filter.getChannel().getValue(),
                filter.getOffset(),
                filter.getPageSize());

        long total = productMapper.countPublicProducts(
                "", // non-search: no query filter
                filter.getCategoryId(),
                filter.getOcopStar(),
                filter.getMinPrice(),
                filter.getMaxPrice(),
                filter.getChannel().getValue());

        List<ProductListItemResponse> items = enrichListItems(products, filter.getChannel());

        return PageResponse.of(items, filter.getPage(), filter.getPageSize(), total);
    }

    @Override
    public ProductDetailResponse getPublicProductDetail(String idOrSlug, SalesChannel channel) {
        log.debug("[CATALOG] getPublicProductDetail: idOrSlug={}, channel={}", idOrSlug, channel);

        Product product = tryFindByIdOrSlug(idOrSlug, true)
                .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Product không tồn tại hoặc không công khai: " + idOrSlug));

        List<ProductVariant> variants = variantMapper.findActiveByProductId(product.getId());
        List<ProductImage> images = imageMapper.findApprovedByProductId(product.getId());

        ProductDetailResponse response = dtoMapper.toDetailResponse(product);
        response.setVariants(buildVariantResponses(variants, channel));
        response.setImageUrls(buildImageUrls(images));
        return response;
    }

    @Override
    public PageResponse<ProductListItemResponse> searchProducts(String query, int page, int pageSize) {
        log.debug("[CATALOG] searchProducts: query='{}', page={}, size={}", query, page, pageSize);

        if (!StringUtils.hasText(query)) {
            return PageResponse.of(List.of(), page, pageSize, 0);
        }

        // MVP: PostgreSQL ILIKE pushed down to SQL; no in-memory filtering.
        List<Product> products = productMapper.findPublicProducts(
                query, null, null, null, null, "NEWEST", "D2C_WEB",
                (page - 1) * pageSize, pageSize);
        long total = productMapper.countPublicProducts(
                query, null, null, null, null, "D2C_WEB");

        List<ProductListItemResponse> items = enrichListItems(products, SalesChannel.D2C_WEB);

        return PageResponse.of(items, page, pageSize, total);
    }

    @Override
    @Cacheable(value = "catalog:categories", key = "'all'")
    public List<CategoryResponse> getAllCategories() {
        log.debug("[CATALOG] getAllCategories");

        // Trả danh sách phẳng toàn bộ category active, giữ parent_id;
        // controller映射 flat, không trả cây children.
        List<Category> all = categoryMapper.findAllActive();
        return all.stream()
                .map(dtoMapper::toCategoryResponse)
                .collect(Collectors.toList());
    }

    @Override
    public ProductVariantResponse getVariantBySkuCode(String skuCode) {
        log.debug("[CATALOG] getVariantBySkuCode: skuCode={}", skuCode);

        ProductVariant variant = variantMapper.findBySkuCode(skuCode)
                .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_VARIANT_NOT_FOUND,
                        "SKU không tồn tại: " + skuCode));

        if (variant.getListingStatus() != ListingStatus.ACTIVE
                || productMapper.findPublicById(variant.getProductId()).isEmpty()) {
            throw new CatalogException(ErrorCode.CATALOG_PRODUCT_NOT_PUBLIC);
        }

        ProductVariantResponse response = dtoMapper.toVariantResponse(variant);
        response.setStockAvailable(stockProjectionService
                .getAvailableQuantities(List.of(variant.getSkuCode()))
                .get(variant.getSkuCode()));
        channelPriceMapper.findActiveByVariantAndChannel(variant.getId(), SalesChannel.D2C_WEB.getValue())
                .ifPresent(price -> response.setPrice(dtoMapper.toMoneyResponse(price)));
        return response;
    }

    @Override
    public AdminProductListResponse getAdminProducts(int page, int pageSize, String q, String status) {
        log.debug("[CATALOG] getAdminProducts: page={}, pageSize={}, q={}, status={}", page, pageSize, q, status);

        int offset = (page - 1) * pageSize;
        List<Product> products = productMapper.findAdminProducts(q, status, offset, pageSize);
        long total = productMapper.countAdminProducts(q, status);

        List<UUID> productIds = products.stream().map(Product::getId).collect(Collectors.toList());

        // Batch variants cho toàn bộ trang (tránh N+1)
        Map<UUID, List<ProductVariant>> variantsByProduct = new HashMap<>();
        if (!productIds.isEmpty()) {
            List<ProductVariant> allVariants = variantMapper.findByProductIds(productIds);
            for (ProductVariant v : allVariants) {
                variantsByProduct.computeIfAbsent(v.getProductId(), k -> new ArrayList<>()).add(v);
            }
        }

        // Batch giá hiệu lực theo channel admin (mặc định D2C_WEB)
        List<UUID> variantIds = variantsByProduct.values().stream()
                .flatMap(List::stream)
                .map(ProductVariant::getId)
                .collect(Collectors.toList());
        Map<UUID, ChannelPrice> currentPriceByVariant = new HashMap<>();
        if (!variantIds.isEmpty()) {
            String channel = SalesChannel.D2C_WEB.getValue();
            for (ChannelPrice price : channelPriceMapper.findActiveCurrentByVariantIds(variantIds, channel)) {
                currentPriceByVariant.putIfAbsent(price.getVariantId(), price);
            }
        }

        // statusCounts: tính trên tập đã lọc bởi q (không lọc theo status đang chọn)
        List<AdminStatusCountRow> statusRows = productMapper.countAdminProductsByStatus(q);
        Map<String, Long> statusCountMap = new HashMap<>();
        for (AdminStatusCountRow row : statusRows) {
            statusCountMap.merge(row.getStatus(), row.getCount(), Long::sum);
        }

        // categories: các category có Product trong tập q
        List<AdminCategoryRow> categoryRows = productMapper.findAdminCategories(q);

        String calculatedAt = Instant.now().toString();

        List<AdminProductListItemResponse> items = products.stream()
                .map(product -> toAdminListItemResponse(
                        product,
                        variantsByProduct.getOrDefault(product.getId(), List.of()),
                        currentPriceByVariant))
                .collect(Collectors.toList());

        List<AdminProductListResponse.StatusCount> statusCounts = List.of(
                ListingStatus.ACTIVE, ListingStatus.SUSPENDED, ListingStatus.ARCHIVED).stream()
                .map(st -> new AdminProductListResponse.StatusCount(
                        st.getValue(),
                        displayStatusLabel(st),
                        statusCountMap.getOrDefault(st.getValue(), 0L)))
                .collect(Collectors.toList());

        List<AdminProductListResponse.CategoryOption> categories = categoryRows.stream()
                .map(row -> new AdminProductListResponse.CategoryOption(
                        row.getCategoryId().toString(), row.getCategoryName()))
                .collect(Collectors.toList());

        return AdminProductListResponse.builder()
                .items(items)
                .statusCounts(statusCounts)
                .categories(categories)
                .calculatedAt(calculatedAt)
                .build();
    }

    @Override
    public AdminProductDetailResponse getAdminProductDetail(String productId) {
        log.debug("[CATALOG] getAdminProductDetail: productId={}", productId);

        UUID id;
        try {
            id = UUID.fromString(productId);
        } catch (IllegalArgumentException e) {
            throw new CatalogException(ErrorCode.CATALOG_PRODUCT_NOT_FOUND, "ID không hợp lệ: " + productId);
        }

        Product product = productMapper.findById(id)
                .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                        "Không tìm thấy sản phẩm: " + productId));

        List<ProductImage> images = imageMapper.findApprovedByProductId(id);
        product.setImages(images);

        AdminProductDetailResponse response = dtoMapper.toAdminDetailResponse(product);
        response.setCategoryName(product.getCategoryName());
        response.setSaleStatus(statusLabel(product.getListingStatus()));
        response.setSaleStatusLabel(displayStatusLabel(product.getListingStatus()));
        response.setLongDescription(product.getDescription());
        response.setCoverImageUrl(coverImageUrl(product));
        response.setCoverImageAlt(coverImageAlt(product));

        // Trả toàn bộ variant (ACTIVE + SUSPENDED + ARCHIVED), kèm giá hiện hành theo
        // channel admin
        List<ProductVariant> variants = variantMapper.findAllByProductId(id);
        List<UUID> variantIds = variants.stream().map(ProductVariant::getId).collect(Collectors.toList());
        Map<UUID, ChannelPrice> currentPriceByVariant = new HashMap<>();
        if (!variantIds.isEmpty()) {
            String channel = SalesChannel.D2C_WEB.getValue();
            for (ChannelPrice price : channelPriceMapper.findActiveCurrentByVariantIds(variantIds, channel)) {
                currentPriceByVariant.putIfAbsent(price.getVariantId(), price);
            }
        }

        List<AdminProductSkuResponse> skus = variants.stream()
                .map(variant -> toAdminSkuResponse(variant, currentPriceByVariant.get(variant.getId())))
                .collect(Collectors.toList());
        response.setSkus(skus);

        response.setSkuCount(variants.size());
        response.setOnSaleSkuCount(variants.stream()
                .filter(v -> v.getListingStatus() == ListingStatus.ACTIVE)
                .mapToInt(v -> 1)
                .sum());
        response.setBasePriceFromVnd(skuBasePriceFromVnd(variants, currentPriceByVariant));
        response.setFoodInformationComplete(foodInformationComplete(product));
        response.setUpdatedAt(formatInstant(product.getUpdatedAt()));
        response.setRevision(product.getVersion());

        return response;
    }

    private AdminProductListItemResponse toAdminListItemResponse(Product product,
            List<ProductVariant> variants,
            Map<UUID, ChannelPrice> currentPriceByVariant) {
        AdminProductListItemResponse item = new AdminProductListItemResponse();
        item.setProductId(product.getId().toString());
        item.setName(product.getName());
        item.setCategoryId(product.getCategoryId().toString());
        item.setCategoryName(product.getCategoryName());
        item.setSaleStatus(statusLabel(product.getListingStatus()));
        item.setSaleStatusLabel(displayStatusLabel(product.getListingStatus()));
        item.setSkuCount(variants.size());
        item.setOnSaleSkuCount(variants.stream()
                .filter(v -> v.getListingStatus() == ListingStatus.ACTIVE)
                .mapToInt(v -> 1)
                .sum());
        item.setBasePriceFromVnd(skuBasePriceFromVnd(variants, currentPriceByVariant));
        item.setFoodInformationComplete(foodInformationComplete(product));
        item.setUpdatedAt(formatInstant(product.getUpdatedAt()));
        item.setRevision(product.getVersion());
        return item;
    }

    private AdminProductSkuResponse toAdminSkuResponse(ProductVariant variant, ChannelPrice price) {
        AdminProductSkuResponse sku = new AdminProductSkuResponse();
        sku.setSkuId(variant.getId().toString());
        sku.setSkuCode(variant.getSkuCode());
        sku.setLabel(variant.getVariantName());
        sku.setWeightGrams(weightGrams(variant.getWeightValue()));
        sku.setFlavor(variant.getFlavor());
        sku.setPackageType(packagingLabel(variant.getPackagingType()));
        sku.setBasePriceVnd(price != null ? price.getAmountUnits() : null);
        sku.setCurrency(price != null ? price.getCurrencyCode() : "VND");
        sku.setSaleStatus(statusLabel(variant.getListingStatus()));
        sku.setSaleStatusLabel(displayStatusLabel(variant.getListingStatus()));
        sku.setUpdatedAt(formatInstant(variant.getUpdatedAt()));
        sku.setRevision(variant.getVersion());
        return sku;
    }

    private String statusLabel(ListingStatus listingStatus) {
        if (listingStatus == null)
            return "DISCONTINUED";
        return switch (listingStatus) {
            case ACTIVE -> "ON_SALE";
            case SUSPENDED -> "PAUSED";
            case ARCHIVED -> "DISCONTINUED";
        };
    }

    private String displayStatusLabel(ListingStatus listingStatus) {
        if (listingStatus == null)
            return "Ngừng kinh doanh";
        return switch (listingStatus) {
            case ACTIVE -> "Đang bán";
            case SUSPENDED -> "Tạm ngừng bán";
            case ARCHIVED -> "Ngừng kinh doanh";
        };
    }

    private Long skuBasePriceFromVnd(List<ProductVariant> variants, Map<UUID, ChannelPrice> currentPriceByVariant) {
        Long min = null;
        for (ProductVariant variant : variants) {
            ChannelPrice price = currentPriceByVariant.get(variant.getId());
            if (price != null && price.getAmountUnits() > 0) {
                Long units = price.getAmountUnits();
                min = (min == null || units < min) ? units : min;
            }
        }
        return min;
    }

    private boolean foodInformationComplete(Product product) {
        return StringUtils.hasText(product.getIngredients())
                && StringUtils.hasText(product.getStorageGuide())
                && StringUtils.hasText(product.getName());
    }

    private Integer weightGrams(BigDecimal weightValue) {
        if (weightValue == null)
            return null;
        return weightValue.intValue();
    }

    private String packagingLabel(String packagingType) {
        if (packagingType == null)
            return null;
        return packagingType.replace('_', ' ').toLowerCase();
    }

    private String formatInstant(Instant instant) {
        return instant == null ? null : instant.toString();
    }

    private String coverImageUrl(Product product) {
        if (product.getImages() == null || product.getImages().isEmpty())
            return null;
        return product.getImages().stream()
                .filter(img -> "COVER".equals(img.getRole()) && "APPROVED".equals(img.getMediaStatus()))
                .findFirst()
                .map(img -> buildPresignedUrl(img.getObjectKey()))
                .orElse(null);
    }

    private String coverImageAlt(Product product) {
        if (product.getImages() == null || product.getImages().isEmpty())
            return null;
        return product.getImages().stream()
                .filter(img -> "COVER".equals(img.getRole()) && "APPROVED".equals(img.getMediaStatus()))
                .findFirst()
                .map(ProductImage::getAltText)
                .filter(StringUtils::hasText)
                .orElse(product.getName());
    }

    @Override
    public PriceValidationResponse validatePrices(PriceValidationRequest request) {
        SalesChannel channel = request.getChannel() == null ? SalesChannel.D2C_WEB : request.getChannel();
        List<PriceValidationResponse.Discrepancy> discrepancies = new ArrayList<>();
        long subtotal = 0L;

        for (PriceValidationRequest.Item item : request.getItems()) {
            if (!StringUtils.hasText(item.getSkuCode()) || item.getQuantity() <= 0) {
                discrepancies.add(PriceValidationResponse.Discrepancy.builder()
                        .skuCode(item.getSkuCode()).clientUnitPrice(item.getClientUnitPrice())
                        .authoritativeUnitPrice(0L).currencyCode("VND").nanos(0)
                        .reason("SKU và số lượng phải hợp lệ").build());
                continue;
            }
            Optional<ProductVariant> variantOpt = variantMapper.findBySkuCode(item.getSkuCode());
            if (variantOpt.isEmpty() || variantOpt.get().getListingStatus() != ListingStatus.ACTIVE
                    || productMapper.findPublicById(variantOpt.get().getProductId()).isEmpty()) {
                discrepancies.add(PriceValidationResponse.Discrepancy.builder()
                        .skuCode(item.getSkuCode()).clientUnitPrice(item.getClientUnitPrice())
                        .authoritativeUnitPrice(0L).currencyCode("VND").nanos(0)
                        .reason("SKU không tồn tại hoặc không còn được bán").build());
                continue;
            }
            Optional<ChannelPrice> priceOpt = channelPriceMapper.findActiveByVariantAndChannel(
                    variantOpt.get().getId(), channel.getValue());
            if (priceOpt.isEmpty()) {
                discrepancies.add(PriceValidationResponse.Discrepancy.builder()
                        .skuCode(item.getSkuCode()).clientUnitPrice(item.getClientUnitPrice())
                        .authoritativeUnitPrice(0L).currencyCode("VND").nanos(0)
                        .reason("Không tìm thấy giá hiệu lực cho kênh " + channel.getValue()).build());
                continue;
            }
            ChannelPrice auth = priceOpt.get();
            subtotal = Math.addExact(subtotal, Math.multiplyExact(auth.getAmountUnits(), item.getQuantity()));
            if (item.getClientUnitPrice() != auth.getAmountUnits()
                    || item.getNanos() != auth.getAmountNanos()
                    || (StringUtils.hasText(item.getCurrencyCode())
                            && !auth.getCurrencyCode().equals(item.getCurrencyCode()))) {
                discrepancies.add(PriceValidationResponse.Discrepancy.builder()
                        .skuCode(item.getSkuCode()).clientUnitPrice(item.getClientUnitPrice())
                        .authoritativeUnitPrice(auth.getAmountUnits())
                        .currencyCode(auth.getCurrencyCode()).nanos(auth.getAmountNanos())
                        .reason("Giá hoặc tiền tệ do client gửi không khớp giá niêm yết").build());
            }
        }

        return PriceValidationResponse.builder()
                .valid(discrepancies.isEmpty())
                .canonicalSubtotalUnits(subtotal)
                .currencyCode("VND")
                .discrepancies(discrepancies)
                .build();
    }

    // ── Private helpers ───────────────────────────────────────────────────────

    private Optional<Product> tryFindByIdOrSlug(String idOrSlug, boolean publicOnly) {
        try {
            UUID id = UUID.fromString(idOrSlug);
            return publicOnly ? productMapper.findPublicById(id) : productMapper.findById(id);
        } catch (IllegalArgumentException e) {
            return productMapper.findPublicBySlug(idOrSlug);
        }
    }

    private List<ProductListItemResponse> enrichListItems(List<Product> products, SalesChannel channel) {
        Map<UUID, List<ProductVariant>> variantsByProduct = new HashMap<>();
        for (Product product : products) {
            variantsByProduct.putIfAbsent(product.getId(), variantMapper.findActiveByProductId(product.getId()));
        }
        List<UUID> variantIds = variantsByProduct.values().stream()
                .flatMap(List::stream)
                .map(ProductVariant::getId)
                .distinct()
                .collect(Collectors.toList());
        Map<UUID, ChannelPrice> currentPriceByVariant = new HashMap<>();
        if (!variantIds.isEmpty()) {
            for (ChannelPrice price : channelPriceMapper.findActiveCurrentByVariantIds(variantIds,
                    channel.getValue())) {
                currentPriceByVariant.putIfAbsent(price.getVariantId(), price);
            }
        }
        Map<String, Long> quantities = stockProjectionService.getAvailableQuantities(variantsByProduct.values().stream()
                .flatMap(List::stream)
                .map(ProductVariant::getSkuCode)
                .distinct()
                .collect(Collectors.toList()));
        return products.stream()
                .map(product -> enrichListItem(product, channel,
                        variantsByProduct.getOrDefault(product.getId(), List.of()), currentPriceByVariant, quantities))
                .collect(Collectors.toList());
    }

    private ProductListItemResponse enrichListItem(Product product, SalesChannel channel,
            List<ProductVariant> variants,
            Map<UUID, ChannelPrice> currentPriceByVariant,
            Map<String, Long> quantities) {
        ProductListItemResponse item = dtoMapper.toListItemResponse(product);
        boolean stockKnown = variants.stream()
                .allMatch(variant -> quantities.get(variant.getSkuCode()) != null);
        Long available = stockKnown ? variants.stream()
                .mapToLong(variant -> quantities.get(variant.getSkuCode()))
                .sum() : null;
        item.setStockAvailable(available);
        item.setInStock(available == null ? null : available > 0);
        Long minPrice = skuBasePriceFromVnd(variants, currentPriceByVariant);
        if (minPrice != null) {
            item.setBasePrice(MoneyResponse.ofVnd(minPrice));
        }
        imageMapper.findApprovedByProductId(product.getId()).stream()
                .filter(img -> "COVER".equals(img.getRole())).findFirst()
                .ifPresent(img -> item.setThumbnailUrl(buildPresignedUrl(img.getObjectKey())));
        return item;
    }

    private List<ProductVariantResponse> buildVariantResponses(List<ProductVariant> variants, SalesChannel channel) {
        Map<String, Long> quantities = stockProjectionService.getAvailableQuantities(variants.stream()
                .map(ProductVariant::getSkuCode)
                .collect(Collectors.toList()));
        return variants.stream().map(v -> {
            ProductVariantResponse resp = dtoMapper.toVariantResponse(v);
            resp.setStockAvailable(quantities.get(v.getSkuCode()));
            channelPriceMapper.findActiveByVariantAndChannel(v.getId(), channel.getValue())
                    .ifPresent(price -> resp.setPrice(dtoMapper.toMoneyResponse(price)));
            return resp;
        }).collect(Collectors.toList());
    }

    private List<String> buildImageUrls(List<ProductImage> images) {
        return images.stream().map(img -> buildPresignedUrl(img.getObjectKey())).collect(Collectors.toList());
    }

    /** TODO Phase 2: MinIO presigned URL generator */
    private String buildPresignedUrl(String objectKey) {
        return "https://cdn.example.com/" + objectKey;
    }
}
