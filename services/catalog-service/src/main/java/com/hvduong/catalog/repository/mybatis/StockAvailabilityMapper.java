package com.hvduong.catalog.repository.mybatis;

import com.hvduong.catalog.infrastructure.inventory.StockAvailabilityProjection;
import com.hvduong.catalog.infrastructure.inventory.StockChangedEvent;
import com.hvduong.catalog.repository.mybatis.projection.AdminStockProjectionRow;

import java.util.List;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

/** Persistence gateway for the Inventory-owned stock read model. */
@Mapper
public interface StockAvailabilityMapper {

  List<StockAvailabilityProjection> findBySkuCodes(@Param("skuCodes") List<String> skuCodes);

  int upsert(StockChangedEvent event);

  List<AdminStockProjectionRow> findAdminStockProjections(
      @Param("q") String q,
      @Param("linked") Boolean linked,
      @Param("offset") int offset,
      @Param("limit") int limit);

  long countAdminStockProjections(@Param("q") String q, @Param("linked") Boolean linked);
}
