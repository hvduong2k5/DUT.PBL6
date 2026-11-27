-- V16: Tách bạch kết quả quét an toàn mã độc (scan_status) và kiểm duyệt nội dung (media_status)
-- Bổ sung reviewed_by, reviewed_at và cập nhật index hiển thị ảnh công khai
-- Catalog Service — om_catalog_db

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'media_scan_status') THEN
        CREATE TYPE media_scan_status AS ENUM ('PENDING', 'CLEAN', 'INFECTED', 'FAILED');
    END IF;
END $$;

ALTER TYPE media_status ADD VALUE IF NOT EXISTS 'PENDING_REVIEW';

ALTER TABLE product_images
    ADD COLUMN IF NOT EXISTS scan_status media_scan_status NOT NULL DEFAULT 'PENDING',
    ADD COLUMN IF NOT EXISTS reviewed_by VARCHAR(100) NULL,
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ NULL;

-- Backfill dữ liệu hiện tại: Các ảnh đã APPROVED trước đó được đánh dấu là CLEAN và cập nhật audit timestamp
UPDATE product_images
SET scan_status = 'CLEAN',
    scanned_at = COALESCE(scanned_at, created_at),
    reviewed_at = COALESCE(reviewed_at, updated_at)
WHERE media_status = 'APPROVED';

-- Index phục vụ truy vấn ảnh đã duyệt và quét sạch
DROP INDEX IF EXISTS idx_product_images_approved_listing;
CREATE INDEX IF NOT EXISTS idx_product_images_approved_listing
    ON product_images(product_id, role, position)
    WHERE media_status = 'APPROVED' AND scan_status = 'CLEAN';

-- Cập nhật partial unique index cho COVER: đảm bảo chỉ 1 COVER được APPROVED và CLEAN
DROP INDEX IF EXISTS uq_product_images_approved_cover;
CREATE UNIQUE INDEX IF NOT EXISTS uq_product_images_approved_cover
    ON product_images(product_id)
    WHERE role = 'COVER' AND media_status = 'APPROVED' AND scan_status = 'CLEAN';

COMMENT ON COLUMN product_images.scan_status IS 'Trạng thái kiểm tra kỹ thuật/mã độc từ Media Service: PENDING, CLEAN, INFECTED, FAILED';
COMMENT ON COLUMN product_images.reviewed_by IS 'Định danh nhân viên/Moderator phê duyệt hoặc từ chối ảnh';
COMMENT ON COLUMN product_images.reviewed_at IS 'Thời điểm phê duyệt hoặc từ chối ảnh';
