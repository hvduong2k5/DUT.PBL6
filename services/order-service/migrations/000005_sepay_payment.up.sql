-- Expand target Payment without changing the legacy simulator's financial identity.
ALTER TABLE receipts ADD COLUMN connection_id uuid REFERENCES payment_provider_connections(id),
 ADD COLUMN receiver_profile_id uuid REFERENCES payment_receiver_profiles(id), ADD COLUMN evidence_cipher bytea,
 ADD COLUMN api_observed_at timestamptz, ADD COLUMN reference_key text, DROP CONSTRAINT receipts_amount_check,
 ADD CONSTRAINT receipts_amount_check CHECK(amount>0),
 ADD CONSTRAINT receipt_target_evidence CHECK(connection_id IS NULL OR (provider='SEPAY' AND receiver_profile_id IS NOT NULL AND evidence_cipher IS NOT NULL AND api_observed_at IS NOT NULL));
CREATE UNIQUE INDEX receipt_connection_identity ON receipts(connection_id,transaction_id) WHERE connection_id IS NOT NULL;
-- Backfill legacy identity by the NULL-connection cohort; SePay uses connection scope.
ALTER TABLE receipts DROP CONSTRAINT receipts_provider_transaction_id_key;
CREATE UNIQUE INDEX receipt_legacy_identity ON receipts(provider,transaction_id) WHERE connection_id IS NULL;
CREATE TABLE payment_provider_observations (
 id uuid PRIMARY KEY, connection_id uuid NOT NULL REFERENCES payment_provider_connections(id), receiver_profile_id uuid NOT NULL REFERENCES payment_receiver_profiles(id),
 external_id text NOT NULL, reference_key text, normalized_hash text NOT NULL, raw_hash text NOT NULL, raw_cipher bytea NOT NULL,
 auth_key_version text NOT NULL, status text NOT NULL DEFAULT 'RECEIVED',
 linked_receipt uuid REFERENCES receipts(id), observed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 last_seen_at timestamptz NOT NULL DEFAULT clock_timestamp(), UNIQUE(connection_id,external_id)
);
CREATE INDEX payment_observation_reference ON payment_provider_observations(connection_id,reference_key);
CREATE INDEX payment_receipt_reference ON receipts(connection_id,reference_key);
CREATE TABLE payment_observation_conflicts (
 id uuid PRIMARY KEY, observation_id uuid NOT NULL REFERENCES payment_provider_observations(id),
 incoming_hash text NOT NULL, raw_cipher bytea NOT NULL, reason text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), UNIQUE(observation_id,incoming_hash)
);
CREATE TABLE payment_order_projections (
 order_id uuid PRIMARY KEY REFERENCES orders(id), payment_hold boolean NOT NULL DEFAULT false,
 has_open_reconciliation boolean NOT NULL DEFAULT false, verification_started_at timestamptz,
 verification_due_at timestamptz, support_reference text, support_reason text,
 version bigint NOT NULL DEFAULT 1 CHECK(version>0)
);
CREATE TABLE guest_checkout_sessions (
 sid uuid PRIMARY KEY, credential_hash text NOT NULL, issued_at timestamptz NOT NULL,
 expires_at timestamptz NOT NULL, revoked_at timestamptz, CHECK(expires_at>issued_at AND expires_at<=issued_at+interval '24 hours')
);
CREATE TABLE guest_payment_grant_versions (
 sid uuid NOT NULL REFERENCES guest_checkout_sessions(sid), order_id uuid NOT NULL REFERENCES orders(id),
 version bigint NOT NULL DEFAULT 0, PRIMARY KEY(sid,order_id)
);
CREATE TABLE guest_payment_grants (
 jti uuid PRIMARY KEY, sid uuid NOT NULL REFERENCES guest_checkout_sessions(sid), order_id uuid NOT NULL REFERENCES orders(id),
 grant_version bigint NOT NULL, expires_at timestamptz NOT NULL, session_expires_at timestamptz NOT NULL, revoked_at timestamptz, token_cipher bytea NOT NULL,
 UNIQUE(sid,order_id,grant_version), CHECK(expires_at<=session_expires_at)
);
ALTER TABLE reconciliation_cases ALTER COLUMN receipt_id DROP NOT NULL, DROP CONSTRAINT reconciliation_cases_receipt_id_key,
 ADD COLUMN observation_id uuid REFERENCES payment_provider_observations(id),
 ADD COLUMN evidence_hash text, ADD COLUMN evidence_cipher bytea,
 ADD CONSTRAINT reconciliation_source CHECK(receipt_id IS NOT NULL OR observation_id IS NOT NULL);
