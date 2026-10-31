-- order-persistence-messaging-1.0-review, PostgreSQL 16, 2026-10-17.
-- EMPTY ISOLATED SCHEMA ONLY. Design artifact, NOT an applied/upgrade migration.
-- UUIDs supplied by application; no provider IDs used as internal PKs.
-- No cross-service foreign keys. All nullable columns are intentionally optional.
CREATE SCHEMA order_pm_review;
SET search_path TO order_pm_review, pg_catalog;

CREATE DOMAIN hash256 AS text CHECK (VALUE ~ '^[0-9a-f]{64}$');
CREATE DOMAIN vnd_amount AS bigint CHECK (VALUE >= 0);

CREATE TABLE checkout_operations (
  id uuid PRIMARY KEY, planned_order_id uuid NOT NULL UNIQUE,
  owner_scope text NOT NULL, quote_id uuid NOT NULL UNIQUE, cart_id uuid NOT NULL,
  cart_revision bigint NOT NULL CHECK (cart_revision >= 0), selection_digest hash256 NOT NULL,
  input_version integer NOT NULL CHECK (input_version > 0), input_cipher bytea NOT NULL,
  input_hash hash256 NOT NULL, canonical_cipher bytea, canonical_hash hash256,
  status text NOT NULL CHECK (status IN ('ACCEPTED','PROCESSING','WAITING_RETRY','COMPENSATING','SUCCEEDED','FAILED','MANUAL_REVIEW')),
  stage text NOT NULL, payment_method text NOT NULL CHECK (payment_method IN ('VIETQR','COD')),
  commercial_expires_at timestamptz, accepted_at timestamptz NOT NULL,
  completed_at timestamptz, error_code text, version bigint NOT NULL CHECK (version > 0),
  UNIQUE (id, planned_order_id)
);
CREATE TABLE idempotency_records (
  scope text NOT NULL, action text NOT NULL, idempotency_key text NOT NULL,
  request_hash hash256 NOT NULL, operation_id uuid NOT NULL REFERENCES checkout_operations(id),
  response_cipher bytea, response_expires_at timestamptz NOT NULL,
  identity_min_until timestamptz NOT NULL,
  PRIMARY KEY (scope, action, idempotency_key),
  CHECK (octet_length(idempotency_key) BETWEEN 16 AND 128)
);
CREATE TABLE checkout_quote_consumptions (
  quote_id uuid PRIMARY KEY, operation_id uuid NOT NULL UNIQUE REFERENCES checkout_operations(id),
  owner_scope text NOT NULL, request_hash hash256 NOT NULL, claimed_at timestamptz NOT NULL
);
CREATE TABLE checkout_submission_guards (
  owner_scope text NOT NULL, cart_id uuid NOT NULL, cart_revision bigint NOT NULL,
  selection_digest hash256 NOT NULL, operation_id uuid NOT NULL REFERENCES checkout_operations(id),
  state text NOT NULL CHECK (state IN ('ACTIVE','CONSUMED','RELEASED')),
  version bigint NOT NULL CHECK (version > 0),
  PRIMARY KEY (owner_scope, cart_id, cart_revision, selection_digest)
);
CREATE TABLE orders (
  id uuid PRIMARY KEY, operation_id uuid NOT NULL UNIQUE,
  order_code text NOT NULL UNIQUE, cohort text NOT NULL,
  owner_kind text NOT NULL CHECK (owner_kind IN ('CUSTOMER','GUEST')),
  customer_id uuid, guest_session_id uuid, warehouse_id text NOT NULL,
  status text NOT NULL CHECK (status IN ('PENDING_PAYMENT','PAYMENT_FINALIZING','PAID','CONFIRMED_COD','PROCESSING','PACKED','SHIPPED','DELIVERED','DELIVERY_FAILED','COMPLETED','CANCELLED_TIMEOUT','CANCELLED_BY_USER','CANCELLED_BY_ADMIN')),
  payment_method text NOT NULL CHECK (payment_method IN ('VIETQR','COD')),
  payment_status text NOT NULL, stock_status text NOT NULL,
  final_amount vnd_amount NOT NULL CHECK (final_amount > 0 AND final_amount <= 1000000000000),
  currency text NOT NULL CHECK (currency = 'VND'),
  snapshot_id uuid NOT NULL UNIQUE, snapshot_version integer NOT NULL CHECK (snapshot_version > 0),
  snapshot_hash hash256 NOT NULL, snapshot_cipher bytea NOT NULL,
  version bigint NOT NULL CHECK (version > 0), readiness_generation bigint NOT NULL DEFAULT 0 CHECK (readiness_generation >= 0),
  payment_expires_at timestamptz, state_entered_at timestamptz NOT NULL,
  sla_due_at timestamptz, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  FOREIGN KEY (operation_id, id) REFERENCES checkout_operations(id, planned_order_id),
  CHECK ((owner_kind = 'CUSTOMER' AND customer_id IS NOT NULL AND guest_session_id IS NULL)
      OR (owner_kind = 'GUEST' AND customer_id IS NULL AND guest_session_id IS NOT NULL))
);
-- operation result is resolved through orders.operation_id, not a second writable order_id.
CREATE TABLE order_items (
  order_id uuid NOT NULL REFERENCES orders(id), sku text NOT NULL,
  quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 999),
  unit_price vnd_amount NOT NULL, line_total vnd_amount NOT NULL,
  PRIMARY KEY (order_id, sku)
);
CREATE TABLE order_holds (
  id uuid PRIMARY KEY, order_id uuid NOT NULL REFERENCES orders(id),
  source text NOT NULL, source_ref text NOT NULL, kind text NOT NULL, mode text NOT NULL,
  state text NOT NULL CHECK (state IN ('OPEN','CLOSED')), reason_code text NOT NULL,
  decision_cipher bytea NOT NULL, version bigint NOT NULL CHECK (version > 0),
  created_at timestamptz NOT NULL, closed_at timestamptz,
  UNIQUE (order_id, source, source_ref)
);
CREATE TABLE order_cancellations (
  id uuid PRIMARY KEY, order_id uuid NOT NULL REFERENCES orders(id),
  active boolean NOT NULL, stage text NOT NULL, compensation_status text NOT NULL,
  approval_cipher bytea, barrier_proof_cipher bytea, version bigint NOT NULL CHECK (version > 0),
  created_at timestamptz NOT NULL, completed_at timestamptz
);
CREATE UNIQUE INDEX cancellation_active ON order_cancellations(order_id) WHERE active;
CREATE TABLE command_registry (
  scope text NOT NULL, action text NOT NULL, idempotency_key text NOT NULL,
  order_id uuid NOT NULL REFERENCES orders(id), request_hash hash256 NOT NULL,
  result_cipher bytea, status text NOT NULL, created_at timestamptz NOT NULL,
  PRIMARY KEY (scope, action, idempotency_key)
);
CREATE TABLE order_milestones (
  order_id uuid NOT NULL REFERENCES orders(id), kind text NOT NULL,
  decision_ref text NOT NULL, occurred_at timestamptz NOT NULL,
  PRIMARY KEY (order_id, kind)
);
CREATE TABLE order_history (
  id uuid PRIMARY KEY, order_id uuid NOT NULL REFERENCES orders(id),
  version bigint NOT NULL CHECK (version > 0), entry_sequence integer NOT NULL CHECK (entry_sequence >= 0),
  actor_ref text NOT NULL, source text NOT NULL, reason_code text NOT NULL,
  changes_cipher bytea NOT NULL, occurred_at timestamptz NOT NULL,
  UNIQUE (order_id, version, entry_sequence)
);

