package com.hvduong.catalog.infrastructure.inventory;

import lombok.RequiredArgsConstructor;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

/**
 * Receives an already-deserialized Inventory snapshot. Exceptions are deliberately
 * propagated: with record acknowledgement Kafka commits the offset only after the
 * transactional projection update returns successfully.
 */
@Component
@RequiredArgsConstructor
public class StockChangedConsumer {

    private final StockProjectionService stockProjectionService;

    @KafkaListener(topics = "${app.kafka.inventory-stock-topic}")
    public void consume(StockChangedEvent event) {
        validate(event);
        stockProjectionService.apply(event);
    }

    private void validate(StockChangedEvent event) {
        if (event == null || event.skuCode() == null || event.skuCode().isBlank()
                || event.sellableQuantity() < 0 || event.inventoryVersion() < 0
                || event.sourceUpdatedAt() == null) {
            throw new IllegalArgumentException("Stock event has invalid required fields");
        }
    }
}
