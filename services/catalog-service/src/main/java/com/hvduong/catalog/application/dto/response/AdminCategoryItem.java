package com.hvduong.catalog.application.dto.response;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** Item danh mục trả về cho giao diện quản trị Admin CMS. */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminCategoryItem {
  private UUID categoryId;
  private UUID parentId;
  private String parentName;
  private String name;
  private String slug;
  private String description;
  private String imageUrl;
  private int position;
  private boolean isActive;
  private long productCount;
  private long childrenCount;
  private long version;
  private Instant createdAt;
  private Instant updatedAt;

  @Builder.Default private List<AdminCategoryItem> children = new ArrayList<>();
}
