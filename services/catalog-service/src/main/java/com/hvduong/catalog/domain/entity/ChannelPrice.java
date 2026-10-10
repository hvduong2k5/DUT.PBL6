package com.hvduong.catalog.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.Instant;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ChannelPrice {
    private UUID id;
    private UUID variantId;
    private com.hvduong.catalog.common.enums.SalesChannel channel;
    private String currencyCode;
    private long amountUnits;
    private int amountNanos;
    private Instant effectiveFrom;
    private Instant effectiveTo;
    private String status;
    private long version;
    private String createdBy;
    private Instant createdAt;
    private Instant updatedAt;
}
