package com.hvduong.catalog.grpc;

import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.application.service.CatalogService;
import com.omamx.proto.catalog.v1.*;
import io.grpc.stub.StreamObserver;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.server.service.GrpcService;

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
        complete(observer, () -> {
            // Snapshot giá và thông số phục vụ đơn hàng
            return GetCheckoutSnapshotResponse.newBuilder().build();
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
        } catch (Exception e) {
            log.error("[gRPC-Internal] Error executing call", e);
            observer.onError(io.grpc.Status.INTERNAL
                    .withDescription(e.getMessage())
                    .asRuntimeException());
        }
    }
}

