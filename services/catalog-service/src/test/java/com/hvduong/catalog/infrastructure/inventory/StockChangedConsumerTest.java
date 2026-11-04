package com.hvduong.catalog.infrastructure.inventory;

import org.junit.jupiter.api.Test;
import org.springframework.dao.DataAccessResourceFailureException;

import java.time.Instant;

import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class StockChangedConsumerTest {

    private final StockProjectionService projectionService = mock(StockProjectionService.class);
    private final StockChangedConsumer consumer = new StockChangedConsumer(projectionService);

    @Test
    void forwardsDeserializedInventorySnapshotToProjection() {
        StockChangedEvent event = new StockChangedEvent("SKU-1", 12L, 9L,
                Instant.parse("2026-10-10T09:59:00Z"));

        consumer.consume(event);

        verify(projectionService).apply(event);
    }

    @Test
    void propagatesDatabaseFailureSoKafkaDoesNotAcknowledgeTheRecord() {
        StockChangedEvent event = new StockChangedEvent("SKU-1", 12L, 9L, Instant.now());
        when(projectionService.apply(event)).thenThrow(new DataAccessResourceFailureException("PostgreSQL unavailable"));

        assertThatThrownBy(() -> consumer.consume(event))
                .isInstanceOf(DataAccessResourceFailureException.class);
        verify(projectionService).apply(event);
    }
}