CREATE TABLE payment_provider_connections (
  id uuid PRIMARY KEY, provider text NOT NULL, merchant_identity text NOT NULL,
  environment text NOT NULL, secret_ref text NOT NULL,
  UNIQUE (provider, merchant_identity, environment)
);
CREATE TABLE payment_receiver_profiles (
  id uuid PRIMARY KEY, connection_id uuid NOT NULL REFERENCES payment_provider_connections(id),
  bank_account_uuid text NOT NULL, config_version integer NOT NULL CHECK (config_version > 0),
  config_cipher bytea NOT NULL, accepting_new_payments boolean NOT NULL,
  UNIQUE (connection_id, bank_account_uuid, config_version), UNIQUE (id, connection_id)
);
CREATE TABLE payment_intents (
  id uuid PRIMARY KEY, order_id uuid NOT NULL UNIQUE REFERENCES orders(id),
  connection_id uuid NOT NULL REFERENCES payment_provider_connections(id),
  receiver_profile_id uuid NOT NULL, payment_code text NOT NULL,
  amount vnd_amount NOT NULL CHECK (amount > 0), currency text NOT NULL CHECK (currency = 'VND'),
  status text NOT NULL, expires_at timestamptz NOT NULL, version bigint NOT NULL CHECK (version > 0),
  UNIQUE (connection_id, payment_code),
  FOREIGN KEY (receiver_profile_id, connection_id) REFERENCES payment_receiver_profiles(id, connection_id)
);
CREATE TABLE receipts (
  id uuid PRIMARY KEY, connection_id uuid NOT NULL REFERENCES payment_provider_connections(id),
  transaction_id text NOT NULL, receiver_profile_id uuid NOT NULL,
  amount vnd_amount NOT NULL CHECK (amount > 0), currency text NOT NULL CHECK (currency = 'VND'),
  normalized_hash hash256 NOT NULL, evidence_cipher bytea NOT NULL, reference_lookup_hash hash256,
  bank_occurred_at timestamptz NOT NULL, observed_at timestamptz NOT NULL,
  version bigint NOT NULL CHECK (version > 0),
  UNIQUE (connection_id, transaction_id),
  FOREIGN KEY (receiver_profile_id, connection_id) REFERENCES payment_receiver_profiles(id, connection_id)
);
-- No order FK on receipts: allocation/matching is a separate decision.
CREATE TABLE allocations (
  id uuid PRIMARY KEY, receipt_id uuid NOT NULL UNIQUE REFERENCES receipts(id),
  order_id uuid NOT NULL UNIQUE REFERENCES orders(id), amount vnd_amount NOT NULL CHECK (amount > 0),
  allocated_at timestamptz NOT NULL
);
CREATE TABLE payment_provider_observations (
  id uuid PRIMARY KEY, connection_id uuid NOT NULL REFERENCES payment_provider_connections(id),
  source_kind text NOT NULL, external_id text NOT NULL, raw_cipher bytea NOT NULL,
  raw_hash hash256 NOT NULL, normalized_hash hash256 NOT NULL, status text NOT NULL,
  linked_receipt_id uuid REFERENCES receipts(id), observed_at timestamptz NOT NULL, last_seen_at timestamptz NOT NULL,
  UNIQUE (connection_id, source_kind, external_id)
);
CREATE TABLE payment_observation_conflicts (
  observation_id uuid NOT NULL REFERENCES payment_provider_observations(id),
  incoming_hash hash256 NOT NULL, evidence_cipher bytea NOT NULL,
  created_at timestamptz NOT NULL, PRIMARY KEY (observation_id, incoming_hash)
);
CREATE TABLE payment_order_projections (
  order_id uuid PRIMARY KEY REFERENCES orders(id), payment_hold boolean NOT NULL,
  has_open_reconciliation boolean NOT NULL, verification_started_at timestamptz,
  verification_due_at timestamptz, support_reference text, version bigint NOT NULL CHECK (version > 0)
);
CREATE TABLE guest_payment_grants (
  id uuid PRIMARY KEY, order_id uuid NOT NULL REFERENCES orders(id), sid uuid NOT NULL,
  jti_hash hash256 NOT NULL UNIQUE, grant_version bigint NOT NULL CHECK (grant_version > 0),
  expires_at timestamptz NOT NULL, revoked_at timestamptz
);
CREATE TABLE reconciliation_cases (
  id uuid PRIMARY KEY, receipt_id uuid REFERENCES receipts(id),
  observation_id uuid REFERENCES payment_provider_observations(id),
  kind text NOT NULL, reason_code text NOT NULL, state text NOT NULL,
  assignee text, next_action_at timestamptz NOT NULL, resolution_cipher bytea,
  version bigint NOT NULL CHECK (version > 0),
  CHECK (receipt_id IS NOT NULL OR observation_id IS NOT NULL)
);
CREATE TABLE refunds (
  id uuid PRIMARY KEY, receipt_id uuid NOT NULL REFERENCES receipts(id), order_id uuid REFERENCES orders(id),
  approval_id text NOT NULL UNIQUE, approval_cipher bytea NOT NULL, operation_key text NOT NULL UNIQUE,
  amount vnd_amount NOT NULL CHECK (amount > 0), currency text NOT NULL CHECK (currency = 'VND'),
  status text NOT NULL, execution_mode text NOT NULL CHECK (execution_mode IN ('MANUAL_BANK','PROVIDER_API')),
  execution_state text NOT NULL, budget_state text NOT NULL CHECK (budget_state IN ('RESERVED','SETTLED','RELEASED')),
  claim_owner text, claim_until timestamptz, provider_ref text, version bigint NOT NULL CHECK (version > 0),
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL
);
CREATE TABLE refund_execution_attempts (
  id uuid PRIMARY KEY, refund_id uuid NOT NULL REFERENCES refunds(id), attempt_no integer NOT NULL CHECK (attempt_no > 0),
  request_hash hash256 NOT NULL, stable_provider_key text NOT NULL,
  evidence_cipher bytea NOT NULL, created_at timestamptz NOT NULL,
  UNIQUE (refund_id, attempt_no)
);
CREATE TABLE payment_evidence (
  id uuid PRIMARY KEY, refund_id uuid REFERENCES refunds(id), receipt_id uuid REFERENCES receipts(id),
  uploader_ref text NOT NULL, object_ref text NOT NULL, content_hash hash256 NOT NULL,
  size_bytes bigint NOT NULL CHECK (size_bytes > 0), media_type text NOT NULL, scan_state text NOT NULL,
  retention_class text NOT NULL, created_at timestamptz NOT NULL,
  CHECK (refund_id IS NOT NULL OR receipt_id IS NOT NULL)
);
CREATE TABLE payment_sync_runs (
  id uuid PRIMARY KEY, connection_id uuid NOT NULL REFERENCES payment_provider_connections(id),
  receiver_profile_id uuid NOT NULL, stream text NOT NULL,
  active boolean NOT NULL, status text NOT NULL, mode text NOT NULL CHECK (mode IN ('WINDOW','CURSOR')),
  window_start timestamptz NOT NULL, window_end timestamptz NOT NULL,
  filter_hash hash256 NOT NULL, cursor_cipher bytea, last_durable_page integer NOT NULL DEFAULT 0,
  observed_count bigint NOT NULL DEFAULT 0, quarantine_count bigint NOT NULL DEFAULT 0,
  FOREIGN KEY (receiver_profile_id, connection_id) REFERENCES payment_receiver_profiles(id, connection_id),
  CHECK (window_end >= window_start)
);
CREATE UNIQUE INDEX payment_sync_active ON payment_sync_runs(receiver_profile_id, stream) WHERE active;
CREATE TABLE payment_sync_checkpoints (
  receiver_profile_id uuid NOT NULL REFERENCES payment_receiver_profiles(id), stream text NOT NULL,
  last_fetch_at timestamptz, last_scanned_end timestamptz, bank_reconciled_through timestamptz,
  version bigint NOT NULL CHECK (version > 0), PRIMARY KEY (receiver_profile_id, stream)
);

