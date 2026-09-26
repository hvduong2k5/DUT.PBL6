package com.hvduong.catalog.domain.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.Instant;
import java.util.UUID;

/**
 * Entity ảnh sản phẩm — ánh xạ bảng {@code product_images}.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProductImage {

    private UUID id;
    private UUID productId;
    private String objectKey;         // MinIO/S3 object key
    private String altText;
    private String role;              // COVER | GALLERY | DETAIL
    private int position;
    private String mediaStatus;       // PENDING | APPROVED | REJECTED
    private Instant createdAt;
    private Instant updatedAt;
}
