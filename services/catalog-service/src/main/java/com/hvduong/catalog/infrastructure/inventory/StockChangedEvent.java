package com.hvduong.catalog.infrastructure.inventory;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.Instant;

/**
 * Absolute sellable-stock snapshot published by Inventory for one SKU.
 *
 * <p>The wire contract uses snake_case to align with the other Inventory events.
 * A snapshot is emitted after every change that can affect sellability, including
 * reservation, expiry and quarantine transitions.</p>
 */
public record StockChangedEvent(
        @JsonProperty("sku_code") String skuCode,
        @JsonProperty("sellable_quantity") long sellableQuantity,
        @JsonProperty("inventory_version") long inventoryVersion,
        @JsonProperty("source_updated_at") Instant sourceUpdatedAt
) { }
