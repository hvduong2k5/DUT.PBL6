package com.hvduong.catalog.application.dto.response;

import java.io.Serializable;
import java.util.List;
import java.util.UUID;

import com.fasterxml.jackson.annotation.JsonProperty;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/** DTO phản hồi danh sách danh mục công khai (Customer & Admin public tree). */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class CategoryListResponse implements Serializable {
  private static final long serialVersionUID = 1L;

  private List<CategoryItem> categories;

  @Data
  @Builder
  @NoArgsConstructor
  @AllArgsConstructor
  public static class CategoryItem implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("category_id")
    private UUID categoryId;

    @JsonProperty("parent_id")
    private UUID parentId;

    private String name;
    private String slug;
    private String description;

    @JsonProperty("image_url")
    private String imageUrl;
  }
}