CREATE TABLE workflows (
  id uuid PRIMARY KEY, kind text NOT NULL, workflow_version integer NOT NULL CHECK (workflow_version > 0),
  aggregate_type text NOT NULL, aggregate_id uuid NOT NULL, order_id uuid REFERENCES orders(id),
  stage text NOT NULL, state text NOT NULL CHECK (state IN ('RUNNING','WAITING_EXTERNAL','COMPENSATING','MANUAL_REVIEW','COMPLETED')),
  completion_outcome text CHECK (completion_outcome IN ('SUCCEEDED','REJECTED','COMPENSATED')),
  policy_version text NOT NULL, stage_started_at timestamptz NOT NULL, stage_deadline timestamptz,
  business_deadlines jsonb NOT NULL, version bigint NOT NULL CHECK (version > 0),
  CHECK ((state = 'COMPLETED') = (completion_outcome IS NOT NULL))
);
CREATE TABLE jobs (
  id uuid PRIMARY KEY, workflow_id uuid NOT NULL REFERENCES workflows(id), logical_work_key text NOT NULL UNIQUE,
  kind text NOT NULL, input_version integer NOT NULL CHECK (input_version > 0), input_cipher bytea NOT NULL,
  input_hash hash256 NOT NULL, policy_version text NOT NULL, policy_snapshot jsonb NOT NULL,
  state text NOT NULL CHECK (state IN ('READY','LEASED','RETRY_WAIT','PARKED','DONE')),
  lane text NOT NULL, priority integer NOT NULL, run_after timestamptz NOT NULL,
  lease_owner uuid, lease_until timestamptz, lease_epoch bigint NOT NULL DEFAULT 0 CHECK (lease_epoch >= 0),
  execution_attempts integer NOT NULL DEFAULT 0 CHECK (execution_attempts >= 0),
  reconciliation_probes bigint NOT NULL DEFAULT 0 CHECK (reconciliation_probes >= 0),
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  CHECK ((state = 'LEASED' AND lease_owner IS NOT NULL AND lease_until IS NOT NULL)
      OR (state <> 'LEASED' AND lease_owner IS NULL AND lease_until IS NULL))
);
CREATE TABLE external_effects (
  id uuid PRIMARY KEY, workflow_id uuid NOT NULL REFERENCES workflows(id),
  provider text NOT NULL, scope text NOT NULL, action text NOT NULL, stable_key text NOT NULL,
  request_version text NOT NULL, request_hash hash256 NOT NULL, request_cipher bytea NOT NULL,
  resource_ref text, outcome text NOT NULL CHECK (outcome IN ('NOT_ATTEMPTED','UNKNOWN','SUCCEEDED','REJECTED','CONFLICT')),
  resource_state text, resource_version bigint, decision_at timestamptz,
  obligation_state text NOT NULL CHECK (obligation_state IN ('OPEN','RESOLVED','BLOCKED')),
  next_reconcile_at timestamptz, version bigint NOT NULL CHECK (version > 0),
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  compensates_effect_id uuid REFERENCES external_effects(id),
  UNIQUE (provider, scope, action, stable_key),
  CHECK (obligation_state <> 'OPEN' OR next_reconcile_at IS NOT NULL)
);
CREATE TABLE external_attempts (
  id uuid PRIMARY KEY, effect_id uuid NOT NULL REFERENCES external_effects(id), dispatch_sequence integer NOT NULL,
  worker_epoch bigint NOT NULL, request_hash hash256 NOT NULL,
  started_at timestamptz NOT NULL, deadline_at timestamptz NOT NULL,
  UNIQUE (effect_id, dispatch_sequence), CHECK (deadline_at > started_at)
);
CREATE TABLE external_observations (
  id uuid PRIMARY KEY, effect_id uuid NOT NULL REFERENCES external_effects(id),
  attempt_id uuid REFERENCES external_attempts(id), source text NOT NULL, observation_key text NOT NULL,
  evidence_hash hash256 NOT NULL, evidence_cipher bytea NOT NULL, observed_at timestamptz NOT NULL,
  UNIQUE (source, observation_key, evidence_hash)
);
CREATE TABLE external_observation_dispositions (
  observation_id uuid PRIMARY KEY REFERENCES external_observations(id),
  state text NOT NULL CHECK (state IN ('PENDING','VALIDATED','REJECTED','CONFLICT')),
  validator_version text NOT NULL, validated_at timestamptz, reason_code text
);
CREATE TABLE runtime_cases (
  id uuid PRIMARY KEY, workflow_id uuid NOT NULL REFERENCES workflows(id),
  effect_id uuid REFERENCES external_effects(id), reason_code text NOT NULL, state text NOT NULL,
  assignee text, next_action_at timestamptz NOT NULL, evidence_cipher bytea,
  version bigint NOT NULL CHECK (version > 0)
);
CREATE TABLE cart_cleanup_results (
  operation_id uuid PRIMARY KEY REFERENCES checkout_operations(id), job_id uuid NOT NULL UNIQUE REFERENCES jobs(id),
  cleanup_key text NOT NULL UNIQUE, status text NOT NULL, result_revision bigint, updated_at timestamptz NOT NULL
);

