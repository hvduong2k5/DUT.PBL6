package com.hvduong.catalog.repository.mybatis.projection;

import java.util.UUID;

import lombok.Data;

/** Projection cho danh mục có Product trong tập admin đang xem. */
@Data
public class AdminCategoryRow {
  private UUID categoryId;
  private String categoryName;
}
