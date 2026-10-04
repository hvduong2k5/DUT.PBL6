package com.hvduong.catalog.grpc;

import com.hvduong.catalog.application.dto.request.CreateProductRequest;
import com.hvduong.catalog.application.dto.request.CreateVariantRequest;
import com.hvduong.catalog.application.dto.request.PriceValidationRequest;
import com.hvduong.catalog.application.dto.request.ProductFilterRequest;
import com.hvduong.catalog.application.dto.request.UpdatePriceRequest;
import com.hvduong.catalog.application.dto.request.UpdateProductRequest;
import com.hvduong.catalog.application.dto.response.CategoryResponse;
import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.dto.response.ProductListItemResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
import com.hvduong.catalog.application.dto.response.AdminProductListItemResponse;
import com.hvduong.catalog.application.dto.response.AdminProductDetailResponse;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.common.enums.SortBy;
import com.hvduong.catalog.common.response.PageResponse;
import com.hvduong.catalog.grpc.v1.*;
import com.omamx.proto.common.v1.*;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.UUID;
import java.util.stream.Collectors;

/**
 * Mapper giữa protobuf generated classes và Application DTOs.
 * <p>
 * Không dùng MapStruct vì mapping logic phức tạp và type hệ thống proto rất khác.
 * Plain @Component với manual mapping để dễ debug.
 */
@Component
@RequiredArgsConstructor
public class CatalogGrpcMapper {

    // ── Proto → Application DTO ───────────────────────────────────────────────

    public ProductFilterRequest toFilterRequest(ListProductsRequest proto) {
        ProductFilterRequest req = new ProductFilterRequest();
        if (proto.hasPagination()) {
            req.setPage(Math.max(1, proto.getPagination().getPage()));
            req.setPageSize(proto.getPagination().getPageSize() > 0
                    ? Math.min(100, proto.getPagination().getPageSize()) : 20);
        }
        if (proto.hasFilter()) {
            ProductFilter f = proto.getFilter();
            if (f.hasCategoryId()) req.setCategoryId(UUID.fromString(f.getCategoryId()));
            if (f.hasOcopStar())   req.setOcopStar(f.getOcopStar());
            if (f.hasMinPrice())   req.setMinPrice(f.getMinPrice());
            if (f.hasMaxPrice())   req.setMaxPrice(f.getMaxPrice());
            req.setSortBy(toSortBy(f.getSortBy()));
            req.setChannel(toSalesChannel(f.getChannel()));
        }
        return req;
    }

    public PriceValidationRequest toValidationRequest(ValidatePriceAndSkuRequest proto) {
        List<PriceValidationRequest.Item> items = proto.getItemsList().stream()
                .map(i -> PriceValidationRequest.Item.builder()
                        .skuCode(i.getSkuCode())
                        .quantity(i.getQuantity())
                        .clientUnitPrice(i.hasClientUnitPrice() ? i.getClientUnitPrice().getUnits() : 0L)
                        .currencyCode(i.hasClientUnitPrice() ? i.getClientUnitPrice().getCurrencyCode() : "VND")
                        .nanos(i.hasClientUnitPrice() ? i.getClientUnitPrice().getNanos() : 0)
                        .build())
                .collect(Collectors.toList());

        return PriceValidationRequest.builder()
                .idempotencyKey(proto.getIdempotencyKey())
                .channel(toSalesChannel(proto.getChannel()))
                .items(items)
                .build();
    }

    public CreateProductRequest toCreateProductRequest(CreateProductRpcRequest proto) {
        CreateProductRequest req = new CreateProductRequest();
        ProductInput input = proto.getProduct();
        req.setCategoryId(UUID.fromString(input.getCategoryId()));
        req.setName(input.getName());
        req.setSlug(input.getSlug());
        req.setDescription(emptyToNull(input.getDescription()));
        req.setIngredients(emptyToNull(input.getIngredients()));
        req.setStorageGuide(emptyToNull(input.getStorageGuide()));
        if (input.hasOcopStar()) req.setOcopStar(input.getOcopStar());
        req.setOcopCertificateNo(emptyToNull(input.getOcopCertificateNo()));
        req.setStory(emptyToNull(input.getStory()));
        return req;
    }

