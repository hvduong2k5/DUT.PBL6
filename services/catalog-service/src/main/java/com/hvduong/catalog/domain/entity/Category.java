package com.hvduong.catalog.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

/**
 * Entity danh mục sản phẩm — ánh xạ bảng {@code categories}.
 * Hỗ trợ cây phân cấp không giới hạn qua {@code parentId}.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class Category {

    private UUID id;
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

    // ── Non-persisted ─────────────────────────────────────────────────────────
    private List<Category> children;
}
