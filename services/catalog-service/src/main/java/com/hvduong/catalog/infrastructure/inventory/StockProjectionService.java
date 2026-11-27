package com.hvduong.catalog.infrastructure.inventory;

import com.hvduong.catalog.common.enums.StockStatus;
import com.hvduong.catalog.repository.mybatis.StockAvailabilityMapper;

import java.time.Duration;
import java.time.Instant;
import java.util.Collection;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

/** Application service for Catalog's durable, read-only stock projection. */
@Slf4j
@Service
@RequiredArgsConstructor
public class StockProjectionService {

  private static final Duration STOCK_FRESHNESS_TTL = Duration.ofHours(24);

  private final StockAvailabilityMapper stockAvailabilityMapper;

  private final InventoryStockClient liveInventory;

  /**
   * Returns one entry per requested SKU. A null value means the quantity is not known yet because
   * the SKU has not been synchronized from Inventory.
   */
  public Map<String, Long> getAvailableQuantities(Collection<String> skuCodes) {
    if (liveInventory != null && liveInventory.configured() && skuCodes != null) {
      return liveInventory.available(skuCodes);
    }
    Map<String, Long> quantities = new HashMap<>();
    if (skuCodes == null || skuCodes.isEmpty()) return quantities;

    List<String> distinctSkuCodes =
        skuCodes.stream().filter(sku -> sku != null && !sku.isBlank()).distinct().toList();
    distinctSkuCodes.forEach(sku -> quantities.put(sku, null));
    if (distinctSkuCodes.isEmpty()) return quantities;

    stockAvailabilityMapper
        .findBySkuCodes(distinctSkuCodes)
        .forEach(projection -> quantities.put(projection.skuCode(), projection.sellableQuantity()));
    return quantities;
  }

  /** Lấy trạng thái tồn kho chuẩn hóa dựa trên số lượng khả dụng và độ tươi mới (synced_at). */
  public StockStatus getStockStatus(String skuCode) {
    if (skuCode == null || skuCode.isBlank()) {
      return StockStatus.UNKNOWN;
    }
    if (liveInventory != null && liveInventory.configured()) {
      Map<String, Long> live = liveInventory.available(List.of(skuCode));
      Long qty = live.get(skuCode);
      if (qty == null) return StockStatus.UNKNOWN;
      return qty <= 0 ? StockStatus.OUT_OF_STOCK : StockStatus.IN_STOCK;
    }
    List<StockAvailabilityProjection> projections =
        stockAvailabilityMapper.findBySkuCodes(List.of(skuCode));
    if (projections == null || projections.isEmpty()) {
      return StockStatus.UNKNOWN;
    }
    return resolveStockStatus(projections.get(0));
  }

  /** Đánh giá StockStatus từ projection của SKU. */
  public StockStatus resolveStockStatus(StockAvailabilityProjection projection) {
    if (projection == null || projection.sellableQuantity() == null) {
      return StockStatus.UNKNOWN;
    }
    if (projection.syncedAt() == null
        || projection.syncedAt().isBefore(Instant.now().minus(STOCK_FRESHNESS_TTL))) {
      return StockStatus.UNKNOWN;
    }
    return projection.sellableQuantity() <= 0 ? StockStatus.OUT_OF_STOCK : StockStatus.IN_STOCK;
  }

  /** Lấy projection tồn kho của một SKU (nếu có). */
  public StockAvailabilityProjection getProjection(String skuCode) {
    if (skuCode == null || skuCode.isBlank()) {
      return null;
    }
    List<StockAvailabilityProjection> projections =
        stockAvailabilityMapper.findBySkuCodes(List.of(skuCode));
    return (projections != null && !projections.isEmpty()) ? projections.get(0) : null;
  }

  /**
   * @return true when the event changed the projection; false for stale/duplicate events.
   */
  @Transactional
  public boolean apply(StockChangedEvent event) {
    int updated = stockAvailabilityMapper.upsert(event);
    if (updated == 0) {
      log.debug(
          "Ignored stale or duplicate stock event for SKU {} (version {})",
          event.skuCode(),
          event.inventoryVersion());
      return false;
    }
    log.debug(
        "Updated available-stock projection for SKU {} (version {})",
        event.skuCode(),
        event.inventoryVersion());
    return true;
  }
}
