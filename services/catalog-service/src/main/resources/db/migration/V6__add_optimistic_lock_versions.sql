-- V6: Harden product image moderation and add missing optimistic-lock revisions.
-- V1-V5 remain immutable; this migration combines both V6 changes before rollout.

DROP INDEX IF EXISTS uq_product_images_cover;
CREATE UNIQUE INDEX IF NOT EXISTS uq_product_images_approved_cover
    ON product_images(product_id)
    WHERE role = 'COVER'
      AND media_status = 'APPROVED';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_product_images_position'
          AND conrelid = 'product_images'::regclass
    ) THEN
        ALTER TABLE product_images
            ADD CONSTRAINT chk_product_images_position CHECK (position >= 0);
    END IF;
END $$;

ALTER TABLE product_images
    ADD COLUMN IF NOT EXISTS scanned_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS rejected_reason TEXT NULL,
    ADD COLUMN IF NOT EXISTS uploaded_by VARCHAR(100) NULL,
    ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE channel_prices
    ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ck_categories_version_non_negative' AND conrelid = 'categories'::regclass) THEN
        ALTER TABLE categories ADD CONSTRAINT ck_categories_version_non_negative CHECK (version >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ck_channel_prices_version_non_negative' AND conrelid = 'channel_prices'::regclass) THEN
        ALTER TABLE channel_prices ADD CONSTRAINT ck_channel_prices_version_non_negative CHECK (version >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ck_product_images_version_non_negative' AND conrelid = 'product_images'::regclass) THEN
        ALTER TABLE product_images ADD CONSTRAINT ck_product_images_version_non_negative CHECK (version >= 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_product_images_approved_listing
    ON product_images(product_id, role, position)
    WHERE media_status = 'APPROVED';

COMMENT ON COLUMN product_images.scanned_at IS 'Thời điểm hoàn tất kiểm tra bảo mật nội dung ảnh';
COMMENT ON COLUMN product_images.rejected_reason IS 'Lý do từ chối ảnh, nếu có';
COMMENT ON COLUMN product_images.uploaded_by IS 'Định danh người dùng hoặc service tải ảnh lên';
COMMENT ON COLUMN categories.version IS 'Optimistic-lock revision; incremented on each successful update';
COMMENT ON COLUMN channel_prices.version IS 'Optimistic-lock revision; incremented on each successful update';
COMMENT ON COLUMN product_images.version IS 'Optimistic-lock revision; incremented on each successful update';