    public UpdateProductRequest toUpdateProductRequest(UpdateProductRpcRequest proto) {
        UpdateProductRequest req = new UpdateProductRequest();
        req.setVersion(proto.getExpectedVersion());
        ProductPatch patch = proto.getProduct();
        if (patch.hasCategoryId())       req.setCategoryId(UUID.fromString(patch.getCategoryId()));
        if (patch.hasName())             req.setName(patch.getName());
        if (patch.hasSlug())             req.setSlug(patch.getSlug());
        if (patch.hasDescription())      req.setDescription(patch.getDescription());
        if (patch.hasIngredients())      req.setIngredients(patch.getIngredients());
        if (patch.hasStorageGuide())     req.setStorageGuide(patch.getStorageGuide());
        if (patch.hasOcopStar())         req.setOcopStar(patch.getOcopStar());
        if (patch.hasOcopCertificateNo()) req.setOcopCertificateNo(patch.getOcopCertificateNo());
        if (patch.hasStory())            req.setStory(patch.getStory());
        return req;
    }

    public CreateVariantRequest toCreateVariantRequest(CreateVariantRpcRequest proto) {
        CreateVariantRequest req = new CreateVariantRequest();
        VariantInput input = proto.getVariant();
        req.setSkuCode(input.getSkuCode());
        req.setVariantName(input.getVariantName());
        req.setWeightValue(new java.math.BigDecimal(input.getWeightValue()));
        req.setWeightUnit(emptyToNull(input.getWeightUnit()) != null ? input.getWeightUnit() : "GRAM");
        req.setFlavor(emptyToNull(input.getFlavor()));
        req.setPackagingType(input.getPackagingType());
        if (input.hasShelfLifeDays()) req.setShelfLifeDays(input.getShelfLifeDays());
        req.setBasePriceUnits(input.getBasePriceUnits());
        return req;
    }

    public UpdatePriceRequest toUpdatePriceRequest(UpdateVariantPricesRpcRequest proto) {
        UpdatePriceRequest req = new UpdatePriceRequest();
        List<UpdatePriceRequest.PriceEntry> entries = proto.getPricesList().stream()
                .map(e -> {
                    UpdatePriceRequest.PriceEntry entry = new UpdatePriceRequest.PriceEntry();
                    entry.setChannel(toSalesChannel(e.getChannel()));
                    entry.setAmountUnits(e.getAmountUnits());
                    entry.setCurrencyCode(e.getCurrencyCode().isEmpty() ? "VND" : e.getCurrencyCode());
                    return entry;
                }).collect(Collectors.toList());
        req.setPrices(entries);
        return req;
    }

    // ── Application DTO → Proto ───────────────────────────────────────────────

    public com.hvduong.catalog.grpc.v1.Category toCategoryProto(CategoryResponse resp) {
        var builder = com.hvduong.catalog.grpc.v1.Category.newBuilder()
                .setCategoryId(str(resp.getCategoryId()))
                .setName(str(resp.getName()))
                .setSlug(str(resp.getSlug()))
                .setDescription(str(resp.getDescription()))
                .setImageUrl(str(resp.getImageUrl()))
                .setPosition(resp.getPosition());
        if (resp.getParentId() != null) builder.setParentId(resp.getParentId().toString());
        if (resp.getChildren() != null) {
            resp.getChildren().forEach(child -> builder.addChildren(toCategoryProto(child)));
        }
        return builder.build();
    }

    public ProductListItem toProductListItemProto(ProductListItemResponse resp) {
        var builder = ProductListItem.newBuilder()
                .setProductId(str(resp.getProductId()))
                .setName(str(resp.getName()))
                .setSlug(str(resp.getSlug()))
                .setSummary(str(resp.getSummary()))
                .setThumbnailUrl(str(resp.getThumbnailUrl()))
                .setCategoryName(str(resp.getCategoryName()))
                .setInStock(resp.isInStock());
        if (resp.getOcopStar() != null) builder.setOcopStar(resp.getOcopStar());
        if (resp.getBasePrice() != null) builder.setBasePrice(toMoneyProto(resp.getBasePrice()));
        return builder.build();
    }

    public ProductDetail toProductDetailProto(ProductDetailResponse resp) {
        var builder = ProductDetail.newBuilder()
                .setProductId(str(resp.getProductId()))
                .setCategoryId(str(resp.getCategoryId()))
                .setCategoryName(str(resp.getCategoryName()))
                .setName(str(resp.getName()))
                .setSlug(str(resp.getSlug()))
                .setDescription(str(resp.getDescription()))
                .setIngredients(str(resp.getIngredients()))
                .setStorageGuide(str(resp.getStorageGuide()))
                .setOcopCertificateNo(str(resp.getOcopCertificateNo()))
                .setStory(str(resp.getStory()))
                .setApprovalStatus(str(resp.getApprovalStatus()))
                .setListingStatus(str(resp.getListingStatus()))
                .setVersion(resp.getVersion());
        if (resp.getOcopStar() != null) builder.setOcopStar(resp.getOcopStar());
        if (resp.getImageUrls() != null) builder.addAllImageUrls(resp.getImageUrls());
        if (resp.getVariants() != null) {
            resp.getVariants().forEach(v -> builder.addVariants(toVariantProto(v)));
        }
        if (resp.getCreatedAt() != null)
            builder.setCreatedAtEpochMillis(resp.getCreatedAt().toEpochMilli());
        if (resp.getUpdatedAt() != null)
            builder.setUpdatedAtEpochMillis(resp.getUpdatedAt().toEpochMilli());
        return builder.build();
    }

