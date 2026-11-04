package com.hvduong.catalog.infrastructure.outbox;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

/** Persists event data in the caller's database transaction. */
@Component
@RequiredArgsConstructor
public class CatalogOutboxWriter {

    private final JdbcTemplate jdbcTemplate;
    private final ObjectMapper objectMapper;

    public void enqueue(String aggregateType, String aggregateId, String eventType, Object data) {
        try {
            String payload = objectMapper.writeValueAsString(data);
            jdbcTemplate.update("""
                            INSERT INTO outbox_events
                                (aggregate_type, aggregate_id, event_type, schema_version, payload)
                            VALUES (?, ?, ?, 1, ?::jsonb)
                            """,
                    aggregateType, aggregateId, eventType, payload);
        } catch (JsonProcessingException exception) {
            throw new IllegalArgumentException("Could not serialize Catalog outbox payload", exception);
        }
    }
}