CREATE TABLE event_streams (
  id uuid PRIMARY KEY, aggregate_type text NOT NULL, aggregate_id uuid NOT NULL,
  topic text NOT NULL, partition_key text NOT NULL, routing_version integer NOT NULL CHECK (routing_version > 0),
  last_sequence bigint NOT NULL DEFAULT 0 CHECK (last_sequence >= 0),
  published_sequence bigint NOT NULL DEFAULT 0 CHECK (published_sequence >= 0),
  lease_owner uuid, lease_until timestamptz, lease_epoch bigint NOT NULL DEFAULT 0 CHECK (lease_epoch >= 0),
  cycle_started_at timestamptz NOT NULL, cycle_attempts integer NOT NULL DEFAULT 0 CHECK (cycle_attempts >= 0),
  last_error_code text, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  publish_state text NOT NULL CHECK (publish_state IN ('ACTIVE','BLOCKED')),
  run_after timestamptz NOT NULL, blocked_reason text,
  UNIQUE (aggregate_type, aggregate_id), UNIQUE (id, aggregate_type, aggregate_id),
  CHECK (published_sequence <= last_sequence),
  CHECK ((lease_owner IS NULL) = (lease_until IS NULL))
);
CREATE TABLE outbox_events (
  event_id uuid PRIMARY KEY, stream_id uuid NOT NULL,
  aggregate_type text NOT NULL, aggregate_id uuid NOT NULL,
  aggregate_version bigint NOT NULL CHECK (aggregate_version > 0), ordinal integer NOT NULL CHECK (ordinal >= 0),
  stream_sequence bigint NOT NULL CHECK (stream_sequence > 0), order_id uuid REFERENCES orders(id),
  event_type text NOT NULL, schema_ref text NOT NULL, schema_hash hash256 NOT NULL,
  payload_bytes bytea NOT NULL, payload_hash hash256 NOT NULL, headers jsonb NOT NULL,
  occurred_at timestamptz NOT NULL, publish_status text NOT NULL CHECK (publish_status IN ('PENDING','PUBLISHED','BLOCKED')),
  broker_partition integer, broker_offset bigint, published_at timestamptz,
  UNIQUE (aggregate_type, aggregate_id, aggregate_version, ordinal), UNIQUE (stream_id, stream_sequence),
  FOREIGN KEY (stream_id, aggregate_type, aggregate_id) REFERENCES event_streams(id, aggregate_type, aggregate_id)
);
CREATE TABLE inbox_events (
  id uuid PRIMARY KEY, consumer_contract text NOT NULL, source text NOT NULL, event_id text NOT NULL,
  payload_hash hash256 NOT NULL, raw_cipher bytea NOT NULL, schema_ref text NOT NULL,
  source_aggregate_type text NOT NULL, source_aggregate_id text NOT NULL,
  source_version bigint, source_ordinal integer, state text NOT NULL CHECK (state IN ('APPLIED','DEFERRED','QUARANTINED','IGNORED')),
  order_id uuid REFERENCES orders(id), disposition_reason text NOT NULL,
  first_received_at timestamptz NOT NULL, last_seen_at timestamptz NOT NULL,
  UNIQUE (consumer_contract, source, event_id)
);
CREATE TABLE inbox_deliveries (
  consumer_contract text NOT NULL, cluster_id text NOT NULL, topic text NOT NULL,
  partition_id integer NOT NULL, offset_id bigint NOT NULL, inbox_id uuid REFERENCES inbox_events(id),
  payload_hash hash256 NOT NULL, disposition text NOT NULL CHECK (disposition IN ('DURABLE','QUARANTINED')),
  PRIMARY KEY (consumer_contract, cluster_id, topic, partition_id, offset_id)
);
CREATE TABLE deferred_events (
  inbox_id uuid PRIMARY KEY REFERENCES inbox_events(id), job_id uuid NOT NULL UNIQUE REFERENCES jobs(id),
  prerequisite_key text NOT NULL, next_check_at timestamptz NOT NULL, reason_code text NOT NULL,
  resolved_at timestamptz
);
CREATE TABLE message_quarantine (
  id uuid PRIMARY KEY, consumer_contract text NOT NULL, source text NOT NULL, event_identity text NOT NULL,
  payload_hash hash256 NOT NULL, raw_cipher bytea NOT NULL, reason_code text NOT NULL,
  schema_ref text, inbox_id uuid REFERENCES inbox_events(id), state text NOT NULL,
  created_at timestamptz NOT NULL, next_action_at timestamptz NOT NULL,
  UNIQUE (consumer_contract, source, event_identity, payload_hash)
);
CREATE TABLE source_projections (
  order_id uuid NOT NULL REFERENCES orders(id), source text NOT NULL, resource_ref text NOT NULL,
  resource_version bigint NOT NULL, proof_hash hash256 NOT NULL, projection_cipher bytea NOT NULL,
  updated_at timestamptz NOT NULL, PRIMARY KEY (order_id, source, resource_ref)
);
CREATE TABLE audit_records (
  id uuid PRIMARY KEY, actor_ref text NOT NULL, action text NOT NULL, reason_code text NOT NULL,
  aggregate_type text NOT NULL, aggregate_id uuid NOT NULL, evidence_hash hash256 NOT NULL,
  evidence_cipher bytea NOT NULL, occurred_at timestamptz NOT NULL
);

