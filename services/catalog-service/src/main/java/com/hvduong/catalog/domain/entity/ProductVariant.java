package com.hvduong.catalog.domain.entity;

import com.hvduong.catalog.common.enums.ListingStatus;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.math.BigDecimal;
import java.time.Instant;
import java.util.List;
import java.util.UUID;

/**
 * Entity biến thể sản phẩm (SKU) — ánh xạ bảng {@code product_variants}.
 * <p>
 * SKU là đơn vị khách chọn để mua và là cấp quản lý giá.
 * Invariant: sku_code phải unique toàn catalog.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProductVariant {

    private UUID id;
    private UUID productId;
    private String skuCode;           // Unique: e.g. "MX-GION-500G"
    private String variantName;       // e.g. "Kẹo Mè Xửng Giòn Huế - Hộp 500g"
    private BigDecimal weightValue;   // Khối lượng tịnh
    private String weightUnit;        // GRAM | KILOGRAM
    private String flavor;            // Hương vị (tùy chọn)
    private String packagingType;     // HOP_GIAY_KRAFT, GOI_HUT_CHAN_KHONG, ...
    private Integer shelfLifeDays;    // Hạn sử dụng (số ngày từ NSX)
    private ListingStatus listingStatus;
    private long version;
    private Instant createdAt;
    private Instant updatedAt;

    // ── Non-persisted ─────────────────────────────────────────────────────────
    private List<ChannelPrice> prices;  // Load khi cần giá theo kênh
}
