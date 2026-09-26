package com.hvduong.catalog.grpc;

import com.hvduong.catalog.application.dto.response.ProductDetailResponse;
import com.hvduong.catalog.application.service.CatalogAdminService;
import com.hvduong.catalog.application.service.CatalogService;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.grpc.v1.*;
import io.grpc.stub.StreamObserver;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.server.service.GrpcService;

import java.util.UUID;

/**
 * gRPC adapter/server cho Catalog Service.
 * <p>
 * Nhận protobuf request → validate cơ bản → gọi Application Service → map response.
 * Không chứa business logic — chỉ orchestrate.
 * <p>
 * Exception handling được xử lý bởi {@code GrpcExceptionInterceptor} ở tầng trên.
 */
@Slf4j
@GrpcService
@RequiredArgsConstructor
public class CatalogGrpcService extends CatalogServiceGrpc.CatalogServiceImplBase {

    private final CatalogService catalogService;
    private final CatalogAdminService adminService;
    private final CatalogGrpcMapper grpcMapper;

    // ── Public API ────────────────────────────────────────────────────────────

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

    // ── Admin API ─────────────────────────────────────────────────────────────

    @Override
    public void createProduct(CreateProductRpcRequest request,
                              StreamObserver<ProductCommandResponse> observer) {
        log.info("[gRPC] createProduct: actor={}", request.getActorId());
        complete(observer, () -> {
            var req = grpcMapper.toCreateProductRequest(request);
            ProductDetailResponse result = adminService.createProduct(req, request.getActorId());
            return ProductCommandResponse.newBuilder()
                    .setProduct(grpcMapper.toProductDetailProto(result)).build();
        });
    }

    @Override
    public void updateProduct(UpdateProductRpcRequest request,
                              StreamObserver<ProductCommandResponse> observer) {
        log.info("[gRPC] updateProduct: id={}", request.getProductId());
        complete(observer, () -> {
            var req = grpcMapper.toUpdateProductRequest(request);
            ProductDetailResponse result = adminService.updateProduct(
                    UUID.fromString(request.getProductId()), req, request.getActorId());
            return ProductCommandResponse.newBuilder()
                    .setProduct(grpcMapper.toProductDetailProto(result)).build();
        });
    }

    @Override
    public void transitionProduct(ProductTransitionRequest request,
                                  StreamObserver<ProductCommandResponse> observer) {
        log.info("[gRPC] transitionProduct: id={}, transition={}", request.getProductId(), request.getTransition());
        complete(observer, () -> {
            UUID productId = UUID.fromString(request.getProductId());
            String actorId = request.getActorId();
            long version   = request.getExpectedVersion();

            ProductDetailResponse result = switch (request.getTransition()) {
                case SUBMIT_FOR_REVIEW -> adminService.submitProductForReview(productId, actorId, version);
                case APPROVE           -> adminService.approveProduct(productId, actorId, version);
                case REJECT            -> adminService.rejectProduct(productId, actorId, version);
                case SUSPEND           -> adminService.suspendProduct(productId, actorId, version);
                case ARCHIVE           -> adminService.archiveProduct(productId, actorId, version);
                default -> throw new IllegalArgumentException(
                        "Unsupported transition: " + request.getTransition());
            };

            return ProductCommandResponse.newBuilder()
                    .setProduct(grpcMapper.toProductDetailProto(result)).build();
        });
    }

    @Override
    public void createVariant(CreateVariantRpcRequest request,
                              StreamObserver<ProductCommandResponse> observer) {
        log.info("[gRPC] createVariant: productId={}", request.getProductId());
        complete(observer, () -> {
            var req = grpcMapper.toCreateVariantRequest(request);
            ProductDetailResponse result = adminService.addVariant(
                    UUID.fromString(request.getProductId()), req, request.getActorId());
            return ProductCommandResponse.newBuilder()
                    .setProduct(grpcMapper.toProductDetailProto(result)).build();
        });
    }

    @Override
    public void updateVariantPrices(UpdateVariantPricesRpcRequest request,
                                    StreamObserver<ProductCommandResponse> observer) {
        log.info("[gRPC] updateVariantPrices: productId={}, variantId={}",
                request.getProductId(), request.getVariantId());
        complete(observer, () -> {
            var req = grpcMapper.toUpdatePriceRequest(request);
            ProductDetailResponse result = adminService.updateVariantPrices(
                    UUID.fromString(request.getProductId()),
                    UUID.fromString(request.getVariantId()),
                    req, request.getActorId());
            return ProductCommandResponse.newBuilder()
                    .setProduct(grpcMapper.toProductDetailProto(result)).build();
        });
    }

    // ── Helper ────────────────────────────────────────────────────────────────

    /**
     * Bọc gọi service + hoàn thành stream, để exception được đẩy lên
     * {@code GrpcExceptionInterceptor} xử lý.
     */
    @FunctionalInterface
    private interface GrpcCall<T> {
        T execute() throws Exception;
    }

    private <T> void complete(StreamObserver<T> observer, GrpcCall<T> call) {
        try {
            observer.onNext(call.execute());
            observer.onCompleted();
        } catch (Exception e) {
            // Re-throw để GrpcExceptionInterceptor bắt
            if (e instanceof RuntimeException re) throw re;
            throw new RuntimeException(e);
        }
    }
}