ALTER TABLE reconciliation_cases ADD COLUMN assigned_to uuid, ADD COLUMN version bigint NOT NULL DEFAULT 1,
 ADD COLUMN support_reference text NOT NULL DEFAULT gen_random_uuid()::text,
 ADD COLUMN resolution text, ADD COLUMN decision_reference text, ADD COLUMN resolution_cipher bytea;
CREATE UNIQUE INDEX reconciliation_receipt_reason ON reconciliation_cases(receipt_id,reason,COALESCE(evidence_hash,'')) WHERE receipt_id IS NOT NULL;
CREATE UNIQUE INDEX reconciliation_observation_reason ON reconciliation_cases(observation_id,reason,COALESCE(evidence_hash,'')) WHERE observation_id IS NOT NULL;
CREATE TABLE payment_sync_checkpoints (
 receiver_id uuid NOT NULL REFERENCES payment_receiver_profiles(id), stream text NOT NULL CHECK(stream IN ('FAST','OVERLAP','DAILY','BACKFILL')),
 bootstrap_at timestamptz NOT NULL, last_scanned_end timestamptz, last_fetch_at timestamptz,
 bank_reconciled_through timestamptz, circuit text NOT NULL DEFAULT 'BOOTSTRAP', PRIMARY KEY(receiver_id,stream)
);
CREATE TABLE payment_sync_runs (
 id uuid PRIMARY KEY, receiver_id uuid NOT NULL REFERENCES payment_receiver_profiles(id), stream text NOT NULL,
 job_id uuid NOT NULL REFERENCES jobs(id), lease_epoch bigint NOT NULL,
 window_start timestamptz NOT NULL, window_end timestamptz NOT NULL, last_durable_page integer NOT NULL DEFAULT 0,
 observed_count bigint NOT NULL DEFAULT 0, quarantine_count bigint NOT NULL DEFAULT 0,
 status text NOT NULL DEFAULT 'RUNNING', CHECK(window_start<=window_end)
);
CREATE UNIQUE INDEX payment_sync_active ON payment_sync_runs(receiver_id,stream) WHERE status='RUNNING';
CREATE TABLE payment_dispatch_quotas (egress_scope text PRIMARY KEY, next_request_at timestamptz NOT NULL);
CREATE TABLE payment_api_quarantine (
 run_id uuid NOT NULL REFERENCES payment_sync_runs(id), evidence_hash text NOT NULL, evidence_cipher bytea NOT NULL,
 reason text NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(), PRIMARY KEY(run_id,evidence_hash)
);
ALTER TABLE refunds ALTER COLUMN order_id DROP NOT NULL,
 ADD COLUMN execution_mode text NOT NULL DEFAULT 'LEGACY', ADD COLUMN execution_state text,
 ADD COLUMN budget_held boolean NOT NULL DEFAULT false, ADD COLUMN approval_cipher bytea,
 ADD COLUMN approval_hash text, ADD COLUMN approver_id uuid, ADD COLUMN executor_id uuid,
 ADD COLUMN claim_until timestamptz, ADD COLUMN current_attempt integer NOT NULL DEFAULT 0,
 ADD CONSTRAINT manual_refund_shape CHECK(execution_mode<>'MANUAL' OR
  (approval_cipher IS NOT NULL AND approval_hash IS NOT NULL AND approver_id IS NOT NULL AND execution_state IS NOT NULL));
