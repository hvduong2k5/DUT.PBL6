-- Order 3.1: async checkout operations. PostgreSQL 16.
CREATE TABLE checkout_operations (
 id uuid PRIMARY KEY, scope text NOT NULL, principal_id uuid NOT NULL,
 planned_order_id uuid NOT NULL UNIQUE, order_id uuid,
 status text NOT NULL CHECK(status IN ('ACCEPTED','PROCESSING','WAITING_RETRY','COMPENSATING','SUCCEEDED','FAILED','MANUAL_REVIEW')),
 stage text NOT NULL DEFAULT 'VALIDATE', input_cipher bytea NOT NULL, canonical_cipher bytea,
 request_hash text NOT NULL, reservation_id text, reservation_expires_at timestamptz,
 accepted_at timestamptz NOT NULL DEFAULT clock_timestamp(), commercial_expires_at timestamptz,
 error_code text, error_http integer, attempts integer NOT NULL DEFAULT 0,
 run_after timestamptz NOT NULL DEFAULT clock_timestamp(), lease_owner text,
 lease_until timestamptz, lease_epoch bigint NOT NULL DEFAULT 0,
 completed_at timestamptz
);
CREATE INDEX checkout_worker ON checkout_operations(run_after,lease_until) WHERE status IN ('ACCEPTED','PROCESSING','WAITING_RETRY','COMPENSATING');
CREATE TABLE idempotency_records (
 scope text NOT NULL, kind text NOT NULL, key text NOT NULL, request_hash text NOT NULL,
 operation_id uuid NOT NULL REFERENCES checkout_operations(id),
 response_expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '7 days',
 tombstone_min_until timestamptz NOT NULL DEFAULT clock_timestamp()+interval '30 days',
 PRIMARY KEY(scope,kind,key)
 -- Registry is never automatically deleted. Expired terminal replay is 410.
);
CREATE TABLE orders (
 id uuid PRIMARY KEY, operation_id uuid NOT NULL UNIQUE REFERENCES checkout_operations(id),
 order_code text NOT NULL UNIQUE, scope text NOT NULL, principal_id uuid NOT NULL,
 customer_id uuid, warehouse_id text NOT NULL DEFAULT 'HUE',
 channel text NOT NULL DEFAULT 'D2C_WEB', method text NOT NULL CHECK(method IN ('VIETQR','COD')),
 status text NOT NULL CHECK(status IN ('PENDING_PAYMENT','PAYMENT_FINALIZING','PAID','CONFIRMED_COD','PROCESSING','PACKED','SHIPPED','DELIVERED','DELIVERY_FAILED','COMPLETED','CANCELLED_TIMEOUT','CANCELLED_BY_USER','CANCELLED_BY_ADMIN')),
 payment_status text NOT NULL DEFAULT 'UNPAID' CHECK(payment_status IN ('UNPAID','PENDING','CONFIRMED','RECONCILIATION_REQUIRED','PARTIALLY_REFUNDED','REFUNDED')),
 stock_status text NOT NULL CHECK(stock_status IN ('RESERVED','COMMITTING','COMMITTED','RELEASING','RELEASED','EXPIRED','UNKNOWN')),
 subtotal bigint NOT NULL CHECK(subtotal BETWEEN 1 AND 1000000000000),
 merchandise_discount bigint NOT NULL DEFAULT 0 CHECK(merchandise_discount>=0),
 shipping_fee bigint NOT NULL CHECK(shipping_fee>=0), shipping_discount bigint NOT NULL DEFAULT 0 CHECK(shipping_discount>=0),
 final_amount bigint NOT NULL CHECK(final_amount BETWEEN 1 AND 1000000000000),
 currency text NOT NULL DEFAULT 'VND' CHECK(currency='VND'),
 snapshot_cipher bytea NOT NULL, reservation_id text NOT NULL,
 payment_expires_at timestamptz NOT NULL, provider_reference text NOT NULL UNIQUE,
 version bigint NOT NULL DEFAULT 1 CHECK(version>0), ready_generation bigint NOT NULL DEFAULT 0,
 ready_authorized boolean NOT NULL DEFAULT false, operational_hold boolean NOT NULL DEFAULT false,
 has_active_case boolean NOT NULL DEFAULT false, task_id text, shipment_id text, seal_code text,
 sla_due_at timestamptz, completion_due_at timestamptz,
 policy_version text NOT NULL DEFAULT 'sandbox-v1',
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK(merchandise_discount<=subtotal AND shipping_discount<=shipping_fee),
 CHECK(final_amount=subtotal-merchandise_discount+shipping_fee-shipping_discount),
 CHECK(status<>'PAID' OR (payment_status IN ('CONFIRMED','PARTIALLY_REFUNDED','REFUNDED') AND stock_status='COMMITTED')),
 CHECK(status<>'CONFIRMED_COD' OR (method='COD' AND stock_status='COMMITTED'))
);
ALTER TABLE checkout_operations ADD FOREIGN KEY(order_id) REFERENCES orders(id);
CREATE INDEX order_owner ON orders(scope,created_at DESC,id DESC);
CREATE INDEX order_expiry ON orders(payment_expires_at) WHERE status='PENDING_PAYMENT';
CREATE INDEX order_sla ON orders(status,sla_due_at,id);
CREATE TABLE order_items (
 order_id uuid NOT NULL REFERENCES orders(id), sku text NOT NULL, quantity integer NOT NULL CHECK(quantity BETWEEN 1 AND 999),
 unit_price bigint NOT NULL CHECK(unit_price BETWEEN 1 AND 1000000000000),
 line_total bigint NOT NULL CHECK(line_total=quantity::bigint*unit_price), product_name text NOT NULL,
 PRIMARY KEY(order_id,sku)
);
CREATE FUNCTION immutable_order_items() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'order snapshots are immutable'; END $$;
CREATE TRIGGER immutable_items BEFORE UPDATE OR DELETE ON order_items FOR EACH ROW EXECUTE FUNCTION immutable_order_items();
CREATE FUNCTION immutable_order_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.snapshot_cipher IS DISTINCT FROM OLD.snapshot_cipher OR NEW.final_amount<>OLD.final_amount OR NEW.subtotal<>OLD.subtotal OR NEW.shipping_fee<>OLD.shipping_fee OR NEW.merchandise_discount<>OLD.merchandise_discount OR NEW.shipping_discount<>OLD.shipping_discount THEN
  RAISE EXCEPTION 'commercial snapshot is immutable';
 END IF; RETURN NEW;
