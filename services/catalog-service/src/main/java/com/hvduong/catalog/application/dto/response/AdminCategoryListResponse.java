package com.hvduong.catalog.application.dto.response;

import java.util.List;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Response danh sách danh mục cho Admin CMS (cả dạng cây và dạng phẳng). */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminCategoryListResponse {
  private List<AdminCategoryItem> categories;
}
