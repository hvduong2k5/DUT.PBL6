package com.hvduong.catalog.grpc;

import com.hvduong.catalog.common.enums.ApprovalStatus;
import com.hvduong.catalog.common.enums.ListingStatus;
import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;
import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductVariant;
import com.hvduong.catalog.repository.mybatis.ChannelPriceMapper;
import com.hvduong.catalog.repository.mybatis.ProductMapper;
import com.hvduong.catalog.repository.mybatis.ProductVariantMapper;

import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;

import org.springframework.util.StringUtils;

import com.google.protobuf.Timestamp;
import com.omamx.proto.catalog.v1.*;
import com.omamx.proto.common.v1.ErrorDetail;
import com.omamx.proto.common.v1.Money;
import com.omamx.proto.common.v1.PaginationResponse;

import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.server.service.GrpcService;

/**
 * Service gRPC nội bộ phục vụ giao tiếp liên microservice (East-West Traffic) Điểm neo trên
 * Critical Path: Thẩm định giá bán chống gian lận F12/Inspect cho order-service.
 */
@Slf4j
@GrpcService
@RequiredArgsConstructor
public class CatalogGrpcService extends CatalogServiceGrpc.CatalogServiceImplBase {

  private final ProductVariantMapper variantMapper;
  private final ProductMapper productMapper;
  private final ChannelPriceMapper channelPriceMapper;
  private final CatalogGrpcMapper grpcMapper;