CREATE INDEX order_customer_list ON orders(customer_id, created_at DESC, id DESC) WHERE owner_kind = 'CUSTOMER';
CREATE INDEX order_guest_list ON orders(guest_session_id, created_at DESC, id DESC) WHERE owner_kind = 'GUEST';
CREATE INDEX order_staff_queue ON orders(warehouse_id, status, sla_due_at, created_at, id);
CREATE INDEX order_payment_timeout ON orders(payment_expires_at, id) WHERE status = 'PENDING_PAYMENT';
CREATE INDEX holds_open ON order_holds(order_id, id) WHERE state = 'OPEN';
CREATE INDEX receipt_lookup ON receipts(connection_id, reference_lookup_hash, observed_at);
CREATE INDEX refunds_budget ON refunds(receipt_id, budget_state);
CREATE INDEX observation_resolver ON payment_provider_observations(status, observed_at, id);
CREATE INDEX workflow_aggregate ON workflows(aggregate_type, aggregate_id);
CREATE INDEX job_ready ON jobs(lane, run_after, priority, id) WHERE state IN ('READY','RETRY_WAIT');
CREATE INDEX job_expired ON jobs(lease_until, id) WHERE state = 'LEASED';
CREATE INDEX effect_reconcile ON external_effects(next_reconcile_at, id) WHERE obligation_state <> 'RESOLVED';
CREATE INDEX effect_workflow ON external_effects(workflow_id);
CREATE INDEX observation_effect ON external_observations(effect_id, observed_at);
CREATE INDEX observation_pending ON external_observation_dispositions(state, observation_id) WHERE state = 'PENDING';
CREATE INDEX runtime_case_due ON runtime_cases(next_action_at, id) WHERE state <> 'RESOLVED';
CREATE INDEX payment_case_due ON reconciliation_cases(next_action_at, id) WHERE state <> 'RESOLVED';
CREATE INDEX stream_ready ON event_streams(run_after, lease_until, id) WHERE publish_state = 'ACTIVE';
CREATE INDEX outbox_pending ON outbox_events(stream_id, stream_sequence) WHERE publish_status <> 'PUBLISHED';
CREATE INDEX deferred_due ON deferred_events(next_check_at, inbox_id) WHERE resolved_at IS NULL;
CREATE INDEX quarantine_due ON message_quarantine(next_action_at, id) WHERE state <> 'RESOLVED';

