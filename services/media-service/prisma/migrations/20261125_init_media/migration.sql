CREATE TYPE media_purpose AS ENUM (
    'CATALOG_PRODUCT',
    'CATALOG_CATEGORY',
    'OCOP_CERTIFICATE',
    'SHIPPING_VIDEO',
    'B2B_DOCUMENT',
    'REFUND_EVIDENCE',
    'AVATAR'
);

CREATE TYPE media_status AS ENUM (
    'AWAITING_UPLOAD',
    'PROCESSING',
    'READY',
    'PROCESSING_FAILED',
    'ARCHIVED'
);

CREATE TYPE access_level AS ENUM (
    'PUBLIC_READ',
    'AUTHENTICATED_READ',
    'PRIVATE_RESTRICTED'
);

CREATE TABLE media_files (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_key        VARCHAR(1024) NOT NULL UNIQUE,
    bucket_name       VARCHAR(128) NOT NULL,
    file_name         VARCHAR(500) NOT NULL,
    media_type        VARCHAR(128) NOT NULL,
    size_bytes        BIGINT NOT NULL,
    sha256_checksum   VARCHAR(64) NULL,
    purpose           media_purpose NOT NULL,
    access_level      access_level NOT NULL DEFAULT 'PRIVATE_RESTRICTED',
    status            media_status NOT NULL DEFAULT 'AWAITING_UPLOAD',
    uploader_id       VARCHAR(128) NOT NULL,
    bound_entity_type VARCHAR(64) NULL,
    bound_entity_id   VARCHAR(128) NULL,
    variants_meta     JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE upload_sessions (
    id                VARCHAR(64) PRIMARY KEY,
    file_id           UUID NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    upload_url        TEXT NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    status            VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_files_purpose ON media_files(purpose);
CREATE INDEX idx_media_files_bound ON media_files(bound_entity_type, bound_entity_id);
CREATE INDEX idx_media_files_status ON media_files(status);
