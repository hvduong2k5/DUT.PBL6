package com.hvduong.catalog.application.dto.request;

import com.hvduong.catalog.common.enums.SalesChannel;
import lombok.Builder;
import lombok.Value;

import java.util.List;

/** Application-level input cho use case validate giá checkout. */
@Value
@Builder
public class PriceValidationRequest {
    String idempotencyKey;
    SalesChannel channel;
    List<Item> items;

    @Value
    @Builder
    public static class Item {
        String skuCode;
        int quantity;
        long clientUnitPrice;
        String currencyCode;
        int nanos;
    }
}
