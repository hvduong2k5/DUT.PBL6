-- V1: Tạo bảng categories (danh mục sản phẩm, hỗ trợ hierarchy)
-- Catalog Service — om_catalog_db

CREATE TABLE IF NOT EXISTS categories (
    id              UUID            NOT NULL DEFAULT gen_random_uuid(),
    parent_id       UUID            NULL,          -- null = root category
    name            VARCHAR(255)    NOT NULL,
    slug            VARCHAR(255)    NOT NULL,
    description     TEXT            NULL,
    image_url       VARCHAR(1024)   NULL,
    position        INT             NOT NULL DEFAULT 0,
    is_active       BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_categories PRIMARY KEY (id),
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id)
        REFERENCES categories(id) ON DELETE SET NULL,
    CONSTRAINT uq_categories_slug UNIQUE (slug)
);

CREATE INDEX idx_categories_parent_id ON categories(parent_id);
CREATE INDEX idx_categories_is_active ON categories(is_active);
CREATE INDEX idx_categories_position ON categories(parent_id, position);

COMMENT ON TABLE categories IS 'Danh mục sản phẩm OCOP — hỗ trợ cây phân cấp không giới hạn độ sâu';
COMMENT ON COLUMN categories.parent_id IS 'NULL = root category';
COMMENT ON COLUMN categories.slug IS 'URL-friendly identifier, unique toàn catalog';
COMMENT ON COLUMN categories.position IS 'Thứ tự hiển thị trong cùng cấp (parent)';
