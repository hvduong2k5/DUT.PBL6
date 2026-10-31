-- Isolated sandbox evidence; not Order Core inbox or a financial ledger.
CREATE TABLE IF NOT EXISTS receipts (
  receipt_id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  envelope TEXT NOT NULL,
  first_seen TEXT NOT NULL,
  last_seen TEXT NOT NULL,
  deliveries INTEGER NOT NULL DEFAULT 1 CHECK(deliveries > 0)
);
CREATE INDEX IF NOT EXISTS receipts_event ON receipts(event_id);
