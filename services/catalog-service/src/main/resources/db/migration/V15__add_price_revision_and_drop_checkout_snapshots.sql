-- V15: Add price_revision to product_variants and drop temporary checkout_snapshots
-- Catalog Service — om_catalog_db

ALTER TABLE product_variants
ADD COLUMN IF NOT EXISTS price_revision BIGINT NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'ck_product_variants_price_revision_non_negative' 
          AND conrelid = 'product_variants'::regclass
    ) THEN
        ALTER TABLE product_variants 
            ADD CONSTRAINT ck_product_variants_price_revision_non_negative CHECK (price_revision >= 0);
    END IF;
END $$;

COMMENT ON COLUMN product_variants.price_revision IS 'Phiên bản giá tăng đơn điệu theo SKU mỗi khi có thay đổi giá ở bất kỳ channel nào';

-- Bỏ bảng checkout_snapshots tạm thời (Order Service tự sở hữu snapshot đơn hàng bất biến)
DROP TABLE IF EXISTS checkout_snapshots;
