package com.hvduong.catalog.infrastructure.outbox;

import java.time.Instant;
import java.util.UUID;

public record OutboxRow(
    UUID id,
    String aggregateType,
    String aggregateId,
    String eventType,
    int schemaVersion,
    String payload,
    Instant occurredAt) {}
