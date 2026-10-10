package com.hvduong.catalog.infrastructure.inventory;

import com.hvduong.catalog.repository.mybatis.StockAvailabilityMapper;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class StockProjectionServiceTest {

    private final StockAvailabilityMapper mapper = mock(StockAvailabilityMapper.class);
    private final StockProjectionService service = new StockProjectionService(mapper);

    @Test
    void returnsNullForSkuWhoseInventoryStateHasNotBeenSynchronized() {
        when(mapper.findBySkuCodes(List.of("SKU-1", "SKU-2"))).thenReturn(List.of(
                new StockAvailabilityProjection("SKU-1", 7L, 4L, Instant.now(), Instant.now())));

        Map<String, Long> quantities = service.getAvailableQuantities(List.of("SKU-1", "SKU-2", "SKU-1"));

        assertThat(quantities).containsEntry("SKU-1", 7L).containsEntry("SKU-2", null);
    }

    @Test
    void appliesEventWithNewerVersion() {
        StockChangedEvent event = new StockChangedEvent("SKU-1", 4L, 8L, Instant.now());
        when(mapper.upsert(event)).thenReturn(1);

        assertThat(service.apply(event)).isTrue();

        verify(mapper).upsert(eq(event));
    }

    @Test
    void ignoresEventWithTheSameVersion() {
        StockChangedEvent event = new StockChangedEvent("SKU-1", 4L, 8L, Instant.now());
        when(mapper.upsert(event)).thenReturn(0);

        assertThat(service.apply(event)).isFalse();
    }

    @Test
    void ignoresEventWithAnOlderVersion() {
        StockChangedEvent event = new StockChangedEvent("SKU-1", 99L, 7L, Instant.now());
        when(mapper.upsert(event)).thenReturn(0);

        assertThat(service.apply(event)).isFalse();
    }
}
