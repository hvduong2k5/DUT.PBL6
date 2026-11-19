-- Catalog Service — om_catalog_db
-- Durable read model replicated from versioned Inventory events; not source-of-truth stock.

CREATE TABLE sku_stock_projection (
    sku_code           VARCHAR(100) PRIMARY KEY
                       REFERENCES product_variants(sku_code)
                       ON DELETE CASCADE,
    sellable_quantity  BIGINT NULL
                       CHECK (sellable_quantity IS NULL OR sellable_quantity >= 0),
    inventory_version  BIGINT NOT NULL DEFAULT -1
                       CHECK (inventory_version >= -1),
    source_updated_at  TIMESTAMPTZ NULL,
    synced_at          TIMESTAMPTZ NULL
);

CREATE INDEX idx_stock_projection_sellable_quantity
    ON sku_stock_projection(sellable_quantity)
    WHERE sellable_quantity IS NOT NULL;

COMMENT ON TABLE sku_stock_projection IS
    'Read model tồn khả dụng nhận từ Inventory Service; không phải nguồn tồn kho gốc';
COMMENT ON COLUMN sku_stock_projection.sellable_quantity IS
    'Số lượng được phép bán theo quy tắc Inventory; NULL nghĩa là chưa đồng bộ/chưa xác định';
COMMENT ON COLUMN sku_stock_projection.inventory_version IS
    'Version tăng đơn điệu theo SKU ở Inventory, dùng để bỏ qua event cũ';
COMMENT ON COLUMN sku_stock_projection.source_updated_at IS
    'Thời điểm cập nhật nguồn do Inventory cung cấp';
COMMENT ON COLUMN sku_stock_projection.synced_at IS
    'Thời điểm Catalog áp dụng snapshot tồn kho';
