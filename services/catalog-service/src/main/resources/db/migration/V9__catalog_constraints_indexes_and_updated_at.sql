-- Catalog Service — om_catalog_db
-- Integrity constraints, public-catalog indexes and database-managed updated_at.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_categories_position'
          AND conrelid = 'categories'::regclass
    ) THEN
        ALTER TABLE categories
            ADD CONSTRAINT chk_categories_position CHECK (position >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_categories_not_own_parent'
          AND conrelid = 'categories'::regclass
    ) THEN
        ALTER TABLE categories
            ADD CONSTRAINT chk_categories_not_own_parent
            CHECK (parent_id IS NULL OR parent_id <> id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_variant_weight_positive'
          AND conrelid = 'product_variants'::regclass
    ) THEN
        ALTER TABLE product_variants
            ADD CONSTRAINT chk_variant_weight_positive CHECK (weight_value > 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_variant_shelf_life_positive'
          AND conrelid = 'product_variants'::regclass
    ) THEN
        ALTER TABLE product_variants
            ADD CONSTRAINT chk_variant_shelf_life_positive
            CHECK (shelf_life_days IS NULL OR shelf_life_days > 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_variant_weight_unit'
          AND conrelid = 'product_variants'::regclass
    ) THEN
        ALTER TABLE product_variants
            ADD CONSTRAINT chk_variant_weight_unit
            CHECK (weight_unit IN ('GRAM', 'KILOGRAM'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_price_effective_range'
          AND conrelid = 'channel_prices'::regclass
    ) THEN
        ALTER TABLE channel_prices
            ADD CONSTRAINT chk_price_effective_range
            CHECK (effective_to IS NULL OR effective_to > effective_from);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_price_nanos_range'
          AND conrelid = 'channel_prices'::regclass
    ) THEN
        ALTER TABLE channel_prices
            ADD CONSTRAINT chk_price_nanos_range
            CHECK (amount_nanos >= 0 AND amount_nanos < 1000000000);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_vnd_nanos_zero'
          AND conrelid = 'channel_prices'::regclass
    ) THEN
        ALTER TABLE channel_prices
            ADD CONSTRAINT chk_vnd_nanos_zero
            CHECK (currency_code <> 'VND' OR amount_nanos = 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_products_public_catalog
    ON products(category_id, created_at DESC)
    WHERE approval_status = 'APPROVED'
      AND listing_status = 'ACTIVE';

CREATE INDEX IF NOT EXISTS idx_variants_active_product
    ON product_variants(product_id)
    WHERE listing_status = 'ACTIVE';

CREATE OR REPLACE FUNCTION catalog_set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_categories_updated_at ON categories;
CREATE TRIGGER trg_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION catalog_set_updated_at();

DROP TRIGGER IF EXISTS trg_products_updated_at ON products;
CREATE TRIGGER trg_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION catalog_set_updated_at();

DROP TRIGGER IF EXISTS trg_product_variants_updated_at ON product_variants;
CREATE TRIGGER trg_product_variants_updated_at
    BEFORE UPDATE ON product_variants
    FOR EACH ROW EXECUTE FUNCTION catalog_set_updated_at();

DROP TRIGGER IF EXISTS trg_channel_prices_updated_at ON channel_prices;
CREATE TRIGGER trg_channel_prices_updated_at
    BEFORE UPDATE ON channel_prices
    FOR EACH ROW EXECUTE FUNCTION catalog_set_updated_at();

DROP TRIGGER IF EXISTS trg_product_images_updated_at ON product_images;
CREATE TRIGGER trg_product_images_updated_at
    BEFORE UPDATE ON product_images
    FOR EACH ROW EXECUTE FUNCTION catalog_set_updated_at();
