CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS suppliers (
    id UUID PRIMARY KEY,
    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(240) NOT NULL,
    supplier_type VARCHAR(32) NOT NULL CHECK (supplier_type IN ('FARMER', 'COOPERATIVE', 'OCOP_BUSINESS', 'COMPANY')),
    contact_name VARCHAR(160) NOT NULL DEFAULT '',
    phone VARCHAR(40) NOT NULL DEFAULT '',
    email VARCHAR(254) NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS suppliers_status_name_idx ON suppliers (status, name);

CREATE TABLE IF NOT EXISTS raw_materials (
    id UUID PRIMARY KEY,
    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(240) NOT NULL,
    base_unit VARCHAR(24) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS raw_materials_status_name_idx ON raw_materials (status, name);

CREATE TABLE IF NOT EXISTS supplier_material_offers (
    id UUID PRIMARY KEY,
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    material_id UUID NOT NULL REFERENCES raw_materials(id),
    supplier_sku VARCHAR(100) NOT NULL DEFAULT '',
    unit VARCHAR(24) NOT NULL,
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor > 0),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    minimum_quantity NUMERIC(18, 6) NOT NULL CHECK (minimum_quantity > 0),
    lead_time_days INTEGER NOT NULL DEFAULT 0 CHECK (lead_time_days >= 0),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT supplier_material_offers_supplier_material_unique UNIQUE (supplier_id, material_id)
);

CREATE INDEX IF NOT EXISTS supplier_material_offers_material_idx ON supplier_material_offers (material_id, active);
