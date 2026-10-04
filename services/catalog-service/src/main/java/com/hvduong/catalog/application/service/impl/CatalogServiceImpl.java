package com.hvduong.catalog.application.service.impl;

import com.hvduong.catalog.application.dto.request.ProductFilterRequest;
import com.hvduong.catalog.application.dto.request.PriceValidationRequest;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
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
import com.hvduong.catalog.repository.mybatis.CategoryMapper;
import com.hvduong.catalog.repository.mybatis.ChannelPriceMapper;
import com.hvduong.catalog.repository.mybatis.ProductImageMapper;
import com.hvduong.catalog.repository.mybatis.ProductMapper;
import com.hvduong.catalog.repository.mybatis.ProductVariantMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.StringUtils;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
@Transactional(readOnly = true)
public class CatalogServiceImpl implements CatalogService {

    private final ProductMapper productMapper;
    private final ProductVariantMapper variantMapper;
    private final CategoryMapper categoryMapper;
    private final ChannelPriceMapper channelPriceMapper;
    private final ProductImageMapper imageMapper;
    private final CatalogDtoMapper dtoMapper;

    @Override
    public PageResponse<ProductListItemResponse> getPublicProducts(ProductFilterRequest filter) {
        log.debug("[CATALOG] getPublicProducts: filter={}", filter);

        List<Product> products = productMapper.findPublicProducts(
                filter.getCategoryId(), filter.getOcopStar(),
                filter.getMinPrice(), filter.getMaxPrice(),
                filter.getSortBy().name(), filter.getChannel().getValue(),
                filter.getOffset(), filter.getPageSize()
        );
        long total = productMapper.countPublicProducts(
                filter.getCategoryId(), filter.getOcopStar(),
                filter.getMinPrice(), filter.getMaxPrice(),
                filter.getChannel().getValue()
        );

        List<ProductListItemResponse> items = products.stream()
                .map(p -> enrichListItem(p, filter.getChannel()))
                .collect(Collectors.toList());

        return PageResponse.of(items, filter.getPage(), filter.getPageSize(), total);
    }

    @Override
    @Cacheable(value = "catalog:product", key = "#idOrSlug + '_' + #channel.value", unless = "#result == null")
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

        // MVP: PostgreSQL ILIKE — Phase 3: Elasticsearch
        List<Product> products = productMapper.findPublicProducts(
                null, null, null, null, "NEWEST", "D2C_WEB",
                (page - 1) * pageSize, pageSize
        );
        String normalized = query.toLowerCase().trim();
        List<Product> filtered = products.stream()
                .filter(p -> p.getName().toLowerCase().contains(normalized)
                        || (p.getDescription() != null && p.getDescription().toLowerCase().contains(normalized)))
                .collect(Collectors.toList());

        List<ProductListItemResponse> items = filtered.stream()
                .map(p -> enrichListItem(p, SalesChannel.D2C_WEB))
                .collect(Collectors.toList());