ALTER TABLE command_registry ALTER COLUMN order_id DROP NOT NULL;
CREATE TABLE refund_execution_attempts (
 id uuid PRIMARY KEY, refund_id uuid NOT NULL REFERENCES refunds(id), attempt_no integer NOT NULL CHECK(attempt_no>0),
 executor_id uuid NOT NULL, operation_key text NOT NULL UNIQUE, request_hash text NOT NULL,
 started_at timestamptz NOT NULL DEFAULT clock_timestamp(), UNIQUE(refund_id,attempt_no)
);
CREATE TABLE payment_evidence (
 id uuid PRIMARY KEY, refund_id uuid NOT NULL REFERENCES refunds(id), attempt_id uuid NOT NULL REFERENCES refund_execution_attempts(id),
 uploader uuid NOT NULL, source text NOT NULL, sha256 text NOT NULL, evidence_cipher bytea NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE payment_evidence_uploads (
 id uuid PRIMARY KEY, resource_type text NOT NULL CHECK(resource_type IN ('REFUND','RECONCILIATION_CASE')),
 resource_id uuid NOT NULL, uploader uuid NOT NULL, sha256 text NOT NULL, media_type text NOT NULL,
 byte_size bigint NOT NULL CHECK(byte_size BETWEEN 1 AND 10485760), expires_at timestamptz NOT NULL
);
CREATE TABLE refund_verifications (
 id uuid PRIMARY KEY, refund_id uuid NOT NULL REFERENCES refunds(id), attempt_id uuid NOT NULL REFERENCES refund_execution_attempts(id),
 verifier_id uuid NOT NULL, result text NOT NULL CHECK(result IN ('SENT','NOT_SENT','INCONCLUSIVE')),
 bank_reference text, evidence_id uuid NOT NULL REFERENCES payment_evidence(id), verified_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE refund_bank_dispositions (
 connection_id uuid NOT NULL REFERENCES payment_provider_connections(id), bank_identity text NOT NULL,
 refund_id uuid NOT NULL UNIQUE REFERENCES refunds(id), PRIMARY KEY(connection_id,bank_identity)
);
CREATE TRIGGER refund_attempt_immutable BEFORE UPDATE OR DELETE ON refund_execution_attempts FOR EACH ROW EXECUTE FUNCTION immutable_audit();
CREATE TRIGGER payment_evidence_immutable BEFORE UPDATE OR DELETE ON payment_evidence FOR EACH ROW EXECUTE FUNCTION immutable_audit();
CREATE TRIGGER refund_verification_immutable BEFORE UPDATE OR DELETE ON refund_verifications FOR EACH ROW EXECUTE FUNCTION immutable_audit();
CREATE FUNCTION protect_payment_receipt() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.connection_id IS NOT NULL AND ROW(NEW.id,NEW.provider,NEW.transaction_id,NEW.connection_id,NEW.receiver_profile_id,NEW.order_id,NEW.amount,NEW.currency,NEW.payload_hash,NEW.provider_paid_at,NEW.evidence_cipher,NEW.reference_key)
 IS DISTINCT FROM ROW(OLD.id,OLD.provider,OLD.transaction_id,OLD.connection_id,OLD.receiver_profile_id,OLD.order_id,OLD.amount,OLD.currency,OLD.payload_hash,OLD.provider_paid_at,OLD.evidence_cipher,OLD.reference_key)
 THEN RAISE EXCEPTION 'immutable financial receipt'; END IF; RETURN NEW;
END $$;
CREATE TRIGGER payment_receipt_immutable BEFORE UPDATE ON receipts FOR EACH ROW EXECUTE FUNCTION protect_payment_receipt();
CREATE TRIGGER payment_conflict_immutable BEFORE UPDATE OR DELETE ON payment_observation_conflicts FOR EACH ROW EXECUTE FUNCTION immutable_audit();
CREATE FUNCTION protect_payment_receiver() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF ROW(NEW.id,NEW.connection_id,NEW.bank_account_uuid,NEW.config_version,NEW.config_cipher)
 IS DISTINCT FROM ROW(OLD.id,OLD.connection_id,OLD.bank_account_uuid,OLD.config_version,OLD.config_cipher)
 THEN RAISE EXCEPTION 'immutable receiver profile'; END IF; RETURN NEW;
END $$;
CREATE TRIGGER payment_receiver_immutable BEFORE UPDATE ON payment_receiver_profiles FOR EACH ROW EXECUTE FUNCTION protect_payment_receiver();
CREATE FUNCTION protect_payment_observation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF ROW(NEW.id,NEW.connection_id,NEW.receiver_profile_id,NEW.external_id,NEW.normalized_hash,NEW.raw_hash,NEW.raw_cipher,NEW.auth_key_version,NEW.reference_key)
 IS DISTINCT FROM ROW(OLD.id,OLD.connection_id,OLD.receiver_profile_id,OLD.external_id,OLD.normalized_hash,OLD.raw_hash,OLD.raw_cipher,OLD.auth_key_version,OLD.reference_key)
 THEN RAISE EXCEPTION 'immutable payment observation'; END IF; RETURN NEW;
END $$;
CREATE TRIGGER payment_observation_immutable BEFORE UPDATE ON payment_provider_observations FOR EACH ROW EXECUTE FUNCTION protect_payment_observation();
