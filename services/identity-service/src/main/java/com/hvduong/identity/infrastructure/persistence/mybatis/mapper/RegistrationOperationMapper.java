package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Param;
import java.util.UUID;
import java.time.OffsetDateTime;

public interface RegistrationOperationMapper {
    void insert(RegistrationOperationRow row);

    RegistrationOperationRow findByIdempotency(
        @Param("operationType") String operationType,
        @Param("idempotencyKey") String idempotencyKey
    );

    int updateStatus(
        @Param("operationId") UUID operationId,
        @Param("newStatus") String newStatus,
        @Param("errorCode") String errorCode,
        @Param("completedAt") OffsetDateTime completedAt
    );
}
