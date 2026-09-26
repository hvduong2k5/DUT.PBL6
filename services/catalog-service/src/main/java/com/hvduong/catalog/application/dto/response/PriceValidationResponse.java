package com.hvduong.catalog.application.dto.response;

import lombok.Builder;
import lombok.Value;

import java.util.List;

/** Application-level result cho use case validate giá checkout. */
@Value
@Builder
public class PriceValidationResponse {
    boolean valid;
    long canonicalSubtotalUnits;
    String currencyCode;
    List<Discrepancy> discrepancies;

    @Value
    @Builder
    public static class Discrepancy {
        String skuCode;
        long clientUnitPrice;
        long authoritativeUnitPrice;
        String currencyCode;
        int nanos;
        String reason;
    }
}
