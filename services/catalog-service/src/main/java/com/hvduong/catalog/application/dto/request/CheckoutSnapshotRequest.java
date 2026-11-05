package com.hvduong.catalog.application.dto.request;

import com.fasterxml.jackson.annotation.JsonAlias;

import java.util.List;

/**
 * Request DTO để tạo checkout snapshot.
 * Thay thế inner record {@code CheckoutSnapshotService.Request} / {@code Selection}.
 */
public record CheckoutSnapshotRequest(

        List<Selection> items,
        String channel

) {

    public record Selection(

            @JsonAlias("sku_code")
            String skuCode,

            int quantity

    ) {}
}
