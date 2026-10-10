package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import lombok.Data;
import java.util.UUID;
import java.time.OffsetDateTime;

@Data
public class RegistrationOperationRow {
    private UUID operationId;
    private String operationType;
    private String idempotencyKey;
    private String requestFingerprint;
    private UUID principalId;
    private String operationStatus;
    private String errorCode;
    private OffsetDateTime createdAt;
    private OffsetDateTime updatedAt;
    private OffsetDateTime completedAt;
}
