package com.hvduong.catalog.infrastructure.outbox;

import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import com.hvduong.catalog.repository.mybatis.OutboxEventMapper;

import java.time.Instant;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.CompletableFuture;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.test.util.ReflectionTestUtils;

import com.fasterxml.jackson.databind.ObjectMapper;

class CatalogOutboxPublisherTest {

  private OutboxEventMapper outboxEventMapper;

  @SuppressWarnings("unchecked")
  private final KafkaTemplate<String, String> kafkaTemplate = mock(KafkaTemplate.class);

  private ObjectMapper objectMapper;
  private CatalogOutboxPublisher publisher;

  @BeforeEach
  void setUp() {
    outboxEventMapper = mock(OutboxEventMapper.class);
    objectMapper = new ObjectMapper();
    publisher = new CatalogOutboxPublisher(outboxEventMapper, kafkaTemplate, objectMapper);
    ReflectionTestUtils.setField(publisher, "catalogEventsTopic", "catalog.events.v1");
    ReflectionTestUtils.setField(publisher, "batchSize", 50);
    ReflectionTestUtils.setField(publisher, "leaseSeconds", 30);
  }

  @Test
  void publishDueEvents_successfullyPublishesAndMarksPublished() {
    UUID eventId = UUID.randomUUID();
    OutboxRow row =
        new OutboxRow(
            eventId,
            "Product",
            "prod-123",
            "ProductApproved",
            1,
            "{\"productId\":\"prod-123\"}",
            Instant.now());

    when(outboxEventMapper.claimDueRows(any(UUID.class), eq(50), eq(30))).thenReturn(List.of(row));
    when(kafkaTemplate.send(eq("catalog.events.v1"), eq("prod-123"), anyString()))
        .thenReturn(CompletableFuture.completedFuture(null));

    publisher.publishDueEvents();

    verify(kafkaTemplate).send(eq("catalog.events.v1"), eq("prod-123"), anyString());
    verify(outboxEventMapper).markPublished(eq(eventId), any(UUID.class));
    verify(outboxEventMapper, never()).recordFailure(any(), any(), anyInt(), any());
  }

  @Test
  void publishDueEvents_onKafkaFailure_recordsFailureWithBackoff() {
    UUID eventId = UUID.randomUUID();
    OutboxRow row =
        new OutboxRow(
            eventId,
            "Product",
            "prod-123",
            "ProductApproved",
            1,
            "{\"productId\":\"prod-123\"}",
            Instant.now());

    when(outboxEventMapper.claimDueRows(any(UUID.class), eq(50), eq(30))).thenReturn(List.of(row));
    CompletableFuture failedFuture = new CompletableFuture();
    failedFuture.completeExceptionally(new RuntimeException("Kafka unreachable"));
    when(kafkaTemplate.send(eq("catalog.events.v1"), eq("prod-123"), anyString()))
        .thenReturn(failedFuture);
    when(outboxEventMapper.findAttemptsById(eventId)).thenReturn(1);

    publisher.publishDueEvents();

    verify(outboxEventMapper, never()).markPublished(any(), any());
    verify(outboxEventMapper)
        .recordFailure(eq(eventId), any(UUID.class), anyInt(), contains("Kafka unreachable"));
  }
}
