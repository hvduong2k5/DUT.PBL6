package com.hvduong.catalog.grpc;

import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.grpc.v1.*;
import io.grpc.stub.StreamObserver;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.server.service.GrpcService;

@Slf4j
@GrpcService
@RequiredArgsConstructor
public class CatalogPublicGrpcService extends CatalogPublicServiceGrpc.CatalogPublicServiceImplBase {

    private final CatalogService catalogService;
    private final CatalogGrpcMapper grpcMapper;

    @Override
    public void listCategories(Empty request, StreamObserver<ListCategoriesResponse> observer) {
        log.debug("[gRPC] listCategories");
        complete(observer, () -> {
            var categories = catalogService.getAllCategories();
            var builder = ListCategoriesResponse.newBuilder();
            categories.forEach(c -> builder.addCategories(grpcMapper.toCategoryProto(c)));
            return builder.build();
        });
    }

    @Override
    public void listProducts(ListProductsRequest request, StreamObserver<ListProductsResponse> observer) {
        log.debug("[gRPC] listProducts");
        complete(observer, () -> {
            var filter = grpcMapper.toFilterRequest(request);
            var page = catalogService.getPublicProducts(filter);
            return grpcMapper.toListProductsResponse(page);
        });
    }

    @Override
    public void getProduct(GetProductRequest request, StreamObserver<GetProductResponse> observer) {
        log.debug("[gRPC] getProduct: idOrSlug={}", request.getIdOrSlug());
        complete(observer, () -> {
            SalesChannel channel = grpcMapper.toSalesChannel(request.getChannel());
            var product = catalogService.getPublicProductDetail(request.getIdOrSlug(), channel);
            return GetProductResponse.newBuilder()
                    .setProduct(grpcMapper.toProductDetailProto(product))
                    .build();
        });
    }

    @Override
    public void searchProducts(SearchProductsRequest request, StreamObserver<ListProductsResponse> observer) {
        log.debug("[gRPC] searchProducts: query={}", request.getQuery());
        complete(observer, () -> {
            int page     = request.hasPagination() ? Math.max(1, request.getPagination().getPage()) : 1;
            int pageSize = request.hasPagination() ? Math.min(100, request.getPagination().getPageSize()) : 20;
            var result = catalogService.searchProducts(request.getQuery(), page, pageSize);
            return grpcMapper.toListProductsResponse(result);
        });
    }

    @Override
    public void getProductVariant(GetProductVariantRequest request,
                                  StreamObserver<GetProductVariantResponse> observer) {
        log.debug("[gRPC] getProductVariant: skuCode={}", request.getSkuCode());
        complete(observer, () -> {
            var variant = catalogService.getVariantBySkuCode(request.getSkuCode());
            return GetProductVariantResponse.newBuilder()
                    .setVariant(grpcMapper.toVariantProto(variant))
                    .build();
        });
    }

    @Override
    public void validatePriceAndSku(ValidatePriceAndSkuRequest request,
                                    StreamObserver<ValidatePriceAndSkuResponse> observer) {
        log.debug("[gRPC] validatePriceAndSku: items={}", request.getItemsCount());
        complete(observer, () -> {
            var validationReq = grpcMapper.toValidationRequest(request);
            var result = catalogService.validatePrices(validationReq);
            return grpcMapper.toValidationResponse(result);
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
            if (e instanceof RuntimeException re) throw re;
            throw new RuntimeException(e);
        }
    }
}
