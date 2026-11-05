package com.hvduong.catalog.application.dto.request;

import java.util.List;

import com.fasterxml.jackson.annotation.JsonAlias;

/**
 * Request DTO để tạo checkout snapshot. Thay thế inner record {@code
 * CheckoutSnapshotService.Request} / {@code Selection}.
 */
public record CheckoutSnapshotRequest(List<Selection> items, String channel) {

  public record Selection(@JsonAlias("sku_code") String skuCode, int quantity) {}
}
