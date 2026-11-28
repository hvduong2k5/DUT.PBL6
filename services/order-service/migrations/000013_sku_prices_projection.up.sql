CREATE TABLE IF NOT EXISTS sku_prices_projection (
    sku_code        VARCHAR(100) NOT NULL,
    channel         VARCHAR(50)  NOT NULL,
    product_name    VARCHAR(500) NOT NULL,
    unit_price      BIGINT       NOT NULL CHECK (unit_price >= 0),
    weight_grams    BIGINT       NOT NULL DEFAULT 0,
    packaging_type  VARCHAR(100) NOT NULL DEFAULT '',
    is_saleable     BOOLEAN      NOT NULL DEFAULT TRUE,
    price_version   BIGINT       NOT NULL DEFAULT 0,
    effective_from  TIMESTAMPTZ  NOT NULL,
    effective_to    TIMESTAMPTZ  NULL,
    synced_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_sku_prices_projection PRIMARY KEY (sku_code, channel)
);

CREATE INDEX IF NOT EXISTS idx_sku_prices_projection_lookup
    ON sku_prices_projection(channel, is_saleable);