-- Immutable evidence has mutable validation/projection in separate tables.
CREATE FUNCTION reject_record_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'immutable evidence: %', TG_TABLE_NAME;
END;
$$;
CREATE TRIGGER history_immutable BEFORE UPDATE OR DELETE ON order_history FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER milestones_immutable BEFORE UPDATE OR DELETE ON order_milestones FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER receipt_immutable BEFORE UPDATE OR DELETE ON receipts FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER allocation_immutable BEFORE UPDATE OR DELETE ON allocations FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER effect_attempt_immutable BEFORE UPDATE OR DELETE ON external_attempts FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER external_observation_immutable BEFORE UPDATE OR DELETE ON external_observations FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER refund_attempt_immutable BEFORE UPDATE OR DELETE ON refund_execution_attempts FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER audit_immutable BEFORE UPDATE OR DELETE ON audit_records FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER quote_consumption_immutable BEFORE UPDATE OR DELETE ON checkout_quote_consumptions FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();
CREATE TRIGGER order_item_immutable BEFORE UPDATE OR DELETE ON order_items FOR EACH ROW EXECUTE FUNCTION reject_record_mutation();

CREATE FUNCTION protect_order_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF ROW(OLD.operation_id, OLD.snapshot_id, OLD.snapshot_version, OLD.snapshot_hash,
         OLD.final_amount, OLD.currency, OLD.payment_method)
     IS DISTINCT FROM
     ROW(NEW.operation_id, NEW.snapshot_id, NEW.snapshot_version, NEW.snapshot_hash,
         NEW.final_amount, NEW.currency, NEW.payment_method) THEN
    RAISE EXCEPTION 'immutable commercial identity';
  END IF;
  -- Ciphertext rewrap allowed only with unchanged logical hash/version and audited role.
  RETURN NEW;
