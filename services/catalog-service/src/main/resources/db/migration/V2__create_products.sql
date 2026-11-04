-- V2: Tạo bảng products (sản phẩm mẹ — Parent Product)
-- Catalog Service — om_catalog_db

CREATE TYPE approval_status AS ENUM ('DRAFT', 'PENDING', 'APPROVED', 'REJECTED');
CREATE TYPE listing_status  AS ENUM ('ACTIVE', 'SUSPENDED', 'ARCHIVED');

CREATE TABLE IF NOT EXISTS products (
    id                  UUID            NOT NULL DEFAULT gen_random_uuid(),
    category_id         UUID            NOT NULL,
    name                VARCHAR(500)    NOT NULL,
    slug                VARCHAR(500)    NOT NULL,
    description         TEXT            NULL,
    ingredients         TEXT            NULL,       -- Thành phần nguyên liệu (food information)
    storage_guide       TEXT            NULL,       -- Hướng dẫn bảo quản
    ocop_star           SMALLINT        NULL CHECK (ocop_star BETWEEN 3 AND 5),
    ocop_certificate_no VARCHAR(100)    NULL,
    story               TEXT            NULL,       -- Câu chuyện thương hiệu / sản phẩm
    approval_status     approval_status NOT NULL DEFAULT 'DRAFT',
    listing_status      listing_status  NOT NULL DEFAULT 'ACTIVE',
    approved_by         VARCHAR(100)    NULL,
    approved_at         TIMESTAMPTZ     NULL,
    version             BIGINT          NOT NULL DEFAULT 0,  -- Optimistic locking
    created_by          VARCHAR(100)    NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_products PRIMARY KEY (id),
    CONSTRAINT fk_products_category FOREIGN KEY (category_id)
        REFERENCES categories(id),
    CONSTRAINT uq_products_slug UNIQUE (slug)
);

CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_approval_listing ON products(approval_status, listing_status);
CREATE INDEX idx_products_ocop_star ON products(ocop_star) WHERE ocop_star IS NOT NULL;
CREATE INDEX idx_products_created_at ON products(created_at DESC);

COMMENT ON TABLE products IS 'Sản phẩm mẹ (Parent Product) — Aggregate Root của Catalog domain';
COMMENT ON COLUMN products.slug IS 'URL-friendly slug, unique toàn catalog — phục vụ SEO';
COMMENT ON COLUMN products.version IS 'Optimistic locking để tránh concurrent update conflict';
COMMENT ON COLUMN products.approval_status IS 'Trạng thái phê duyệt: DRAFT → PENDING → APPROVED | REJECTED';
COMMENT ON COLUMN products.listing_status IS 'Trạng thái kinh doanh: ACTIVE | SUSPENDED | ARCHIVED';
