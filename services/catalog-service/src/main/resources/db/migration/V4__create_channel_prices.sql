-- V4: Tạo bảng channel_prices (Giá theo kênh bán)
-- Catalog Service — om_catalog_db

CREATE TYPE sales_channel AS ENUM (
    'D2C_WEB',
    'MOBILE_APP',
    'B2B_WHOLESALE',
    'POS_QUAY',
    'SHOPEE',
    'TIKTOK'
);

CREATE TYPE price_status AS ENUM ('ACTIVE', 'SUPERSEDED', 'EXPIRED');

CREATE TABLE IF NOT EXISTS channel_prices (
    id              UUID            NOT NULL DEFAULT gen_random_uuid(),
    variant_id      UUID            NOT NULL,
    channel         sales_channel   NOT NULL,
    currency_code   CHAR(3)         NOT NULL DEFAULT 'VND',
    -- Giá lưu theo đơn vị nhỏ nhất (units) — BigInt safe, pattern từ proto Money
    amount_units    BIGINT          NOT NULL CHECK (amount_units >= 0),
    amount_nanos    INT             NOT NULL DEFAULT 0,
    status          price_status    NOT NULL DEFAULT 'ACTIVE',
    effective_from  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    effective_to    TIMESTAMPTZ     NULL,           -- NULL = không có ngày hết hạn
    created_by      VARCHAR(100)    NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_channel_prices PRIMARY KEY (id),
    CONSTRAINT fk_prices_variant FOREIGN KEY (variant_id)
        REFERENCES product_variants(id) ON DELETE CASCADE
);

-- Chỉ cho phép 1 giá ACTIVE tại mỗi thời điểm cho mỗi (variant, channel)
CREATE UNIQUE INDEX uq_channel_prices_active
    ON channel_prices(variant_id, channel)
    WHERE status = 'ACTIVE';

CREATE INDEX idx_channel_prices_variant_channel ON channel_prices(variant_id, channel);
CREATE INDEX idx_channel_prices_status ON channel_prices(status);
CREATE INDEX idx_channel_prices_effective ON channel_prices(effective_from, effective_to);

COMMENT ON TABLE channel_prices IS 'Giá niêm yết theo kênh bán và hiệu lực thời gian';
COMMENT ON COLUMN channel_prices.amount_units IS 'Phần nguyên của giá (đồng VND). Pattern từ google.type.Money / proto Money';
COMMENT ON COLUMN channel_prices.amount_nanos IS 'Phần lẻ nano (10^-9), VND luôn = 0';
COMMENT ON COLUMN channel_prices.status IS 'ACTIVE = đang có hiệu lực; SUPERSEDED = đã bị thay thế bởi giá mới';