  @Override
  public void validateCheckoutPricing(
      ValidateCheckoutPricingRequest request,
      StreamObserver<ValidateCheckoutPricingResponse> observer) {
    log.debug(
        "[gRPC-Internal] validateCheckoutPricing: channel={}, itemsCount={}",
        request.getChannel(),
        request.getItemsCount());
    complete(
        observer,
        () -> {
          if (request.getChannel() == null
              || request.getChannel() == SalesChannel.SALES_CHANNEL_UNSPECIFIED) {
            throw new CatalogException(
                ErrorCode.INVALID_ARGUMENT, "Sales channel must be specified");
          }

          if (request.getItemsCount() < 1 || request.getItemsCount() > 100) {
            throw new CatalogException(
                ErrorCode.INVALID_ARGUMENT, "Items count must be between 1 and 100");
          }

          var domainChannel = grpcMapper.toSalesChannel(request.getChannel());
          if (domainChannel == null) {
            throw new CatalogException(
                ErrorCode.INVALID_ARGUMENT, "Unsupported sales channel: " + request.getChannel());
          }

          // Kiểm tra tính hợp lệ và không trùng lặp SKU
          Set<String> seenSkus = new HashSet<>();
          for (CheckoutItemInput item : request.getItemsList()) {
            if (!StringUtils.hasText(item.getSkuCode())) {
              throw new CatalogException(ErrorCode.INVALID_ARGUMENT, "sku_code is required");
            }
            if (item.getQuantity() < 1 || item.getQuantity() > 999) {
              throw new CatalogException(
                  ErrorCode.INVALID_ARGUMENT,
                  "Quantity must be between 1 and 999 for SKU: " + item.getSkuCode());
            }
            if (!seenSkus.add(item.getSkuCode())) {
              throw new CatalogException(
                  ErrorCode.INVALID_ARGUMENT, "Duplicate SKU detected: " + item.getSkuCode());
            }
          }

          List<ValidatedCheckoutItem> validatedItems = new ArrayList<>();
          List<PriceDiscrepancy> discrepancies = new ArrayList<>();
          long canonicalSubtotalUnits = 0L;
          Money zeroMoney =
              Money.newBuilder().setCurrencyCode("VND").setUnits(0).setNanos(0).build();

          for (CheckoutItemInput item : request.getItemsList()) {
            String sku = item.getSkuCode();
            Optional<ProductVariant> variantOpt = variantMapper.findBySkuCode(sku);

            if (variantOpt.isEmpty()) {
              discrepancies.add(
                  PriceDiscrepancy.newBuilder()
                      .setSkuCode(sku)
                      .setClientUnitPrice(item.getClientUnitPrice())
                      .setAuthoritativeUnitPrice(zeroMoney)
                      .setDiscrepancyReason(ValidationIssue.VALIDATION_ISSUE_SKU_NOT_FOUND)
                      .build());
              validatedItems.add(
                  ValidatedCheckoutItem.newBuilder()
                      .setSkuCode(sku)
                      .setQuantity(item.getQuantity())
                      .setAuthoritativeUnitPrice(zeroMoney)
                      .setIsSellable(false)
                      .build());
              continue;
            }

            ProductVariant variant = variantOpt.get();
            if (variant.getListingStatus() != ListingStatus.ACTIVE) {
              discrepancies.add(
                  PriceDiscrepancy.newBuilder()
                      .setSkuCode(sku)
                      .setClientUnitPrice(item.getClientUnitPrice())
                      .setAuthoritativeUnitPrice(zeroMoney)
                      .setDiscrepancyReason(ValidationIssue.VALIDATION_ISSUE_SKU_INACTIVE)
                      .build());
              validatedItems.add(
                  ValidatedCheckoutItem.newBuilder()
                      .setProductId(
                          variant.getProductId() != null ? variant.getProductId().toString() : "")
                      .setVariantId(variant.getId() != null ? variant.getId().toString() : "")
                      .setSkuCode(sku)
                      .setVariantName(
                          variant.getVariantName() != null ? variant.getVariantName() : "")
                      .setQuantity(item.getQuantity())
                      .setAuthoritativeUnitPrice(zeroMoney)
                      .setNetWeight(
                          grpcMapper.toWeightProto(
                              variant.getWeightValue(), variant.getWeightUnit()))
                      .setPackagingType(grpcMapper.toPackagingTypeProto(variant.getPackagingType()))
                      .setIsSellable(false)
                      .setVariantVersion(variant.getVersion())
                      .setPriceVersion(variant.getPriceRevision())
                      .build());
              continue;
            }

            Optional<Product> productOpt = productMapper.findById(variant.getProductId());
            if (productOpt.isEmpty()
                || productOpt.get().getApprovalStatus() != ApprovalStatus.APPROVED
                || productOpt.get().getListingStatus() != ListingStatus.ACTIVE) {
              discrepancies.add(
                  PriceDiscrepancy.newBuilder()
                      .setSkuCode(sku)
                      .setClientUnitPrice(item.getClientUnitPrice())
                      .setAuthoritativeUnitPrice(zeroMoney)
                      .setDiscrepancyReason(ValidationIssue.VALIDATION_ISSUE_PRODUCT_NOT_APPROVED)
                      .build());
              validatedItems.add(
                  ValidatedCheckoutItem.newBuilder()
                      .setProductId(
                          variant.getProductId() != null ? variant.getProductId().toString() : "")
                      .setVariantId(variant.getId() != null ? variant.getId().toString() : "")
                      .setSkuCode(sku)
                      .setProductName(productOpt.map(Product::getName).orElse(""))
                      .setVariantName(
                          variant.getVariantName() != null ? variant.getVariantName() : "")
                      .setQuantity(item.getQuantity())
                      .setAuthoritativeUnitPrice(zeroMoney)
                      .setNetWeight(
                          grpcMapper.toWeightProto(
                              variant.getWeightValue(), variant.getWeightUnit()))
                      .setPackagingType(grpcMapper.toPackagingTypeProto(variant.getPackagingType()))
                      .setIsSellable(false)
                      .setProductVersion(productOpt.map(Product::getVersion).orElse(0L))
                      .setVariantVersion(variant.getVersion())
                      .setPriceVersion(variant.getPriceRevision())
                      .build());
              continue;
            }

            Product product = productOpt.get();
            Optional<ChannelPrice> priceOpt =
                channelPriceMapper.findActiveByVariantAndChannel(
                    variant.getId(), domainChannel.getValue());
            if (priceOpt.isEmpty()) {
              discrepancies.add(
                  PriceDiscrepancy.newBuilder()
                      .setSkuCode(sku)
                      .setClientUnitPrice(item.getClientUnitPrice())
                      .setAuthoritativeUnitPrice(zeroMoney)
                      .setDiscrepancyReason(ValidationIssue.VALIDATION_ISSUE_PRICE_NOT_CONFIGURED)
                      .build());
              validatedItems.add(
                  ValidatedCheckoutItem.newBuilder()
                      .setProductId(product.getId().toString())
                      .setVariantId(variant.getId().toString())
                      .setSkuCode(sku)
                      .setProductName(product.getName() != null ? product.getName() : "")
                      .setVariantName(
                          variant.getVariantName() != null ? variant.getVariantName() : "")
                      .setQuantity(item.getQuantity())
                      .setAuthoritativeUnitPrice(zeroMoney)
                      .setNetWeight(
                          grpcMapper.toWeightProto(
                              variant.getWeightValue(), variant.getWeightUnit()))
                      .setPackagingType(grpcMapper.toPackagingTypeProto(variant.getPackagingType()))
                      .setIsSellable(false)
                      .setProductVersion(product.getVersion())
                      .setVariantVersion(variant.getVersion())
                      .setPriceVersion(variant.getPriceRevision())
                      .build());
              continue;
            }

            ChannelPrice price = priceOpt.get();
            Money authoritativePrice =
                Money.newBuilder()
                    .setCurrencyCode(
                        price.getCurrencyCode() != null ? price.getCurrencyCode() : "VND")
                    .setUnits(price.getAmountUnits())
                    .setNanos(price.getAmountNanos())
                    .build();

            canonicalSubtotalUnits =
                Math.addExact(
                    canonicalSubtotalUnits,
                    Math.multiplyExact(price.getAmountUnits(), (long) item.getQuantity()));

            boolean priceMatches = true;
            ValidationIssue discrepancyIssue = ValidationIssue.VALIDATION_ISSUE_PRICE_CHANGED;

            if (item.hasClientUnitPrice()) {
              if (StringUtils.hasText(item.getClientUnitPrice().getCurrencyCode())
                  && !"VND".equalsIgnoreCase(item.getClientUnitPrice().getCurrencyCode())) {
                priceMatches = false;
                discrepancyIssue = ValidationIssue.VALIDATION_ISSUE_CURRENCY_MISMATCH;
              } else if (item.getClientUnitPrice().getUnits() != price.getAmountUnits()
                  || item.getClientUnitPrice().getNanos() != price.getAmountNanos()) {
                priceMatches = false;
                discrepancyIssue = ValidationIssue.VALIDATION_ISSUE_PRICE_CHANGED;
              }
            }

            if (!priceMatches) {
              discrepancies.add(
                  PriceDiscrepancy.newBuilder()
                      .setSkuCode(sku)
                      .setClientUnitPrice(item.getClientUnitPrice())
                      .setAuthoritativeUnitPrice(authoritativePrice)
                      .setDiscrepancyReason(discrepancyIssue)
                      .build());
            }

            validatedItems.add(
                ValidatedCheckoutItem.newBuilder()
                    .setProductId(product.getId().toString())
                    .setVariantId(variant.getId().toString())
                    .setSkuCode(sku)
                    .setProductName(product.getName() != null ? product.getName() : "")
                    .setVariantName(
                        variant.getVariantName() != null ? variant.getVariantName() : "")
                    .setQuantity(item.getQuantity())
                    .setAuthoritativeUnitPrice(authoritativePrice)
                    .setNetWeight(
                        grpcMapper.toWeightProto(variant.getWeightValue(), variant.getWeightUnit()))
                    .setPackagingType(grpcMapper.toPackagingTypeProto(variant.getPackagingType()))
                    .setIsSellable(true)
                    .setProductVersion(product.getVersion())
                    .setVariantVersion(variant.getVersion())
                    .setPriceVersion(variant.getPriceRevision())
                    .build());
          }

          boolean isValid = discrepancies.isEmpty();
          Instant now = Instant.now();
          Timestamp evaluatedAt =
              Timestamp.newBuilder()
                  .setSeconds(now.getEpochSecond())
                  .setNanos(now.getNano())
                  .build();

          var responseBuilder =
              ValidateCheckoutPricingResponse.newBuilder()
                  .setIsValid(isValid)
                  .addAllItems(validatedItems)
                  .setCanonicalSubtotal(
                      Money.newBuilder()
                          .setCurrencyCode("VND")
                          .setUnits(canonicalSubtotalUnits)
                          .setNanos(0)
                          .build())
                  .addAllDiscrepancies(discrepancies)
                  .setEvaluatedAt(evaluatedAt);

          if (!isValid) {
            responseBuilder.setError(
                ErrorDetail.newBuilder()
                    .setErrorCode(com.omamx.proto.common.v1.ErrorCode.ERR_CATALOG_PRICE_TAMPERED)
                    .setUserMessage(
                        "Thông tin giá hoặc sản phẩm đã thay đổi, vui lòng tải lại giỏ hàng.")
                    .setDevMessage(
                        "Checkout pricing validation detected discrepancies in items or pricing")
                    .setDomain("catalog")
                    .build());
          }

          return responseBuilder.build();
        });
  }

