package com.hvduong.catalog.application.dto.response;

import lombok.Builder;
import lombok.Data;

import java.util.List;
import java.util.UUID;

/** Response danh mục sản phẩm. */
@Data
@Builder
public class CategoryResponse {
    private UUID categoryId;
    private UUID parentId;
    private String name;
    private String slug;
    private String description;
    private String imageUrl;
    private int position;
    private List<CategoryResponse> children;
}