END;
$$;
CREATE TRIGGER snapshot_identity BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION protect_order_snapshot();

CREATE FUNCTION protect_outbox_identity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF ROW(OLD.event_id, OLD.stream_id, OLD.aggregate_type, OLD.aggregate_id,
         OLD.aggregate_version, OLD.ordinal, OLD.stream_sequence, OLD.order_id,
         OLD.event_type, OLD.schema_ref, OLD.schema_hash, OLD.payload_bytes,
         OLD.payload_hash, OLD.headers, OLD.occurred_at)
     IS DISTINCT FROM
     ROW(NEW.event_id, NEW.stream_id, NEW.aggregate_type, NEW.aggregate_id,
         NEW.aggregate_version, NEW.ordinal, NEW.stream_sequence, NEW.order_id,
         NEW.event_type, NEW.schema_ref, NEW.schema_hash, NEW.payload_bytes,
         NEW.payload_hash, NEW.headers, NEW.occurred_at) THEN
    RAISE EXCEPTION 'immutable outbox event';
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER outbox_identity BEFORE UPDATE ON outbox_events FOR EACH ROW EXECUTE FUNCTION protect_outbox_identity();
-- Deployment migrations must add least-privilege grants, expand/backfill/validate,
-- cohort adapters and approved archive procedures; this file does not perform them.