        return PageResponse.of(items, page, pageSize, filtered.size());
    }

    @Override
    @Cacheable(value = "catalog:categories", key = "'all'")
    public List<CategoryResponse> getAllCategories() {
        log.debug("[CATALOG] getAllCategories");

        List<Category> all = categoryMapper.findAllActive();
        Map<UUID, List<Category>> byParent = all.stream()
                .filter(c -> c.getParentId() != null)
                .collect(Collectors.groupingBy(Category::getParentId));
        List<Category> roots = all.stream()
                .filter(c -> c.getParentId() == null)
                .collect(Collectors.toList());

        return roots.stream()
                .map(root -> buildCategoryTree(root, byParent))
                .collect(Collectors.toList());
    }

    @Override
    public ProductVariantResponse getVariantBySkuCode(String skuCode) {
        log.debug("[CATALOG] getVariantBySkuCode: skuCode={}", skuCode);

        ProductVariant variant = variantMapper.findBySkuCode(skuCode)
                .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_VARIANT_NOT_FOUND,
                        "SKU không tồn tại: " + skuCode));

        ProductVariantResponse response = dtoMapper.toVariantResponse(variant);
        channelPriceMapper.findActiveByVariantAndChannel(variant.getId(), SalesChannel.D2C_WEB.getValue())
                .ifPresent(price -> response.setPrice(dtoMapper.toMoneyResponse(price)));
        return response;
    }

    @Override
    public PageResponse<com.hvduong.catalog.application.dto.response.AdminProductListItemResponse> getAdminProducts(int page, int pageSize) {
        log.debug("[CATALOG] getAdminProducts: page={}, pageSize={}", page, pageSize);
        
        int offset = (page - 1) * pageSize;
        List<Product> products = productMapper.findAll(offset, pageSize);
        long total = productMapper.countAll();
        
        List<com.hvduong.catalog.application.dto.response.AdminProductListItemResponse> items = products.stream()
                .map(dtoMapper::toAdminListItemResponse)
                .collect(Collectors.toList());
        
        return PageResponse.of(items, page, pageSize, total);
    }

    @Override
    public com.hvduong.catalog.application.dto.response.AdminProductDetailResponse getAdminProductDetail(String productId) {
        log.debug("[CATALOG] getAdminProductDetail: productId={}", productId);
        
        UUID id;
        try {
            id = UUID.fromString(productId);
        } catch (IllegalArgumentException e) {
            throw new CatalogException(ErrorCode.CATALOG_PRODUCT_NOT_FOUND, "ID không hợp lệ: " + productId);
        }
        
        Product product = productMapper.findById(id)
                .orElseThrow(() -> new CatalogException(ErrorCode.CATALOG_PRODUCT_NOT_FOUND, "Không tìm thấy sản phẩm: " + productId));
                
        com.hvduong.catalog.application.dto.response.AdminProductDetailResponse response = dtoMapper.toAdminDetailResponse(product);
        
        // Cập nhật foodInformation
        com.hvduong.catalog.application.dto.response.AdminProductDetailResponse.AdminFoodInformationResponse foodInfo = 
            new com.hvduong.catalog.application.dto.response.AdminProductDetailResponse.AdminFoodInformationResponse(
                product.getIngredients(),
                "Thông tin cảnh báo dị ứng đang được cập nhật",
                product.getStorageGuide(),
                "Ngày sản xuất ghi trên bao bì",
                "Hạn sử dụng ghi trên bao bì"
            );
        response.setFoodInformation(foodInfo);
        
        // Cập nhật SKUs
        List<ProductVariant> variants = variantMapper.findActiveByProductId(id);
        List<com.hvduong.catalog.application.dto.response.AdminProductDetailResponse.AdminProductSkuResponse> skus = variants.stream()
            .map(dtoMapper::toAdminSkuResponse)
            .collect(Collectors.toList());
            
        response.setSkus(skus);
        
        return response;
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
            if (variantOpt.isEmpty() || variantOpt.get().getListingStatus() != ListingStatus.ACTIVE) {
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
                    || (StringUtils.hasText(item.getCurrencyCode()) && !auth.getCurrencyCode().equals(item.getCurrencyCode()))) {
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

    private ProductListItemResponse enrichListItem(Product product, SalesChannel channel) {
        ProductListItemResponse item = dtoMapper.toListItemResponse(product);
        List<ProductVariant> variants = variantMapper.findActiveByProductId(product.getId());
        if (!variants.isEmpty()) {
            channelPriceMapper.findActiveByVariantAndChannel(variants.get(0).getId(), channel.getValue())
                    .ifPresent(price -> item.setBasePrice(MoneyResponse.ofVnd(price.getAmountUnits())));
        }
        imageMapper.findApprovedByProductId(product.getId()).stream()
                .filter(img -> "COVER".equals(img.getRole())).findFirst()
                .ifPresent(img -> item.setThumbnailUrl(buildPresignedUrl(img.getObjectKey())));
        return item;
    }

    private List<ProductVariantResponse> buildVariantResponses(List<ProductVariant> variants, SalesChannel channel) {
        return variants.stream().map(v -> {
            ProductVariantResponse resp = dtoMapper.toVariantResponse(v);
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

    private CategoryResponse buildCategoryTree(Category category, Map<UUID, List<Category>> byParent) {
        CategoryResponse response = dtoMapper.toCategoryResponse(category);
        List<Category> children = byParent.getOrDefault(category.getId(), List.of());
        if (!children.isEmpty()) {
            response.setChildren(children.stream()
                    .map(child -> buildCategoryTree(child, byParent))
                    .collect(Collectors.toList()));
        }
        return response;
    }
}
