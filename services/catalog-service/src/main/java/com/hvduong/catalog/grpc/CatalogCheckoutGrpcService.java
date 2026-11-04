package com.hvduong.catalog.grpc;

import com.hvduong.catalog.application.service.CheckoutSnapshotService;
import com.omamx.proto.catalog.v1.*;
import com.omamx.proto.common.v1.*;
import com.google.protobuf.Timestamp;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import lombok.RequiredArgsConstructor;
import net.devh.boot.grpc.server.service.GrpcService;
import org.springframework.web.server.ResponseStatusException;

@GrpcService
@RequiredArgsConstructor
public class CatalogCheckoutGrpcService extends CatalogServiceGrpc.CatalogServiceImplBase {
    private final CheckoutSnapshotService snapshots;
    @Override
    public void getCheckoutSnapshot(GetCheckoutSnapshotRequest request, StreamObserver<GetCheckoutSnapshotResponse> observer) {
        try {
            if (!"catalog-checkout-1.0".equals(request.getContractVersion())) {
                observer.onError(Status.INVALID_ARGUMENT.withDescription("UNSUPPORTED_CONTRACT_VERSION").asRuntimeException()); return;
            }
            var snapshot = snapshots.create(new CheckoutSnapshotService.Request(request.getItemsList().stream()
                    .map(item -> new CheckoutSnapshotService.Selection(item.getSkuCode(), item.getQuantity())).toList(), request.getChannel()));
            var response = GetCheckoutSnapshotResponse.newBuilder().setSnapshotId(snapshot.snapshot_id()).setCatalogVersion(snapshot.catalog_version())
                    .setContractVersion(snapshot.contract_version()).setValidUntil(Timestamp.newBuilder().setSeconds(snapshot.valid_until().getEpochSecond()).setNanos(snapshot.valid_until().getNano()))
                    .setSubtotal(Money.newBuilder().setCurrencyCode("VND").setUnits(snapshot.subtotal()));
            for (var item : snapshot.items()) response.addItems(CheckoutSnapshotItem.newBuilder().setSkuCode(item.sku_code()).setQuantity(Math.toIntExact(item.quantity()))
                    .setProductName(item.product_name()).setUnitPrice(Money.newBuilder().setCurrencyCode("VND").setUnits(item.unit_price()))
                    .setNetWeight(Weight.newBuilder().setDecimalValue(Long.toString(item.weight_grams())).setUnit(WeightUnit.WEIGHT_UNIT_GRAM))
                    .setPackagingType(item.packaging_type()).setIsActive(item.is_active()));
            observer.onNext(response.build()); observer.onCompleted();
        } catch (ResponseStatusException | IllegalArgumentException error) { observer.onError(Status.FAILED_PRECONDITION.withDescription("CATALOG_SELECTION_UNAVAILABLE").asRuntimeException()); }
        catch (Exception error) { observer.onError(Status.UNAVAILABLE.withDescription("CATALOG_UNAVAILABLE").asRuntimeException()); }
    }
}
