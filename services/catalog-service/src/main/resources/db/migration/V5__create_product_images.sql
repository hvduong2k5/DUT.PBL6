-- V5: Tạo bảng product_images (Ảnh sản phẩm — S3 object references)
-- Catalog Service — om_catalog_db

CREATE TYPE image_role AS ENUM ('COVER', 'GALLERY', 'THUMBNAIL');
CREATE TYPE media_status AS ENUM ('PENDING_SCAN', 'APPROVED', 'REJECTED');

CREATE TABLE IF NOT EXISTS product_images (
    id              UUID            NOT NULL DEFAULT gen_random_uuid(),
    product_id      UUID            NOT NULL,
    object_key      VARCHAR(1024)   NOT NULL,   -- MinIO/S3 object key (KHÔNG lưu public URL)
    alt_text        VARCHAR(500)    NULL,
    role            image_role      NOT NULL DEFAULT 'GALLERY',
    position        INT             NOT NULL DEFAULT 0,
    media_status    media_status    NOT NULL DEFAULT 'PENDING_SCAN',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_product_images PRIMARY KEY (id),
    CONSTRAINT fk_images_product FOREIGN KEY (product_id)
        REFERENCES products(id) ON DELETE CASCADE
);

CREATE INDEX idx_product_images_product_id ON product_images(product_id, position);
CREATE INDEX idx_product_images_role ON product_images(product_id, role);

-- Đảm bảo mỗi sản phẩm chỉ có 1 ảnh COVER
CREATE UNIQUE INDEX uq_product_images_cover
    ON product_images(product_id)
    WHERE role = 'COVER';

COMMENT ON TABLE product_images IS 'Metadata ảnh sản phẩm — chỉ lưu object_key, không lưu URL trực tiếp';
COMMENT ON COLUMN product_images.object_key IS 'MinIO/S3 object key để generate presigned URL — KHÔNG public URL';
COMMENT ON COLUMN product_images.role IS 'COVER = ảnh đại diện chính; GALLERY = ảnh phụ; THUMBNAIL = ảnh thu nhỏ';
