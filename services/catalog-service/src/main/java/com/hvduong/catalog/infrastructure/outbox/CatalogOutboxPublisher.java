package com.hvduong.catalog.infrastructure.outbox;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.sql.Timestamp;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

/** At-least-once outbox publisher; event ID remains stable across retries. */
@Slf4j
@Component
@RequiredArgsConstructor
@ConditionalOnProperty(prefix = "catalog.outbox", name = "enabled", havingValue = "true", matchIfMissing = true)
public class CatalogOutboxPublisher {

    private static final String SOURCE = "https://omama.vn/services/catalog-service";

    private final JdbcTemplate jdbcTemplate;
    private final KafkaTemplate<String, String> kafkaTemplate;
    private final ObjectMapper objectMapper;

    @Value("${catalog.events.topic:catalog.events.v1}")
    private String catalogEventsTopic;

    @Value("${catalog.outbox.batch-size:50}")
    private int batchSize;

    @Value("${catalog.outbox.lease-seconds:30}")
    private int leaseSeconds;

    @Scheduled(fixedDelayString = "${catalog.outbox.poll-interval-ms:1000}")
    public void publishDueEvents() {
        UUID claimToken = UUID.randomUUID();
        List<OutboxRow> rows = claimDueRows(claimToken);
        for (OutboxRow row : rows) publish(row, claimToken);
    }

    private List<OutboxRow> claimDueRows(UUID claimToken) {
        return jdbcTemplate.query("""
                        WITH candidates AS (
                            SELECT id
                            FROM outbox_events
                            WHERE published_at IS NULL
                              AND next_attempt_at <= NOW()
                              AND (claimed_until IS NULL OR claimed_until < NOW())
                            ORDER BY occurred_at, id
                            FOR UPDATE SKIP LOCKED
                            LIMIT ?
                        )
                        UPDATE outbox_events AS event
                        SET claim_token = ?,
                            claimed_until = NOW() + (? * INTERVAL '1 second'),
                            attempts = attempts + 1
                        FROM candidates
                        WHERE event.id = candidates.id
                        RETURNING event.id, event.aggregate_type, event.aggregate_id,
                                  event.event_type, event.schema_version, event.payload,
                                  event.occurred_at
                        """,
                (resultSet, rowNum) -> new OutboxRow(
                        resultSet.getObject("id", UUID.class),
                        resultSet.getString("aggregate_type"),
                        resultSet.getString("aggregate_id"),
                        resultSet.getString("event_type"),
                        resultSet.getInt("schema_version"),
                        resultSet.getString("payload"),
                        resultSet.getTimestamp("occurred_at").toInstant()),
                batchSize, claimToken, leaseSeconds);
    }

    private void publish(OutboxRow row, UUID claimToken) {
        try {
            String event = serializeCloudEvent(row);
            kafkaTemplate.send(catalogEventsTopic, row.aggregateId(), event).get(10, TimeUnit.SECONDS);
            jdbcTemplate.update("""
                            UPDATE outbox_events
                            SET published_at = NOW(), claim_token = NULL, claimed_until = NULL, last_error = NULL
                            WHERE id = ? AND claim_token = ?
                            """,
                    row.id(), claimToken);
        } catch (Exception exception) {
            int delaySeconds = Math.min(3600, 1 << Math.min(12, Math.max(0, retryAttempt(row.id()) - 1)));
            jdbcTemplate.update("""
                            UPDATE outbox_events
                            SET next_attempt_at = NOW() + (? * INTERVAL '1 second'),
                                claim_token = NULL, claimed_until = NULL, last_error = ?
                            WHERE id = ? AND claim_token = ?
                            """,
                    delaySeconds, abbreviate(exception.getMessage()), row.id(), claimToken);
            log.warn("Catalog event {} delivery failed; retrying after {}s", row.id(), delaySeconds, exception);
        }
    }

    private String serializeCloudEvent(OutboxRow row) throws Exception {
        JsonNode data = objectMapper.readTree(row.payload());
        String traceId = UUID.randomUUID().toString().replace("-", "");
        String spanId = UUID.randomUUID().toString().replace("-", "").substring(0, 16);
        Map<String, Object> event = new LinkedHashMap<>();
        event.put("specversion", "1.0");
        event.put("id", row.id().toString());
        event.put("source", SOURCE);
        event.put("type", row.eventType());
        event.put("subject", row.aggregateType().toLowerCase() + ":" + row.aggregateId());
        event.put("time", row.occurredAt().toString());
        event.put("datacontenttype", "application/json");
        event.put("traceparent", "00-" + traceId + "-" + spanId + "-01");
        event.put("dataschema", "urn:omama:catalog:events:v" + row.schemaVersion());
        event.put("data", data);
        return objectMapper.writeValueAsString(event);
    }

    private int retryAttempt(UUID id) {
        Integer attempts = jdbcTemplate.queryForObject(
                "SELECT attempts FROM outbox_events WHERE id = ?", Integer.class, id);
        return attempts == null ? 1 : attempts;
    }

    private String abbreviate(String message) {
        if (message == null) return "Unknown Kafka delivery error";
        return message.length() <= 2000 ? message : message.substring(0, 2000);
    }

    private record OutboxRow(UUID id, String aggregateType, String aggregateId,
                             String eventType, int schemaVersion, String payload, Instant occurredAt) { }
}
