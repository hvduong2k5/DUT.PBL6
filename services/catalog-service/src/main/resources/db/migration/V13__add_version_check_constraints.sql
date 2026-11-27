-- V13: Bổ sung CHECK constraint version >= 0 cho products và product_variants
-- Đảm bảo tính toàn vẹn cho khóa lạc quan (Optimistic Locking)

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_products_version_non_negative'
          AND conrelid = 'products'::regclass
    ) THEN
        ALTER TABLE products
            ADD CONSTRAINT chk_products_version_non_negative
            CHECK (version >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_variants_version_non_negative'
          AND conrelid = 'product_variants'::regclass
    ) THEN
        ALTER TABLE product_variants
            ADD CONSTRAINT chk_variants_version_non_negative
            CHECK (version >= 0);
    END IF;
END $$;

COMMENT ON CONSTRAINT chk_products_version_non_negative ON products IS 'Đảm bảo version của product luôn không âm';
COMMENT ON CONSTRAINT chk_variants_version_non_negative ON product_variants IS 'Đảm bảo version của variant luôn không âm';
