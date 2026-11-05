package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.infrastructure.outbox.OutboxRow;

import java.util.List;
import java.util.UUID;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

@Mapper
public interface OutboxEventMapper {

  void insert(
      @Param("aggregateType") String aggregateType,
      @Param("aggregateId") String aggregateId,
      @Param("eventType") String eventType,
      @Param("payload") String payload);

  List<OutboxRow> claimDueRows(
      @Param("claimToken") UUID claimToken,
      @Param("batchSize") int batchSize,
      @Param("leaseSeconds") int leaseSeconds);

  int markPublished(@Param("id") UUID id, @Param("claimToken") UUID claimToken);

  int recordFailure(
      @Param("id") UUID id,
      @Param("claimToken") UUID claimToken,
      @Param("delaySeconds") int delaySeconds,
      @Param("lastError") String lastError);

  Integer findAttemptsById(@Param("id") UUID id);
}
