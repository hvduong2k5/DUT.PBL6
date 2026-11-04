package com.hvduong.catalog.infrastructure.snapshot;

import java.math.BigDecimal;
import java.time.Instant;

public record SkuCheckoutDetail(
        String name,
        Long productVersion,
        Long variantVersion,
        BigDecimal weightValue,
        String weightUnit,
        String packagingType,
        Long amountUnits,
        String currencyCode,
        Integer amountNanos,
        Long priceVersion,
        Instant effectiveTo
) { }

