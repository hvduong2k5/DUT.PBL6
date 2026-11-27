-- V17: MVP — Bỏ yêu cầu scan_status = 'CLEAN' khỏi điều kiện public ảnh và partial unique index COVER
-- Trong phạm vi MVP chưa triển khai ClamAV/Antivirus Worker nên chỉ kiểm duyệt bằng media_status = 'APPROVED'
-- Catalog Service — om_catalog_db

DROP INDEX IF EXISTS idx_product_images_approved_listing;
CREATE INDEX IF NOT EXISTS idx_product_images_approved_listing
    ON product_images(product_id, role, position)
    WHERE media_status = 'APPROVED';

DROP INDEX IF EXISTS uq_product_images_approved_cover;
CREATE UNIQUE INDEX IF NOT EXISTS uq_product_images_approved_cover
    ON product_images(product_id)
    WHERE role = 'COVER' AND media_status = 'APPROVED';
