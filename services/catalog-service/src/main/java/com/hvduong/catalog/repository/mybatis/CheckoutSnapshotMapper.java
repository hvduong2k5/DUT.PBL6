package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.infrastructure.snapshot.SkuCheckoutDetail;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

@Mapper
public interface CheckoutSnapshotMapper {

    void insertSnapshot(
            @Param("id") UUID id,
            @Param("payload") String payload,
            @Param("expiresAt") Instant expiresAt
    );

    String findValidPayloadById(@Param("id") UUID id);

    Instant getCurrentTimestamp();

    List<SkuCheckoutDetail> findSkuCheckoutDetails(
            @Param("skuCode") String skuCode,
            @Param("channel") String channel
    );
}

