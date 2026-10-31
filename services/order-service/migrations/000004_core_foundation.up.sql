-- Expand only. Existing rows and application defaults remain sandbox-3.1.
-- Target readers/writers opt into order-core-1; no traffic switch or down migration.
ALTER TABLE checkout_operations
 ADD COLUMN cohort text NOT NULL DEFAULT 'sandbox-3.1' CHECK(cohort IN ('sandbox-3.1','order-core-1')),
 ADD COLUMN quote_id uuid, ADD COLUMN cart_id uuid, ADD COLUMN cart_revision bigint,
 ADD COLUMN selection_digest text, ADD COLUMN quote_hash text, ADD COLUMN input_hash text,
 ADD COLUMN input_version integer, ADD COLUMN canonical_hash text,
 ADD COLUMN owner_kind text, ADD COLUMN customer_id uuid, ADD COLUMN guest_session_id uuid, ADD COLUMN payment_method text,
 ADD CONSTRAINT core_operation_shape CHECK(cohort <> 'order-core-1' OR
  (quote_id IS NOT NULL AND cart_id IS NOT NULL AND cart_revision IS NOT NULL AND cart_revision>=0
   AND input_version=1 AND input_version IS NOT NULL
   AND selection_digest IS NOT NULL AND selection_digest ~ '^[0-9a-f]{64}$'
   AND quote_hash IS NOT NULL AND quote_hash ~ '^[0-9a-f]{64}$'
   AND input_hash IS NOT NULL AND input_hash ~ '^[0-9a-f]{64}$'
   AND ((owner_kind='CUSTOMER' AND customer_id IS NOT NULL AND guest_session_id IS NULL)
     OR (owner_kind='GUEST' AND guest_session_id IS NOT NULL AND customer_id IS NULL))
   AND owner_kind IS NOT NULL AND payment_method IS NOT NULL AND payment_method IN ('VIETQR','COD')
   AND scope=lower(owner_kind)||':'||principal_id::text)),
 ADD UNIQUE(id,planned_order_id);
