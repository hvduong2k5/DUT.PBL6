-- Additive: legacy sandbox obligations stay readable and are not silently migrated.
CREATE TABLE payment_attempts (
 id uuid PRIMARY KEY, order_id uuid NOT NULL UNIQUE REFERENCES orders(id),
 provider text NOT NULL, environment text NOT NULL CHECK(environment='sandbox'),
 merchant_invoice text NOT NULL, merchant_id text NOT NULL,
 amount bigint NOT NULL CHECK(amount>0), currency text NOT NULL CHECK(currency='VND'),
 expires_at timestamptz NOT NULL, provider_order_id text,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(provider,environment,merchant_invoice)
);
CREATE TABLE sepay_notifications (
 id uuid PRIMARY KEY, payload_hash text NOT NULL UNIQUE, payload_cipher bytea NOT NULL,
 status text NOT NULL DEFAULT 'PENDING' CHECK(status IN ('PENDING','APPLIED','MANUAL_REVIEW')),
 reason text, received_at timestamptz NOT NULL DEFAULT clock_timestamp(), applied_at timestamptz
);
CREATE INDEX sepay_notification_pending ON sepay_notifications(received_at) WHERE status='PENDING';
CREATE TABLE sepay_scan (
 provider text PRIMARY KEY, page integer NOT NULL DEFAULT 1 CHECK(page>0),
 next_run timestamptz NOT NULL DEFAULT clock_timestamp(), lease_until timestamptz,
 last_success timestamptz, error_code text
);
CREATE TABLE manual_refunds (
 id uuid PRIMARY KEY, receipt_id uuid NOT NULL REFERENCES receipts(id),
 order_id uuid REFERENCES orders(id), request_key text NOT NULL UNIQUE, request_hash text NOT NULL,
 amount bigint NOT NULL CHECK(amount>0), status text NOT NULL DEFAULT 'APPROVED'
 CHECK(status IN ('APPROVED','SUBMITTED','UNKNOWN','SUCCEEDED')),
 approved_by uuid NOT NULL, reason text NOT NULL,
 payee_cipher bytea NOT NULL, claimed_by uuid, verified_by uuid,
 evidence_cipher bytea, evidence_hash text, settlement_key text UNIQUE,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK(status='APPROVED' OR claimed_by IS NOT NULL),
 CHECK(status<>'SUCCEEDED' OR (verified_by IS NOT NULL AND verified_by<>claimed_by AND evidence_hash IS NOT NULL))
);
CREATE INDEX manual_refunds_receipt ON manual_refunds(receipt_id);
CREATE INDEX manual_refunds_pending ON manual_refunds(created_at) WHERE status<>'SUCCEEDED';
ALTER TABLE cancellation_requests ADD COLUMN decided_by uuid, ADD COLUMN decision_hash text,
 ADD COLUMN decision_reason text, ADD COLUMN decided_at timestamptz;
CREATE UNIQUE INDEX mvp_cancel_one_approval ON cancellation_requests(order_id) WHERE status='APPROVED';
