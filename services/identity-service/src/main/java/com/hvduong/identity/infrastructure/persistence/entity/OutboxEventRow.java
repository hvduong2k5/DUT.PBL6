package com.hvduong.identity.infrastructure.persistence.entity;

import lombok.Data;
import java.time.OffsetDateTime;
import java.util.UUID;

@Data
public class OutboxEventRow {
    private UUID eventId;
    private String aggregateType;
    private String aggregateId;
    private String eventType;
    private Integer schemaVersion;
    private String partitionKey;
    private String payload; // JSONB
    private String deliveryStatus;
    private Integer attemptCount;
    private OffsetDateTime nextAttemptAt;
    private String lockedBy;
    private OffsetDateTime lockedUntil;
    private String lastError;
    private OffsetDateTime occurredAt;
    private OffsetDateTime createdAt;
    private OffsetDateTime publishedAt;
}