    public com.hvduong.catalog.grpc.v1.ProductVariant toVariantProto(ProductVariantResponse resp) {
        var builder = com.hvduong.catalog.grpc.v1.ProductVariant.newBuilder()
                .setVariantId(str(resp.getVariantId()))
                .setSkuCode(str(resp.getSkuCode()))
                .setName(str(resp.getName()))
                .setWeightValue(resp.getWeightValue() != null ? resp.getWeightValue().toPlainString() : "")
                .setWeightUnit(str(resp.getWeightUnit()))
                .setFlavor(str(resp.getFlavor()))
                .setPackagingType(str(resp.getPackagingType()))
                .setListingStatus(str(resp.getListingStatus()));
        if (resp.getShelfLifeDays() != null) builder.setShelfLifeDays(resp.getShelfLifeDays());
        if (resp.getPrice() != null) builder.setPrice(toMoneyProto(resp.getPrice()));
        if (resp.getMinimumQuantity() != null) builder.setMinimumQuantity(resp.getMinimumQuantity());
        return builder.build();
    }

    public ListProductsResponse toListProductsResponse(PageResponse<ProductListItemResponse> page) {
        var pageInfo = PaginationResponse.newBuilder()
                .setCurrentPage(page.getPage()).setPageSize(page.getPageSize())
                .setTotalItems(page.getTotal()).setTotalPages(page.getTotalPages()).build();
        return ListProductsResponse.newBuilder()
                .addAllProducts(page.getItems().stream().map(item -> toProductListItemProto(item)).collect(Collectors.toList()))
                .setPageInfo(pageInfo).build();
    }

    public ValidatePriceAndSkuResponse toValidationResponse(PriceValidationResponse resp) {
        var builder = ValidatePriceAndSkuResponse.newBuilder()
                .setValid(resp.isValid())
                .setCanonicalSubtotal(Money.newBuilder()
                        .setCurrencyCode(resp.getCurrencyCode())
                        .setUnits(resp.getCanonicalSubtotalUnits()).setNanos(0).build());
        if (resp.getDiscrepancies() != null) {
            resp.getDiscrepancies().forEach(d -> builder.addDiscrepancies(
                    PriceDiscrepancy.newBuilder()
                            .setSkuCode(str(d.getSkuCode()))
                            .setClientUnitPrice(Money.newBuilder().setCurrencyCode(d.getCurrencyCode())
                                    .setUnits(d.getClientUnitPrice()).setNanos(d.getNanos()).build())
                            .setAuthoritativeUnitPrice(Money.newBuilder().setCurrencyCode(d.getCurrencyCode())
                                    .setUnits(d.getAuthoritativeUnitPrice()).setNanos(0).build())
                            .setReason(str(d.getReason())).build()));
        }
        return builder.build();
    }

    public AdminProductListItem toAdminProductListItemProto(AdminProductListItemResponse resp) {
        var builder = AdminProductListItem.newBuilder()
                .setProductId(str(resp.getProductId()))
                .setName(str(resp.getName()))
                .setCategoryId(str(resp.getCategoryId()))
                .setCategoryName(str(resp.getCategoryName()))
                .setSaleStatus(str(resp.getSaleStatus()))
                .setSaleStatusLabel(str(resp.getSaleStatusLabel()));
        if (resp.getSkuCount() != null) builder.setSkuCount(resp.getSkuCount());
        if (resp.getOnSaleSkuCount() != null) builder.setOnSaleSkuCount(resp.getOnSaleSkuCount());
        if (resp.getBasePriceFromVnd() != null) builder.setBasePriceFromVnd(resp.getBasePriceFromVnd());
        if (resp.getFoodInformationComplete() != null) builder.setFoodInformationComplete(resp.getFoodInformationComplete());
        builder.setUpdatedAt(str(resp.getUpdatedAt()));
        if (resp.getRevision() != null) builder.setRevision(resp.getRevision());
        return builder.build();
    }
    
    public ListAdminProductsResponse toListAdminProductsResponse(PageResponse<AdminProductListItemResponse> page) {
        return ListAdminProductsResponse.newBuilder()
                .addAllItems(page.getItems().stream().map(item -> toAdminProductListItemProto(item)).collect(Collectors.toList()))
                .setTotal((int) page.getTotal()).build();
    }

