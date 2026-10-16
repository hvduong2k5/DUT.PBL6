package com.hvduong.identity.infrastructure.outbox;

import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.OutboxEventMapper;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.OutboxEventRow;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.time.OffsetDateTime;
import java.util.List;

@Slf4j
@Component
@RequiredArgsConstructor
public class OutboxPublisher {

    private final OutboxEventMapper outboxEventMapper;
    private final KafkaTemplate<String, String> kafkaTemplate;

    @Scheduled(fixedDelay = 2000)
    public void publishPendingEvents() {
        OffsetDateTime now = OffsetDateTime.now();
        List<OutboxEventRow> pendingEvents = outboxEventMapper.findPendingEvents(50, now);

        if (pendingEvents.isEmpty()) {
            return;
        }

        log.debug("Found {} pending outbox events to publish", pendingEvents.size());

        for (OutboxEventRow event : pendingEvents) {
            try {
                String topic = "identity.events.v1";
                String key = event.getPartitionKey();
                String payload = event.getPayload();

                // Send event asynchronously and register callback or wait
                kafkaTemplate.send(topic, key, payload).whenComplete((result, ex) -> {
                    if (ex == null) {
                        outboxEventMapper.markPublished(event.getEventId(), OffsetDateTime.now());
                        log.info("Successfully published outbox event: id={}, type={}",
                                event.getEventId(), event.getEventType());
                    } else {
                        log.error("Failed to publish outbox event id={}: {}",
                                event.getEventId(), ex.getMessage());
                        int nextDelaySeconds = (int) Math.min(300, Math.pow(2, event.getAttemptCount() + 1) * 5);
                        outboxEventMapper.recordFailure(
                                event.getEventId(),
                                ex.getMessage(),
                                OffsetDateTime.now().plusSeconds(nextDelaySeconds)
                        );
                    }
                });

            } catch (Exception ex) {
                log.error("Error dispatching outbox event id={}", event.getEventId(), ex);
                int nextDelaySeconds = (int) Math.min(300, Math.pow(2, event.getAttemptCount() + 1) * 5);
                outboxEventMapper.recordFailure(
                        event.getEventId(),
                        ex.getMessage(),
                        OffsetDateTime.now().plusSeconds(nextDelaySeconds)
                );
            }
        }
    }
}

