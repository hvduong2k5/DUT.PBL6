package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.infrastructure.inventory.StockAvailabilityProjection;
import com.hvduong.catalog.infrastructure.inventory.StockChangedEvent;

import java.util.List;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

/** Persistence gateway for the Inventory-owned stock read model. */
@Mapper
public interface StockAvailabilityMapper {

  List<StockAvailabilityProjection> findBySkuCodes(@Param("skuCodes") List<String> skuCodes);

  int upsert(StockChangedEvent event);
}
