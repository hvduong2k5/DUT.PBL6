ALTER TABLE goods_receipts
    ADD COLUMN IF NOT EXISTS variances JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(variances) = 'array');
