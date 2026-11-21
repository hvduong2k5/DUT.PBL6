CREATE TABLE notification_requests (
 id uuid PRIMARY KEY,
 producer_id text NOT NULL,
 intent_key text NOT NULL UNIQUE,
 fingerprint text NOT NULL,
 command jsonb NOT NULL,
 status text NOT NULL CHECK (status IN ('WAITING_CONTEXT')),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 next_run_at timestamptz NOT NULL DEFAULT now(),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notification_requests_due ON notification_requests(next_run_at,id) WHERE status='WAITING_CONTEXT';
CREATE INDEX notification_requests_owner ON notification_requests(producer_id,created_at,id);
CREATE TABLE idempotency_records (
 producer_id text NOT NULL,
 command_kind text NOT NULL DEFAULT 'INTAKE',
 key text NOT NULL,
 fingerprint text NOT NULL,
 request_id uuid NOT NULL REFERENCES notification_requests(id),
 PRIMARY KEY(producer_id,command_kind,key)
);
CREATE TABLE inbox_events (
 source text NOT NULL,
 event_id text NOT NULL,
 fingerprint text NOT NULL,
 disposition text NOT NULL,
 PRIMARY KEY(source,event_id)
);
CREATE TABLE event_conflicts (
 id bigserial PRIMARY KEY,
 producer_id text NOT NULL,
 identity_hash text NOT NULL,
 conflicting_hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(producer_id,identity_hash,conflicting_hash)
);
CREATE TABLE audit_outbox (
 id uuid PRIMARY KEY,
 request_id uuid NOT NULL REFERENCES notification_requests(id),
 action text NOT NULL,
 actor text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 published_at timestamptz
);
