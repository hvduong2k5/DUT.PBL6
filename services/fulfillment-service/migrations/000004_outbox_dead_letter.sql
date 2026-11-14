ALTER TABLE fulfillment.outbox
    ADD COLUMN consecutive_failures integer NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    ADD COLUMN dead_lettered_at timestamptz,
    ADD COLUMN last_error_code text;

-- Quarantine already exhausted deliveries while retaining their envelope and ordering.
UPDATE fulfillment.outbox
SET consecutive_failures = attempts,
    dead_lettered_at = clock_timestamp(),
    last_error_code = 'KAFKA_PUBLISH_FAILED',
    lease_until = NULL,
    lease_epoch = lease_epoch + 1
WHERE published_at IS NULL AND attempts >= 10;
