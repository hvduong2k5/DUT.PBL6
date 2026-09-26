package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.domain.entity.ChannelPrice;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

/**
 * MyBatis Mapper interface cho bảng {@code channel_prices}.
 */
@Mapper
public interface ChannelPriceMapper {

    Optional<ChannelPrice> findActiveByVariantAndChannel(
            @Param("variantId") UUID variantId,
            @Param("channel") String channel
    );

    List<ChannelPrice> findActiveByVariantId(@Param("variantId") UUID variantId);

    List<ChannelPrice> findHistoryByVariantAndChannel(
            @Param("variantId") UUID variantId,
            @Param("channel") String channel
    );

    void insert(ChannelPrice price);

    void supersedePreviousPrice(
            @Param("variantId") UUID variantId,
            @Param("channel") String channel
    );

    Optional<ChannelPrice> findById(@Param("id") UUID id);
}
