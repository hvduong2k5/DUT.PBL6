ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS legal_identifier VARCHAR(120) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS suppliers_legal_identifier_unique_idx
    ON suppliers (upper(legal_identifier)) WHERE legal_identifier <> '';

ALTER TABLE raw_materials ADD COLUMN IF NOT EXISTS preferred_supplier_id UUID NULL REFERENCES suppliers(id);

ALTER TABLE supplier_material_offers
    ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    ADD COLUMN IF NOT EXISTS effective_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS effective_to TIMESTAMPTZ NULL;
UPDATE supplier_material_offers SET effective_from = created_at WHERE version = 1;
ALTER TABLE supplier_material_offers DROP CONSTRAINT IF EXISTS supplier_material_offers_supplier_material_unique;
CREATE UNIQUE INDEX IF NOT EXISTS supplier_material_offers_version_unique_idx
    ON supplier_material_offers (supplier_id, material_id, version);
CREATE INDEX IF NOT EXISTS supplier_material_offers_effective_idx
    ON supplier_material_offers (supplier_id, material_id, effective_from DESC, effective_to);
ALTER TABLE supplier_material_offers ADD CONSTRAINT supplier_material_offers_effective_range_check
    CHECK (effective_to IS NULL OR effective_from < effective_to);

CREATE TABLE IF NOT EXISTS supplier_contracts (
    id UUID PRIMARY KEY,
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    contract_number VARCHAR(100) NOT NULL,
    revision INTEGER NOT NULL CHECK (revision > 0),
    scope TEXT NOT NULL,
    payment_terms TEXT NOT NULL,
    document_file_id VARCHAR(160) NOT NULL DEFAULT '',
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to TIMESTAMPTZ NULL,
    status VARCHAR(16) NOT NULL CHECK (status IN ('DRAFT','ACTIVE','EXPIRED','TERMINATED')),
    created_by VARCHAR(160) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT supplier_contracts_revision_unique UNIQUE (supplier_id, contract_number, revision),
    CONSTRAINT supplier_contracts_effective_range_check CHECK (effective_to IS NULL OR effective_from < effective_to)
);
CREATE INDEX IF NOT EXISTS supplier_contracts_supplier_effective_idx
    ON supplier_contracts (supplier_id, status, effective_from DESC);

CREATE TABLE IF NOT EXISTS supplier_contract_status_history (
    id UUID PRIMARY KEY,
    supplier_contract_id UUID NOT NULL REFERENCES supplier_contracts(id) ON DELETE CASCADE,
    from_status VARCHAR(16) NULL,
    to_status VARCHAR(16) NOT NULL,
    action VARCHAR(24) NOT NULL,
    actor_id VARCHAR(160) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
