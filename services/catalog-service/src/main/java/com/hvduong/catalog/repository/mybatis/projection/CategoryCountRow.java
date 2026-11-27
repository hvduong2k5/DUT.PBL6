package com.hvduong.catalog.repository.mybatis.projection;

import java.util.UUID;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Hàng kết quả aggregate đếm số lượng product theo category. */
@Data
@NoArgsConstructor
@AllArgsConstructor
public class CategoryCountRow {
  private UUID categoryId;
  private long count;
}
