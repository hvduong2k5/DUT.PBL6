package com.hvduong.catalog.domain.entity;

import com.hvduong.catalog.common.enums.ApprovalStatus;
import com.hvduong.catalog.common.enums.ListingStatus;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

/**
 * Entity sản phẩm mẹ (Parent Product) — Aggregate Root của Catalog domain.
 * Ánh xạ bảng {@code products}.
 * <p>
 * Invariant:
 * - Public read chỉ trả Product có approvalStatus=APPROVED và listingStatus=ACTIVE.
 * - version dùng cho optimistic locking để tránh concurrent update conflict.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class Product {

    private UUID id;
    private UUID categoryId;
    private String name;
    private String slug;
    private String description;
    private String ingredients;       // Thành phần nguyên liệu
    private String storageGuide;      // Hướng dẫn bảo quản
    private Integer ocopStar;         // 3, 4, hoặc 5 sao OCOP
    private String ocopCertificateNo; // Số chứng nhận OCOP
    private String story;             // Câu chuyện thương hiệu

    private ApprovalStatus approvalStatus;
    private ListingStatus listingStatus;
    private String approvedBy;
    private Instant approvedAt;

    private long version;             // Optimistic locking
    private String createdBy;
    private Instant createdAt;
    private Instant updatedAt;

    // ── Non-persisted: load khi cần ───────────────────────────────────────────
    private List<ProductVariant> variants;
    private List<ProductImage> images;
    private String categoryName;      // Denormalized để tránh N+1 khi list
}
