-- F-42 slice 3: local evidence vault. The actual evidence files (bank
-- statements, receipts) behind payment case decisions and manual refunds
-- live in this service's own database, replacing the never-deployed
-- external finance vault. An upload is ticketed first (the ticket row in
-- payment_evidence_uploads pins the declared SHA-256, media type and
-- size); the bytes arrive through a short-lived capability-signed URL
-- and are stored only when their length and SHA-256 match the ticket
-- exactly, otherwise the object is REJECTED. Downloads are minted per
-- read, again as short-lived capability-signed URLs, and only for CLEAN
-- objects. Files are capped at 10 MiB by the upload contract.
CREATE TABLE payment_evidence_objects (
 evidence_id uuid PRIMARY KEY,
 stable_key text NOT NULL UNIQUE,
 resource_type text NOT NULL CHECK(resource_type IN ('REFUND','RECONCILIATION_CASE')),
 resource_id uuid NOT NULL,
 uploader uuid NOT NULL,
 sha256 text NOT NULL,
 media_type text NOT NULL CHECK(media_type IN ('application/pdf','image/png','image/jpeg')),
 byte_size bigint NOT NULL CHECK(byte_size BETWEEN 1 AND 10485760),
 content bytea,
 state text NOT NULL DEFAULT 'AWAITING_UPLOAD' CHECK(state IN ('AWAITING_UPLOAD','CLEAN','REJECTED')),
 ticket_expires_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 uploaded_at timestamptz
);