  @Override
  public void getProductVariant(
      GetProductVariantRequest request, StreamObserver<GetProductVariantResponse> observer) {
    log.debug(
        "[gRPC-Internal] getProductVariant: skuCode={}, channel={}",
        request.getSkuCode(),
        request.getChannel());
    complete(
        observer,
        () -> {
          if (!StringUtils.hasText(request.getSkuCode())) {
            throw new CatalogException(ErrorCode.INVALID_ARGUMENT, "sku_code is required");
          }

          ProductVariant variant =
              variantMapper
                  .findBySkuCode(request.getSkuCode())
                  .orElseThrow(
                      () ->
                          new CatalogException(
                              ErrorCode.CATALOG_VARIANT_NOT_FOUND,
                              "SKU not found: " + request.getSkuCode()));

          Product product =
              productMapper
                  .findById(variant.getProductId())
                  .orElseThrow(
                      () ->
                          new CatalogException(
                              ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                              "Product not found for SKU: " + request.getSkuCode()));

          ChannelPrice price = null;
          if (request.getChannel() != SalesChannel.SALES_CHANNEL_UNSPECIFIED) {
            var domainChannel = grpcMapper.toSalesChannel(request.getChannel());
            if (domainChannel != null) {
              price =
                  channelPriceMapper
                      .findActiveByVariantAndChannel(variant.getId(), domainChannel.getValue())
                      .orElse(null);
            }
          }

          ProductVariantDetail detail = grpcMapper.toVariantDetailProto(variant, product, price);
          return GetProductVariantResponse.newBuilder().setVariant(detail).build();
        });
  }

