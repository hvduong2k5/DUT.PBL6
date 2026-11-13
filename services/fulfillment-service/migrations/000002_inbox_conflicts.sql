BEGIN;
-- Keep the original event immutable; conflicting reuses have a separate ledger.
CREATE TABLE fulfillment.inbox_conflicts (
 source text NOT NULL,
 event_id text NOT NULL,
 payload_hash text NOT NULL,
 payload bytea NOT NULL,
 received_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(source,event_id,payload_hash)
);
COMMIT;