CREATE UNIQUE INDEX core_operation_quote ON checkout_operations(quote_id) WHERE cohort='order-core-1';
CREATE TABLE checkout_quote_consumptions (
 quote_id uuid PRIMARY KEY, operation_id uuid NOT NULL UNIQUE REFERENCES checkout_operations(id),
 owner_scope text NOT NULL, request_hash text NOT NULL CHECK(request_hash ~ '^[0-9a-f]{64}$'),
 claimed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE checkout_submission_guards (
 owner_scope text NOT NULL, cart_id uuid NOT NULL, cart_revision bigint NOT NULL CHECK(cart_revision>=0),
 selection_digest text NOT NULL CHECK(selection_digest ~ '^[0-9a-f]{64}$'),
 operation_id uuid NOT NULL REFERENCES checkout_operations(id),
 state text NOT NULL CHECK(state IN ('ACTIVE','CONSUMED','RELEASED')), version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 PRIMARY KEY(owner_scope,cart_id,cart_revision,selection_digest)
);
ALTER TABLE orders
 ADD COLUMN cohort text NOT NULL DEFAULT 'sandbox-3.1' CHECK(cohort IN ('sandbox-3.1','order-core-1')),
 ADD COLUMN owner_kind text, ADD COLUMN guest_session_id uuid,
 ADD COLUMN snapshot_id uuid, ADD COLUMN snapshot_schema_version text, ADD COLUMN snapshot_hash text,
 ADD COLUMN core_state_cipher bytea,
 ALTER COLUMN payment_expires_at DROP NOT NULL, ALTER COLUMN provider_reference DROP NOT NULL,
 ADD CONSTRAINT legacy_order_shape CHECK(cohort<>'sandbox-3.1' OR (payment_expires_at IS NOT NULL AND provider_reference IS NOT NULL)),
 ADD CONSTRAINT core_order_shape CHECK(cohort<>'order-core-1' OR
  (snapshot_id IS NOT NULL AND snapshot_schema_version='1.0' AND snapshot_schema_version IS NOT NULL
   AND snapshot_hash IS NOT NULL AND snapshot_hash ~ '^[0-9a-f]{64}$' AND core_state_cipher IS NOT NULL
   AND ((owner_kind='CUSTOMER' AND customer_id IS NOT NULL AND guest_session_id IS NULL)
     OR (owner_kind='GUEST' AND guest_session_id IS NOT NULL AND customer_id IS NULL)) AND owner_kind IS NOT NULL
   AND ((method='VIETQR' AND payment_expires_at IS NOT NULL) OR (method='COD' AND payment_expires_at IS NULL))));
CREATE UNIQUE INDEX core_snapshot_identity ON orders(snapshot_id) WHERE cohort='order-core-1';
-- Conditional FK via generated keys preserves even historical legacy ID mappings.
ALTER TABLE orders
 ADD COLUMN core_operation_id uuid GENERATED ALWAYS AS (CASE WHEN cohort='order-core-1' THEN operation_id END) STORED,
 ADD COLUMN core_order_id uuid GENERATED ALWAYS AS (CASE WHEN cohort='order-core-1' THEN id END) STORED,
 ADD CONSTRAINT order_planned_identity FOREIGN KEY(core_operation_id,core_order_id)
 REFERENCES checkout_operations(id,planned_order_id);
CREATE INDEX core_order_owner ON orders(scope,created_at DESC,id DESC) WHERE cohort='order-core-1';

CREATE TABLE payment_provider_connections (
 id uuid PRIMARY KEY, provider text NOT NULL, merchant_identity text NOT NULL, environment text NOT NULL,
 secret_ref text NOT NULL, UNIQUE(provider,merchant_identity,environment)
);
CREATE TABLE payment_receiver_profiles (
 id uuid PRIMARY KEY, connection_id uuid NOT NULL REFERENCES payment_provider_connections(id),
 bank_account_uuid text NOT NULL, config_version integer NOT NULL CHECK(config_version>0),
 config_cipher bytea NOT NULL, accepting_new_payments boolean NOT NULL,
 UNIQUE(connection_id,bank_account_uuid,config_version), UNIQUE(id,connection_id)
);
CREATE TABLE payment_intents (
 id uuid PRIMARY KEY, order_id uuid NOT NULL UNIQUE REFERENCES orders(id),
 connection_id uuid NOT NULL REFERENCES payment_provider_connections(id), receiver_profile_id uuid NOT NULL,
 payment_code text NOT NULL, amount bigint NOT NULL CHECK(amount BETWEEN 1 AND 1000000000000),
 currency text NOT NULL CHECK(currency='VND'), status text NOT NULL, expires_at timestamptz NOT NULL,
 version bigint NOT NULL DEFAULT 1 CHECK(version>0), UNIQUE(connection_id,payment_code),
 FOREIGN KEY(receiver_profile_id,connection_id) REFERENCES payment_receiver_profiles(id,connection_id)
);
CREATE TABLE workflows (
 id uuid PRIMARY KEY, kind text NOT NULL, workflow_version integer NOT NULL CHECK(workflow_version>0),
 aggregate_type text NOT NULL, aggregate_id uuid NOT NULL, order_id uuid REFERENCES orders(id),
 stage text NOT NULL, state text NOT NULL CHECK(state IN ('RUNNING','WAITING_EXTERNAL','COMPENSATING','MANUAL_REVIEW','COMPLETED')),
 completion_outcome text CHECK(completion_outcome IN ('SUCCEEDED','REJECTED','COMPENSATED')),
 policy_version text NOT NULL, stage_started_at timestamptz NOT NULL, stage_deadline timestamptz,
 business_deadlines jsonb NOT NULL DEFAULT '{}', version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 CHECK((state='COMPLETED')=(completion_outcome IS NOT NULL))
);
CREATE UNIQUE INDEX checkout_workflow_identity ON workflows(aggregate_type,aggregate_id,kind) WHERE kind='CHECKOUT';
CREATE TABLE order_milestones (
 order_id uuid NOT NULL REFERENCES orders(id), kind text NOT NULL CHECK(kind IN ('PLACED','FIRST_PAID','COMPLETED')),
 decision_ref text NOT NULL, occurred_at timestamptz NOT NULL, PRIMARY KEY(order_id,kind)
);
ALTER TABLE jobs ALTER COLUMN order_id DROP NOT NULL,
 ADD COLUMN cohort text NOT NULL DEFAULT 'sandbox-3.1' CHECK(cohort IN ('sandbox-3.1','order-core-1')),
 ADD COLUMN workflow_id uuid REFERENCES workflows(id), ADD COLUMN input_version integer,
 ADD COLUMN input_cipher bytea, ADD COLUMN input_hash text, ADD COLUMN lane text,
 ADD COLUMN priority integer NOT NULL DEFAULT 0,
 ADD CONSTRAINT legacy_job_shape CHECK(cohort<>'sandbox-3.1' OR order_id IS NOT NULL),
 ADD CONSTRAINT core_job_shape CHECK(cohort<>'order-core-1' OR
  (workflow_id IS NOT NULL AND input_version IS NOT NULL AND input_version>0 AND input_cipher IS NOT NULL
   AND input_hash IS NOT NULL AND input_hash ~ '^[0-9a-f]{64}$' AND lane IS NOT NULL
   AND status IN ('READY','LEASED','RETRY_WAIT','PARKED','DONE')
   AND ((status='LEASED' AND lease_owner IS NOT NULL AND lease_until IS NOT NULL)
     OR (status<>'LEASED' AND lease_owner IS NULL AND lease_until IS NULL))));
CREATE INDEX core_job_ready ON jobs(lane,run_after,priority,id) WHERE cohort='order-core-1' AND status IN ('READY','RETRY_WAIT');
CREATE INDEX core_job_expired ON jobs(lease_until,id) WHERE cohort='order-core-1' AND status='LEASED';
CREATE INDEX workflow_aggregate ON workflows(aggregate_type,aggregate_id);
CREATE TABLE event_streams (
 id uuid PRIMARY KEY, aggregate_type text NOT NULL, aggregate_id uuid NOT NULL,
 topic text NOT NULL, partition_key text NOT NULL, routing_version integer NOT NULL CHECK(routing_version>0),
 last_sequence bigint NOT NULL DEFAULT 0 CHECK(last_sequence>=0),
 published_sequence bigint NOT NULL DEFAULT 0 CHECK(published_sequence>=0 AND published_sequence<=last_sequence),
 lease_owner uuid, lease_until timestamptz, lease_epoch bigint NOT NULL DEFAULT 0 CHECK(lease_epoch>=0),
 publish_state text NOT NULL DEFAULT 'ACTIVE' CHECK(publish_state IN ('ACTIVE','BLOCKED')),
 run_after timestamptz NOT NULL DEFAULT clock_timestamp(), cycle_started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 cycle_attempts integer NOT NULL DEFAULT 0 CHECK(cycle_attempts>=0), last_error_code text,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(aggregate_type,aggregate_id), UNIQUE(id,aggregate_type,aggregate_id),
 CHECK((lease_owner IS NULL)=(lease_until IS NULL))
);
CREATE TABLE outbox_events (
 event_id uuid PRIMARY KEY, stream_id uuid NOT NULL, aggregate_type text NOT NULL, aggregate_id uuid NOT NULL,
 aggregate_version bigint NOT NULL CHECK(aggregate_version>0), ordinal integer NOT NULL CHECK(ordinal>=0),
 stream_sequence bigint NOT NULL CHECK(stream_sequence>0), order_id uuid REFERENCES orders(id),
 event_type text NOT NULL, schema_ref text NOT NULL, schema_hash text NOT NULL CHECK(schema_hash ~ '^[0-9a-f]{64}$'),
 payload_bytes bytea NOT NULL, payload_hash text NOT NULL CHECK(payload_hash ~ '^[0-9a-f]{64}$'), headers jsonb NOT NULL,
 occurred_at timestamptz NOT NULL, publish_status text NOT NULL DEFAULT 'PENDING' CHECK(publish_status IN ('PENDING','PUBLISHED','BLOCKED')),
 broker_partition integer, broker_offset bigint, published_at timestamptz,
 UNIQUE(aggregate_type,aggregate_id,aggregate_version,ordinal), UNIQUE(stream_id,stream_sequence),
 FOREIGN KEY(stream_id,aggregate_type,aggregate_id) REFERENCES event_streams(id,aggregate_type,aggregate_id)
);
CREATE INDEX outbox_events_head ON outbox_events(stream_id,stream_sequence) WHERE publish_status<>'PUBLISHED';
CREATE FUNCTION protect_core_outbox() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF ROW(NEW.event_id,NEW.stream_id,NEW.aggregate_type,NEW.aggregate_id,NEW.aggregate_version,NEW.ordinal,
        NEW.stream_sequence,NEW.order_id,NEW.event_type,NEW.schema_ref,NEW.schema_hash,NEW.payload_bytes,NEW.payload_hash,NEW.headers,NEW.occurred_at)
 IS DISTINCT FROM ROW(OLD.event_id,OLD.stream_id,OLD.aggregate_type,OLD.aggregate_id,OLD.aggregate_version,OLD.ordinal,
        OLD.stream_sequence,OLD.order_id,OLD.event_type,OLD.schema_ref,OLD.schema_hash,OLD.payload_bytes,OLD.payload_hash,OLD.headers,OLD.occurred_at)
 THEN RAISE EXCEPTION 'immutable outbox identity'; END IF; RETURN NEW;
END $$;
CREATE TRIGGER core_outbox_immutable BEFORE UPDATE ON outbox_events FOR EACH ROW EXECUTE FUNCTION protect_core_outbox();
CREATE FUNCTION protect_core_snapshot_identity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.cohort='order-core-1' AND ROW(NEW.id,NEW.operation_id,NEW.cohort,NEW.snapshot_id,NEW.snapshot_schema_version,NEW.snapshot_hash,NEW.method,NEW.currency)
 IS DISTINCT FROM ROW(OLD.id,OLD.operation_id,OLD.cohort,OLD.snapshot_id,OLD.snapshot_schema_version,OLD.snapshot_hash,OLD.method,OLD.currency)
 THEN RAISE EXCEPTION 'immutable core snapshot identity'; END IF; RETURN NEW;
END $$;
CREATE TRIGGER core_snapshot_identity BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION protect_core_snapshot_identity();
CREATE TRIGGER quote_consumption_immutable BEFORE UPDATE OR DELETE ON checkout_quote_consumptions FOR EACH ROW EXECUTE FUNCTION immutable_audit();
CREATE FUNCTION protect_core_accepted_input() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.cohort='order-core-1' AND ROW(NEW.id,NEW.cohort,NEW.scope,NEW.principal_id,NEW.planned_order_id,NEW.input_cipher,NEW.input_hash,
 NEW.input_version,NEW.request_hash,NEW.quote_id,NEW.cart_id,NEW.cart_revision,NEW.selection_digest,NEW.quote_hash,
 NEW.owner_kind,NEW.customer_id,NEW.guest_session_id,NEW.payment_method,NEW.accepted_at)
 IS DISTINCT FROM ROW(OLD.id,OLD.cohort,OLD.scope,OLD.principal_id,OLD.planned_order_id,OLD.input_cipher,OLD.input_hash,
 OLD.input_version,OLD.request_hash,OLD.quote_id,OLD.cart_id,OLD.cart_revision,OLD.selection_digest,OLD.quote_hash,
 OLD.owner_kind,OLD.customer_id,OLD.guest_session_id,OLD.payment_method,OLD.accepted_at)
 THEN RAISE EXCEPTION 'immutable accepted input'; END IF; RETURN NEW;
END $$;
CREATE TRIGGER core_accepted_immutable BEFORE UPDATE ON checkout_operations FOR EACH ROW EXECUTE FUNCTION protect_core_accepted_input();
CREATE FUNCTION protect_core_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM orders WHERE id=OLD.order_id AND cohort='order-core-1') THEN
  RAISE EXCEPTION 'immutable order history'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
CREATE TRIGGER core_history_immutable BEFORE UPDATE OR DELETE ON order_history FOR EACH ROW EXECUTE FUNCTION protect_core_history();
CREATE TRIGGER core_milestone_immutable BEFORE UPDATE OR DELETE ON order_milestones FOR EACH ROW EXECUTE FUNCTION immutable_audit();
