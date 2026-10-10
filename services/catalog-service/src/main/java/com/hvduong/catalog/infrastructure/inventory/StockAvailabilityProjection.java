package com.hvduong.catalog.infrastructure.inventory;

import java.time.Instant;

/**
 * Read model tồn kho theo SKU trong Catalog.
 *
 * <p>Đây là projection nhận từ Inventory, không phải nguồn chân lý tồn kho.</p>
 */
public record StockAvailabilityProjection(
        String skuCode,
        Long sellableQuantity,
        long inventoryVersion,
        Instant sourceUpdatedAt,
        Instant syncedAt
) { }
