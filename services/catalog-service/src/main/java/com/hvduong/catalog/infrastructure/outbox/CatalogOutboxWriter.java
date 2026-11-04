package com.hvduong.catalog.infrastructure.outbox;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.catalog.repository.mybatis.OutboxEventMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Component;

/** Persists event data in the caller's database transaction. */
@Component
@RequiredArgsConstructor
public class CatalogOutboxWriter {

    private final OutboxEventMapper outboxEventMapper;
    private final ObjectMapper objectMapper;

    public void enqueue(String aggregateType, String aggregateId, String eventType, Object data) {
        try {
            String payload = objectMapper.writeValueAsString(data);
            outboxEventMapper.insert(aggregateType, aggregateId, eventType, payload);
        } catch (JsonProcessingException exception) {
            throw new IllegalArgumentException("Could not serialize Catalog outbox payload", exception);
        }
    }
}