  @Override
  public void listProductVariants(
      ListProductVariantsRequest request, StreamObserver<ListProductVariantsResponse> observer) {
    log.debug(
        "[gRPC-Internal] listProductVariants: categoryId={}, ocopRating={}, activeOnly={}, channel={}, page={}, pageSize={}",
        request.getCategoryId(),
        request.getOcopRating(),
        request.getActiveOnly(),
        request.getChannel(),
        request.getPagination().getPage(),
        request.getPagination().getPageSize());
    complete(
        observer,
        () -> {
          int page = request.getPagination().getPage();
          if (page < 1) page = 1;
          int pageSize = request.getPagination().getPageSize();
          if (pageSize < 1) pageSize = 20;
          if (pageSize > 100) pageSize = 100;
          int offset = (page - 1) * pageSize;

          UUID categoryId = null;
          if (!request.getCategoryId().isEmpty()) {
            try {
              categoryId = UUID.fromString(request.getCategoryId());
            } catch (IllegalArgumentException e) {
              throw new CatalogException(
                  ErrorCode.INVALID_ARGUMENT,
                  "category_id không phải UUID hợp lệ: " + request.getCategoryId());
            }
          }

          String ocopRating = null;
          if (request.hasOcopRating()
              && request.getOcopRating() != OcopRating.OCOP_RATING_UNSPECIFIED) {
            ocopRating = String.valueOf(request.getOcopRating().getNumber());
          }

          Boolean activeOnly = request.hasActiveOnly() && request.getActiveOnly() ? true : null;

          List<ProductVariant> variants =
              variantMapper.findByFilter(categoryId, ocopRating, activeOnly, offset, pageSize);
          long total = variantMapper.countByFilter(categoryId, ocopRating, activeOnly);

          // Batch load prices theo kênh nếu có
          List<UUID> variantIds =
              variants.stream().map(ProductVariant::getId).collect(Collectors.toList());
          Map<UUID, ChannelPrice> currentPriceByVariant = new HashMap<>();
          var domainChannel = grpcMapper.toSalesChannel(request.getChannel());
          if (domainChannel != null && !variantIds.isEmpty()) {
            for (ChannelPrice price :
                channelPriceMapper.findActiveCurrentByVariantIds(
                    variantIds, domainChannel.getValue())) {
              currentPriceByVariant.putIfAbsent(price.getVariantId(), price);
            }
          }

          // Batch load products
          List<UUID> productIds =
              variants.stream()
                  .map(ProductVariant::getProductId)
                  .distinct()
                  .collect(Collectors.toList());
          Map<UUID, Product> productsById = new HashMap<>();
          for (UUID pid : productIds) {
            productMapper.findById(pid).ifPresent(p -> productsById.put(p.getId(), p));
          }

          List<ProductVariantDetail> variantDetails =
              variants.stream()
                  .map(
                      v ->
                          grpcMapper.toVariantDetailProto(
                              v,
                              productsById.get(v.getProductId()),
                              currentPriceByVariant.get(v.getId())))
                  .collect(Collectors.toList());

          int totalPages = pageSize == 0 ? 0 : (int) Math.ceil((double) total / pageSize);

          return ListProductVariantsResponse.newBuilder()
              .addAllVariants(variantDetails)
              .setPagination(
                  PaginationResponse.newBuilder()
                      .setCurrentPage(page)
                      .setPageSize(pageSize)
                      .setTotalItems(total)
                      .setTotalPages(totalPages)
                      .build())
              .build();
        });
  }

