-- V18: Tách độc lập bảng sku_stock_projection khỏi khóa ngoại product_variants(sku_code)
-- Catalog chỉ nhận event tồn kho từ Inventory để lưu vào sku_stock_projection;
-- việc liên kết với ProductVariant được cấu hình qua inventory_sku_code (mặc định khớp theo sku_code),
-- và xóa Product/Variant trong Catalog không xóa dữ liệu tồn kho đã đồng bộ.

ALTER TABLE sku_stock_projection
    DROP CONSTRAINT IF EXISTS sku_stock_projection_sku_code_fkey;

ALTER TABLE product_variants
    ADD COLUMN IF NOT EXISTS inventory_sku_code VARCHAR(100) NULL;

CREATE INDEX IF NOT EXISTS idx_product_variants_inventory_sku_code
    ON product_variants(inventory_sku_code)
    WHERE inventory_sku_code IS NOT NULL;

COMMENT ON COLUMN product_variants.inventory_sku_code IS
    'Mã SKU bên Inventory (sku_stock_projection.sku_code) để liên kết tồn kho; NULL nghĩa là khớp mặc định theo sku_code';
