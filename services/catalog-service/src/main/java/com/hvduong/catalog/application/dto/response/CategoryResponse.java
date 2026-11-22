package com.hvduong.catalog.application.dto.response;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Response danh mục sản phẩm. */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class CategoryResponse implements java.io.Serializable {
  private static final long serialVersionUID = 1L;
  private UUID categoryId;
  private UUID parentId;
  private String name;
  private String slug;
  private String description;
  private String imageUrl;
  private int position;
  private boolean isActive;
  private long version;
  private Instant createdAt;
  private Instant updatedAt;
  private List<CategoryResponse> children;
}