END $$;
CREATE TRIGGER immutable_snapshot BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION immutable_order_snapshot();
CREATE TABLE order_history (
 id uuid PRIMARY KEY, order_id uuid NOT NULL REFERENCES orders(id), version bigint NOT NULL,
 from_status text, to_status text NOT NULL, source text NOT NULL, reason text NOT NULL DEFAULT '',
 occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(), UNIQUE(order_id,version)
);
CREATE TABLE receipts (
 id uuid PRIMARY KEY, provider text NOT NULL, transaction_id text NOT NULL,
 order_id uuid REFERENCES orders(id), reference text NOT NULL, receiver text NOT NULL,
 amount bigint NOT NULL CHECK(amount BETWEEN 1 AND 1000000000000), currency text NOT NULL CHECK(currency='VND'),
 payload_hash text NOT NULL, status text NOT NULL CHECK(status IN ('VERIFIED','ALLOCATED','RECONCILIATION_REQUIRED')),
 provider_status text NOT NULL CHECK(provider_status='SUCCEEDED'), version bigint NOT NULL DEFAULT 1,
 provider_paid_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(provider,transaction_id)
);
CREATE TABLE allocations (
 receipt_id uuid PRIMARY KEY REFERENCES receipts(id), order_id uuid NOT NULL UNIQUE REFERENCES orders(id), amount bigint NOT NULL CHECK(amount>0)
);
CREATE TABLE reconciliation_cases (
 id uuid PRIMARY KEY, receipt_id uuid NOT NULL UNIQUE REFERENCES receipts(id), reason text NOT NULL,
 status text NOT NULL DEFAULT 'OPEN', created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE jobs (
 id uuid PRIMARY KEY, order_id uuid NOT NULL REFERENCES orders(id), kind text NOT NULL, key text NOT NULL UNIQUE,
 stage text NOT NULL DEFAULT 'START', status text NOT NULL DEFAULT 'PENDING', payload jsonb NOT NULL DEFAULT '{}',
 attempts integer NOT NULL DEFAULT 0, run_after timestamptz NOT NULL DEFAULT clock_timestamp(),
 lease_owner text, lease_until timestamptz, lease_epoch bigint NOT NULL DEFAULT 0, error_code text,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX jobs_worker ON jobs(run_after,lease_until) WHERE status IN ('PENDING','RETRY');
CREATE TABLE cancellation_requests (
 id uuid PRIMARY KEY, order_id uuid NOT NULL REFERENCES orders(id), actor_id uuid NOT NULL,
 request_key text NOT NULL UNIQUE, reason text NOT NULL, status text NOT NULL DEFAULT 'PENDING_REVIEW',
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE refunds (
 id uuid PRIMARY KEY, order_id uuid NOT NULL REFERENCES orders(id), receipt_id uuid NOT NULL REFERENCES receipts(id),
 approval_id text NOT NULL UNIQUE, operation_key text NOT NULL UNIQUE,
 amount bigint NOT NULL CHECK(amount>0), status text NOT NULL CHECK(status IN ('APPROVED','SUBMITTED','SUCCEEDED','FAILED','UNKNOWN','MANUAL_REVIEW')),
 provider_ref text, version bigint NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE outbox (
 id uuid PRIMARY KEY, aggregate_id uuid NOT NULL, aggregate_type text NOT NULL, version bigint NOT NULL,
 ordinal integer NOT NULL, event_type text NOT NULL, topic text NOT NULL DEFAULT 'order.events.v2',
 payload jsonb NOT NULL, published_at timestamptz, run_after timestamptz NOT NULL DEFAULT clock_timestamp(),
 lease_owner text, lease_until timestamptz, lease_epoch bigint NOT NULL DEFAULT 0,
 UNIQUE(aggregate_type,aggregate_id,version,ordinal)
);
CREATE INDEX outbox_queue ON outbox(run_after) WHERE published_at IS NULL;
CREATE TABLE inbox (
 source text NOT NULL, event_id text NOT NULL, hash text NOT NULL, status text NOT NULL,
 order_id uuid NOT NULL, resource_id text NOT NULL, source_version bigint NOT NULL,
 payload jsonb NOT NULL, run_after timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(source,event_id)
);
CREATE TABLE source_projections (
 order_id uuid NOT NULL REFERENCES orders(id), source text NOT NULL, resource_id text NOT NULL,
 source_version bigint NOT NULL, PRIMARY KEY(order_id,source,resource_id)
);
CREATE TABLE command_registry (
 scope text NOT NULL, kind text NOT NULL, key text NOT NULL, request_hash text NOT NULL,
 order_id uuid NOT NULL REFERENCES orders(id), response jsonb NOT NULL,
 PRIMARY KEY(scope,kind,key)
);
CREATE TABLE audit_records (
 sequence bigserial UNIQUE NOT NULL, id uuid PRIMARY KEY, order_id uuid, actor text NOT NULL, action text NOT NULL,
 details jsonb NOT NULL, prev_hash text NOT NULL, hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE FUNCTION immutable_audit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'audit records are append-only'; END $$;
CREATE TRIGGER immutable_audit BEFORE UPDATE OR DELETE ON audit_records FOR EACH ROW EXECUTE FUNCTION immutable_audit();

CREATE TABLE quarantine (message_key text PRIMARY KEY,payload_hash text NOT NULL,error_code text NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp());
