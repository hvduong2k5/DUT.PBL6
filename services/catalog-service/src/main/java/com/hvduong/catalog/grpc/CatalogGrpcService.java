package com.hvduong.catalog.grpc;

import com.google.protobuf.Timestamp;
import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.application.service.CheckoutSnapshotService;
import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;
import com.omamx.proto.catalog.v1.*;
import com.omamx.proto.common.v1.Money;
import com.omamx.proto.common.v1.Weight;
import com.omamx.proto.common.v1.WeightUnit;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.server.service.GrpcService;
import org.springframework.web.server.ResponseStatusException;

/**
 * Service gRPC nội bộ phục vụ giao tiếp liên microservice (East-West Traffic)
 * Điểm neo trên Critical Path: Thẩm định giá bán chống gian lận F12/Inspect cho order-service.
 */
@Slf4j
@GrpcService
@RequiredArgsConstructor
public class CatalogGrpcService extends CatalogServiceGrpc.CatalogServiceImplBase {

    private final CatalogService catalogService;
    private final CatalogGrpcMapper grpcMapper;
    private final CheckoutSnapshotService snapshots;

    @Override
    public void validatePriceAndSKU(ValidatePriceAndSKURequest request,
                                    StreamObserver<ValidatePriceAndSKUResponse> observer) {
        log.debug("[gRPC-Internal] validatePriceAndSKU: itemsCount={}", request.getItemsCount());
        complete(observer, () -> {
            var validationReq = grpcMapper.toValidationRequest(request);
            PriceValidationResponse result = catalogService.validatePrices(validationReq);
            return grpcMapper.toValidationResponse(result);
        });
    }

    @Override
    public void getProductVariant(GetProductVariantRequest request,
                                  StreamObserver<GetProductVariantResponse> observer) {
        log.debug("[gRPC-Internal] getProductVariant: skuCode={}", request.getSkuCode());
        complete(observer, () -> {
            ProductVariantResponse variant = catalogService.getVariantBySkuCode(request.getSkuCode());
            return GetProductVariantResponse.newBuilder()
                    .setVariant(grpcMapper.toVariantDetailProto(variant))
                    .build();
        });
    }

    @Override
    public void listProductVariants(ListProductVariantsRequest request,
                                    StreamObserver<ListProductVariantsResponse> observer) {
        log.debug("[gRPC-Internal] listProductVariants");
        complete(observer, () -> {
            // Danh sách biến thể cơ bản
            return ListProductVariantsResponse.newBuilder().build();
        });
    }

    @Override
    public void getCheckoutSnapshot(GetCheckoutSnapshotRequest request,
                                    StreamObserver<GetCheckoutSnapshotResponse> observer) {
        log.debug("[gRPC-Internal] getCheckoutSnapshot: itemsCount={}", request.getItemsCount());
        try {
            if (!"catalog-checkout-1.0".equals(request.getContractVersion())) {
                observer.onError(Status.INVALID_ARGUMENT.withDescription("UNSUPPORTED_CONTRACT_VERSION").asRuntimeException());
                return;
            }
            var snapshot = snapshots.create(new CheckoutSnapshotService.Request(
                    request.getItemsList().stream()
                            .map(item -> new CheckoutSnapshotService.Selection(item.getSkuCode(), item.getQuantity()))
                            .toList(),
                    request.getChannel()
            ));
            var response = GetCheckoutSnapshotResponse.newBuilder()
                    .setSnapshotId(snapshot.snapshot_id())
                    .setCatalogVersion(snapshot.catalog_version())
                    .setContractVersion(snapshot.contract_version())
                    .setValidUntil(Timestamp.newBuilder()
                            .setSeconds(snapshot.valid_until().getEpochSecond())
                            .setNanos(snapshot.valid_until().getNano()))
                    .setSubtotal(Money.newBuilder().setCurrencyCode("VND").setUnits(snapshot.subtotal()));

            for (var item : snapshot.items()) {
                response.addItems(CheckoutSnapshotItem.newBuilder()
                        .setSkuCode(item.sku_code())
                        .setQuantity(Math.toIntExact(item.quantity()))
                        .setProductName(item.product_name())
                        .setUnitPrice(Money.newBuilder().setCurrencyCode("VND").setUnits(item.unit_price()))
                        .setNetWeight(Weight.newBuilder()
                                .setDecimalValue(Long.toString(item.weight_grams()))
                                .setUnit(WeightUnit.WEIGHT_UNIT_GRAM))
                        .setPackagingType(item.packaging_type())
                        .setIsActive(item.is_active()));
            }
            observer.onNext(response.build());
            observer.onCompleted();
        } catch (ResponseStatusException | IllegalArgumentException error) {
            observer.onError(Status.FAILED_PRECONDITION.withDescription("CATALOG_SELECTION_UNAVAILABLE").asRuntimeException());
        } catch (Exception error) {
            log.error("[gRPC-Internal] Error executing getCheckoutSnapshot", error);
            observer.onError(Status.UNAVAILABLE.withDescription("CATALOG_UNAVAILABLE").asRuntimeException());
        }
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
            log.warn("[gRPC-Internal] CatalogException: code={}, message={}", ex.getErrorCode(), ex.getMessage());
            observer.onError(status.withDescription(ex.getMessage()).asRuntimeException());
        } catch (Exception e) {
            log.error("[gRPC-Internal] Error executing call", e);
            observer.onError(Status.INTERNAL
                    .withDescription(e.getMessage())
                    .asRuntimeException());
        }
    }

    private Status mapCatalogExceptionToStatus(CatalogException ex) {
        ErrorCode code = ex.getErrorCode();
        String name = code.name();
        if (name.endsWith("_NOT_FOUND")) return Status.NOT_FOUND;
        if (name.endsWith("_INVALID_STATE") || name.equals("CATALOG_PRICE_TAMPERED")
                || name.equals("CATALOG_PRODUCT_OUT_OF_SEASON")) return Status.FAILED_PRECONDITION;
        if (name.equals("INVALID_ARGUMENT")) return Status.INVALID_ARGUMENT;
        if (name.equals("PERMISSION_DENIED") || name.equals("UNAUTHENTICATED")) return Status.PERMISSION_DENIED;
        if (name.equals("RESOURCE_NOT_FOUND")) return Status.NOT_FOUND;
        if (name.equals("CIRCUIT_BREAKER_OPEN") || name.equals("DEADLINE_EXCEEDED")) return Status.UNAVAILABLE;
        return Status.INTERNAL;
    }
}
