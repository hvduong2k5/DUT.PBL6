package com.hvduong.catalog.infrastructure.inventory;

import com.hvduong.catalog.repository.mybatis.StockAvailabilityMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Collection;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/** Application service for Catalog's durable, read-only stock projection. */
@Slf4j
@Service
@RequiredArgsConstructor
public class StockProjectionService {

    private final StockAvailabilityMapper stockAvailabilityMapper;

    /**
     * Returns one entry per requested SKU. A null value means the quantity is not
     * known yet because the SKU has not been synchronized from Inventory.
     */
    public Map<String, Long> getAvailableQuantities(Collection<String> skuCodes) {
        Map<String, Long> quantities = new HashMap<>();
        if (skuCodes == null || skuCodes.isEmpty()) return quantities;

        List<String> distinctSkuCodes = skuCodes.stream().filter(sku -> sku != null && !sku.isBlank())
                .distinct().toList();
        distinctSkuCodes.forEach(sku -> quantities.put(sku, null));
        if (distinctSkuCodes.isEmpty()) return quantities;

        stockAvailabilityMapper.findBySkuCodes(distinctSkuCodes)
                .forEach(projection -> quantities.put(projection.skuCode(), projection.sellableQuantity()));
        return quantities;
    }

    /** @return true when the event changed the projection; false for stale/duplicate events. */
    @Transactional
    public boolean apply(StockChangedEvent event) {
        int updated = stockAvailabilityMapper.upsert(event);
        if (updated == 0) {
            log.debug("Ignored stale or duplicate stock event for SKU {} (version {})",
                    event.skuCode(), event.inventoryVersion());
            return false;
        }
        log.debug("Updated available-stock projection for SKU {} (version {})",
                event.skuCode(), event.inventoryVersion());
        return true;
    }
}
