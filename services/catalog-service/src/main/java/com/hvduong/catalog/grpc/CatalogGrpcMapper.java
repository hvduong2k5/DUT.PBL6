package com.hvduong.catalog.grpc;

import com.hvduong.catalog.application.dto.request.PriceValidationRequest;
import com.hvduong.catalog.application.dto.response.MoneyResponse;
import com.hvduong.catalog.application.dto.response.PriceValidationResponse;
import com.hvduong.catalog.application.dto.response.ProductVariantResponse;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.omamx.proto.catalog.v1.*;
import com.omamx.proto.common.v1.Money;
import com.omamx.proto.common.v1.Weight;
import com.omamx.proto.common.v1.WeightUnit;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Component;

import java.math.BigDecimal;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Mapper giữa Protobuf generated classes của omamx.catalog.v1 và Application DTOs.
 */
@Component
@RequiredArgsConstructor
public class CatalogGrpcMapper {

    public PriceValidationRequest toValidationRequest(ValidatePriceAndSKURequest proto) {
        List<PriceValidationRequest.Item> items = proto.getItemsList().stream()
                .map(i -> PriceValidationRequest.Item.builder()
                        .skuCode(i.getSkuCode())
                        .quantity(i.getQuantity())
                        .clientUnitPrice(i.hasClientUnitPrice() ? i.getClientUnitPrice().getUnits() : 0L)
                        .currencyCode(i.hasClientUnitPrice() && !i.getClientUnitPrice().getCurrencyCode().isBlank()
                                ? i.getClientUnitPrice().getCurrencyCode() : "VND")
                        .nanos(i.hasClientUnitPrice() ? i.getClientUnitPrice().getNanos() : 0)
                        .build())
                .collect(Collectors.toList());

        return PriceValidationRequest.builder()
                .idempotencyKey(proto.getIdempotencyKey())
                .channel(toSalesChannel(proto.getChannel()))
                .items(items)
                .build();
    }

    public ValidatePriceAndSKUResponse toValidationResponse(PriceValidationResponse resp) {
        var builder = ValidatePriceAndSKUResponse.newBuilder()
                .setIsValid(resp.isValid())
                .setCanonicalSubtotal(Money.newBuilder()
                        .setCurrencyCode(resp.getCurrencyCode() != null ? resp.getCurrencyCode() : "VND")
                        .setUnits(resp.getCanonicalSubtotalUnits())
                        .setNanos(0)
                        .build());

        if (resp.getDiscrepancies() != null) {
            resp.getDiscrepancies().forEach(d -> builder.addDiscrepancies(
                    PriceDiscrepancy.newBuilder()
                            .setSkuCode(d.getSkuCode())
                            .setClientUnitPrice(Money.newBuilder()
                                    .setCurrencyCode(d.getCurrencyCode() != null ? d.getCurrencyCode() : "VND")
                                    .setUnits(d.getClientUnitPrice())
                                    .setNanos(d.getNanos())
                                    .build())
                            .setAuthoritativeUnitPrice(Money.newBuilder()
                                    .setCurrencyCode(d.getCurrencyCode() != null ? d.getCurrencyCode() : "VND")
                                    .setUnits(d.getAuthoritativeUnitPrice())
                                    .setNanos(0)
                                    .build())
                            .setDiscrepancyReason(d.getReason() != null ? d.getReason() : "")
                            .build()
            ));
        }

        return builder.build();
    }

    public ProductVariantDetail toVariantDetailProto(ProductVariantResponse v) {
        var builder = ProductVariantDetail.newBuilder()
                .setSkuCode(v.getSkuCode() != null ? v.getSkuCode() : "")
                .setProductId(v.getVariantId() != null ? v.getVariantId().toString() : "")
                .setVariantName(v.getName() != null ? v.getName() : "")
                .setPackagingType(v.getPackagingType() != null ? v.getPackagingType() : "")
                .setShelfLifeDays(v.getShelfLifeDays() != null ? v.getShelfLifeDays() : 90)
                .setIsActive(v.getListingStatus() != null && "ACTIVE".equalsIgnoreCase(v.getListingStatus()));

        if (v.getPrice() != null) {
            builder.setUnitPrice(toMoneyProto(v.getPrice()));
        }

        if (v.getWeightValue() != null) {
            builder.setNetWeight(toWeightProto(v.getWeightValue(), v.getWeightUnit()));
        }

        return builder.build();
    }

    public Money toMoneyProto(MoneyResponse m) {
        if (m == null) {
            return Money.newBuilder().setCurrencyCode("VND").setUnits(0).setNanos(0).build();
        }
        return Money.newBuilder()
                .setCurrencyCode(m.getCurrencyCode() != null ? m.getCurrencyCode() : "VND")
                .setUnits(m.getUnits())
                .setNanos(m.getNanos())
                .build();
    }

    public Weight toWeightProto(BigDecimal weightValue, String unit) {
        WeightUnit weightUnit = WeightUnit.WEIGHT_UNIT_GRAM;
        if (unit != null && unit.toUpperCase().contains("KG")) {
            weightUnit = WeightUnit.WEIGHT_UNIT_KILOGRAM;
        }
        return Weight.newBuilder()
                .setDecimalValue(weightValue != null ? weightValue.toPlainString() : "0")
                .setUnit(weightUnit)
                .build();
    }

    public SalesChannel toSalesChannel(String channelStr) {
        if (channelStr == null || channelStr.isBlank()) {
            return SalesChannel.D2C_WEB;
        }
        try {
            return SalesChannel.valueOf(channelStr.toUpperCase());
        } catch (IllegalArgumentException e) {
            return SalesChannel.D2C_WEB;
        }
    }
}
