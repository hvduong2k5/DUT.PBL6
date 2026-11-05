package com.hvduong.catalog.application.dto.response;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.Instant;
import java.util.List;

/**
 * Response DTO cho checkout snapshot.
 * Thay thế inner record {@code CheckoutSnapshotService.Snapshot} / {@code Item}.
 * Các trường dùng {@link JsonProperty} để giữ tên khóa JSON (snake_case) khi lưu/đọc
 * snapshot trong cơ sở dữ liệu, đảm bảo tương thích ngược với dữ liệu đã tồn tại.
 */
public record CheckoutSnapshotResponse(

        @JsonProperty("snapshot_id")
        String snapshotId,

        @JsonProperty("catalog_version")
        long catalogVersion,

        @JsonProperty("valid_until")
        Instant validUntil,

        @JsonProperty("items")
        List<Item> items,

        @JsonProperty("subtotal")
        long subtotal,

        @JsonProperty("contract_version")
        String contractVersion,

        @JsonProperty("channel")
        String channel

) {

    public record Item(

            @JsonProperty("sku_code")
            String skuCode,

            @JsonProperty("quantity")
            long quantity,

            @JsonProperty("product_name")
            String productName,

            @JsonProperty("unit_price")
            long unitPrice,

            @JsonProperty("weight_grams")
            long weightGrams,

            @JsonProperty("packaging_type")
            String packagingType,

            @JsonProperty("is_active")
            boolean isActive

    ) {}
}
