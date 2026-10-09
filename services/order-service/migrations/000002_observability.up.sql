ALTER TABLE outbox ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT clock_timestamp();
UPDATE outbox SET created_at=(payload->>'time')::timestamptz WHERE payload ? 'time';
ALTER TABLE inbox ADD COLUMN IF NOT EXISTS received_at timestamptz NOT NULL DEFAULT clock_timestamp();