  @Override
  public void getTraceabilityBinding(
      GetTraceabilityBindingRequest request,
      StreamObserver<GetTraceabilityBindingResponse> observer) {
    log.debug(
        "[gRPC-Internal] getTraceabilityBinding: productId={}, skuCode={}",
        request.getProductId(),
        request.getSkuCode());
    complete(
        observer,
        () -> {
          UUID productUuid;
          try {
            productUuid = UUID.fromString(request.getProductId());
          } catch (IllegalArgumentException e) {
            throw new CatalogException(ErrorCode.INVALID_ARGUMENT, "Invalid product UUID");
          }
          ProductVariant variant =
              variantMapper
                  .findBySkuCode(request.getSkuCode())
                  .orElseThrow(
                      () ->
                          new CatalogException(
                              ErrorCode.CATALOG_VARIANT_NOT_FOUND,
                              "SKU not found: " + request.getSkuCode()));
          if (!productUuid.equals(variant.getProductId())) {
            throw new CatalogException(
                ErrorCode.CATALOG_VARIANT_NOT_FOUND,
                "SKU "
                    + request.getSkuCode()
                    + " does not belong to product "
                    + request.getProductId());
          }
          Product product =
              productMapper
                  .findById(productUuid)
                  .orElseThrow(
                      () ->
                          new CatalogException(
                              ErrorCode.CATALOG_PRODUCT_NOT_FOUND,
                              "Product not found: " + request.getProductId()));

          String versionFingerprint =
              variant.getId().toString() + ":" + product.getVersion() + ":" + variant.getVersion();

          return GetTraceabilityBindingResponse.newBuilder()
              .setProductId(product.getId().toString())
              .setSkuCode(variant.getSkuCode())
              .setVersionFingerprint(versionFingerprint)
              .setProductVersion(product.getVersion())
              .setVariantVersion(variant.getVersion())
              .setVariantId(variant.getId().toString())
              .build();
        });
  }

  @FunctionalInterface
  private interface GrpcCall<T> {
    T execute() throws Exception;
  }

  private <T> void complete(StreamObserver<T> observer, GrpcCall<T> call) {
    try {
      observer.onNext(call.execute());
      observer.onCompleted();
    } catch (CatalogException ex) {
      Status status = mapCatalogExceptionToStatus(ex);
      log.warn(
          "[gRPC-Internal] CatalogException: code={}, message={}",
          ex.getErrorCode(),
          ex.getMessage());
      observer.onError(status.withDescription(ex.getMessage()).asRuntimeException());
    } catch (Exception e) {
      log.error("[gRPC-Internal] Error executing call", e);
      observer.onError(Status.INTERNAL.withDescription(e.getMessage()).asRuntimeException());
    }
  }

  private Status mapCatalogExceptionToStatus(CatalogException ex) {
    ErrorCode code = ex.getErrorCode();
    String name = code.name();
    if (name.endsWith("_NOT_FOUND")) return Status.NOT_FOUND;
    if (name.endsWith("_INVALID_STATE")
        || name.equals("CATALOG_PRICE_TAMPERED")
        || name.equals("CATALOG_PRODUCT_OUT_OF_SEASON")) return Status.FAILED_PRECONDITION;
    if (name.equals("INVALID_ARGUMENT")) return Status.INVALID_ARGUMENT;
    if (name.equals("PERMISSION_DENIED") || name.equals("UNAUTHENTICATED"))
      return Status.PERMISSION_DENIED;
    if (name.equals("RESOURCE_NOT_FOUND")) return Status.NOT_FOUND;
    if (name.equals("CIRCUIT_BREAKER_OPEN") || name.equals("DEADLINE_EXCEEDED"))
      return Status.UNAVAILABLE;
    return Status.INTERNAL;
  }
}
