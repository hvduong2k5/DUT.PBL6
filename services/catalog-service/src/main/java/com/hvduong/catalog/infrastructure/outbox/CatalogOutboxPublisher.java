package com.hvduong.catalog.infrastructure.outbox;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hvduong.catalog.repository.mybatis.OutboxEventMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

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

    private final OutboxEventMapper outboxEventMapper;
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
        List<OutboxRow> rows = outboxEventMapper.claimDueRows(claimToken, batchSize, leaseSeconds);
        for (OutboxRow row : rows) {
            publish(row, claimToken);
        }
    }

    private void publish(OutboxRow row, UUID claimToken) {
        try {
            String event = serializeCloudEvent(row);
            kafkaTemplate.send(catalogEventsTopic, row.aggregateId(), event).get(10, TimeUnit.SECONDS);
            outboxEventMapper.markPublished(row.id(), claimToken);
        } catch (Exception exception) {
            int delaySeconds = Math.min(3600, 1 << Math.min(12, Math.max(0, retryAttempt(row.id()) - 1)));
            outboxEventMapper.recordFailure(row.id(), claimToken, delaySeconds, abbreviate(exception.getMessage()));
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
        Integer attempts = outboxEventMapper.findAttemptsById(id);
        return attempts == null ? 1 : attempts;
    }

    private String abbreviate(String message) {
        if (message == null) return "Unknown Kafka delivery error";
        return message.length() <= 2000 ? message : message.substring(0, 2000);
    }
}