    public AdminProductDetail toAdminProductDetailProto(AdminProductDetailResponse resp) {
        var builder = AdminProductDetail.newBuilder()
                .setProductId(str(resp.getProductId()))
                .setName(str(resp.getName()))
                .setCategoryId(str(resp.getCategoryId()))
                .setCategoryName(str(resp.getCategoryName()))
                .setSaleStatus(str(resp.getSaleStatus()))
                .setSaleStatusLabel(str(resp.getSaleStatusLabel()))
                .setShortDescription(str(resp.getShortDescription()))
                .setLongDescription(str(resp.getLongDescription()))
                .setCoverImageUrl(str(resp.getCoverImageUrl()))
                .setCoverImageAlt(str(resp.getCoverImageAlt()));
                
        if (resp.getSkuCount() != null) builder.setSkuCount(resp.getSkuCount());
        if (resp.getOnSaleSkuCount() != null) builder.setOnSaleSkuCount(resp.getOnSaleSkuCount());
        if (resp.getBasePriceFromVnd() != null) builder.setBasePriceFromVnd(resp.getBasePriceFromVnd());
        if (resp.getFoodInformationComplete() != null) builder.setFoodInformationComplete(resp.getFoodInformationComplete());
        builder.setUpdatedAt(str(resp.getUpdatedAt()));
        if (resp.getRevision() != null) builder.setRevision(resp.getRevision());
        
        if (resp.getFoodInformation() != null) {
            builder.setFoodInformation(AdminFoodInformation.newBuilder()
                .setIngredients(str(resp.getFoodInformation().getIngredients()))
                .setAllergenStatement(str(resp.getFoodInformation().getAllergenStatement()))
                .setStorageInstructions(str(resp.getFoodInformation().getStorageInstructions()))
                .setManufacturingDatePolicy(str(resp.getFoodInformation().getManufacturingDatePolicy()))
                .setShelfLifeDescription(str(resp.getFoodInformation().getShelfLifeDescription()))
                .build());
        }
        
        if (resp.getSkus() != null) {
            resp.getSkus().forEach(s -> {
                var skuBuilder = AdminProductSku.newBuilder()
                    .setSkuId(str(s.getSkuId()))
                    .setSkuCode(str(s.getSkuCode()))
                    .setLabel(str(s.getLabel()))
                    .setFlavor(str(s.getFlavor()))
                    .setPackageType(str(s.getPackageType()))
                    .setCurrency(str(s.getCurrency()))
                    .setSaleStatus(str(s.getSaleStatus()))
                    .setSaleStatusLabel(str(s.getSaleStatusLabel()))
                    .setUpdatedAt(str(s.getUpdatedAt()));
                if (s.getWeightGrams() != null) skuBuilder.setWeightGrams(s.getWeightGrams());
                if (s.getBasePriceVnd() != null) skuBuilder.setBasePriceVnd(s.getBasePriceVnd());
                if (s.getRevision() != null) skuBuilder.setRevision(s.getRevision());
                builder.addSkus(skuBuilder.build());
            });
        }
        
        return builder.build();
    }

    // ── Enum mappers ──────────────────────────────────────────────────────────

    public SalesChannel toSalesChannel(com.hvduong.catalog.grpc.v1.SalesChannel proto) {
        return switch (proto) {
            case MOBILE_APP    -> SalesChannel.MOBILE_APP;
            case B2B_WHOLESALE -> SalesChannel.B2B_WHOLESALE;
            case POS_QUAY      -> SalesChannel.POS_QUAY;
            case SHOPEE        -> SalesChannel.SHOPEE;
            case TIKTOK        -> SalesChannel.TIKTOK;
            default            -> SalesChannel.D2C_WEB;
        };
    }

    public SortBy toSortBy(com.hvduong.catalog.grpc.v1.SortBy proto) {
        return switch (proto) {
            case PRICE_ASC   -> SortBy.PRICE_ASC;
            case PRICE_DESC  -> SortBy.PRICE_DESC;
            case BEST_SELLING -> SortBy.BEST_SELLING;
            default          -> SortBy.NEWEST;
        };
    }

    // ── Util ─────────────────────────────────────────────────────────────────

    private Money toMoneyProto(MoneyResponse money) {
        return Money.newBuilder()
                .setCurrencyCode(str(money.getCurrencyCode()))
                .setUnits(money.getUnits())
                .setNanos(money.getNanos())
                .build();
    }

    private String str(Object o) {
        return o == null ? "" : o.toString();
    }

    private String emptyToNull(String s) {
        return (s == null || s.isBlank()) ? null : s;
    }
}
