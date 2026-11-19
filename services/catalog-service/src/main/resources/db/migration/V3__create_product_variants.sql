-- V3: Tạo bảng product_variants (Biến thể / SKU)
-- Catalog Service — om_catalog_db

CREATE TYPE variant_listing_status AS ENUM ('ACTIVE', 'SUSPENDED', 'ARCHIVED');
CREATE TYPE packaging_type AS ENUM (
    'HOP_GIAY_KRAFT',
    'HOP_THIEC',
    'GOI_HUT_CHAN_KHONG',
    'HOP_QUA_TANG',
    'DANG_ROI'
);

CREATE TABLE IF NOT EXISTS product_variants (
    id                  UUID                    NOT NULL DEFAULT gen_random_uuid(),
    product_id          UUID                    NOT NULL,
    sku_code            VARCHAR(100)            NOT NULL,   -- Unique toàn catalog: e.g. "MX-GION-500G"
    variant_name        VARCHAR(500)            NOT NULL,
    weight_value        NUMERIC(10, 3)          NOT NULL,   -- Khối lượng tịnh
    weight_unit         VARCHAR(10)             NOT NULL DEFAULT 'GRAM',  -- GRAM | KILOGRAM
    flavor              VARCHAR(200)            NULL,
    packaging_type      packaging_type          NOT NULL DEFAULT 'GOI_HUT_CHAN_KHONG',
    shelf_life_days     INT                     NULL,       -- Hạn sử dụng (số ngày từ NSX)
    listing_status      variant_listing_status  NOT NULL DEFAULT 'ACTIVE',
    version             BIGINT                  NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ             NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_product_variants PRIMARY KEY (id),
    CONSTRAINT fk_variants_product FOREIGN KEY (product_id)
        REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT uq_variants_sku_code UNIQUE (sku_code)
);

CREATE INDEX idx_variants_product_id ON product_variants(product_id);
CREATE INDEX idx_variants_listing_status ON product_variants(listing_status);
CREATE INDEX idx_variants_sku_code ON product_variants(sku_code);

COMMENT ON TABLE product_variants IS 'Biến thể SKU — đơn vị khách chọn để mua';
COMMENT ON COLUMN product_variants.sku_code IS 'Mã SKU duy nhất toàn catalog, chuẩn hóa: MX-{LOAI}-{KHOILUONG}{DONVI}';
COMMENT ON COLUMN product_variants.weight_value IS 'Khối lượng tịnh bảo đảm — numeric để tránh sai số floating point';
COMMENT ON COLUMN product_variants.shelf_life_days IS 'Hạn sử dụng theo nhãn sản phẩm (catalog label), NOT batch actual HSD';
