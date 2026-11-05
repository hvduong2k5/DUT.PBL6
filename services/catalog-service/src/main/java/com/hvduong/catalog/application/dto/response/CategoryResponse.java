package com.hvduong.catalog.application.dto.response;

import java.util.List;
import java.util.UUID;

import lombok.Builder;
import lombok.Data;

/** Response danh mục sản phẩm. */
@Data
@Builder
public class CategoryResponse implements java.io.Serializable {
  private static final long serialVersionUID = 1L;
  private UUID categoryId;
  private UUID parentId;
  private String name;
  private String slug;
  private String description;
  private String imageUrl;
  private int position;
  private List<CategoryResponse> children;
}
