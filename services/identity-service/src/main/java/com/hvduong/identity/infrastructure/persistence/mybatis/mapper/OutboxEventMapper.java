package com.hvduong.identity.infrastructure.persistence.mybatis.mapper;

import com.hvduong.identity.infrastructure.persistence.entity.*;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.time.OffsetDateTime;
import java.util.List;
import java.util.UUID;

@Mapper
public interface OutboxEventMapper {

    void insert(OutboxEventRow row);

    List<OutboxEventRow> findPendingEvents(
            @Param("limit") int limit,
            @Param("now") OffsetDateTime now
    );

    int markPublished(
            @Param("eventId") UUID eventId,
            @Param("publishedAt") OffsetDateTime publishedAt
    );

    int recordFailure(
            @Param("eventId") UUID eventId,
            @Param("lastError") String lastError,
            @Param("nextAttemptAt") OffsetDateTime nextAttemptAt
    );
}
