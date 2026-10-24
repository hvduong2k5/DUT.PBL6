CREATE TABLE service_metadata (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    mode text NOT NULL CHECK (mode IN ('sandbox', 'integration'))
);

CREATE TABLE traceability_profiles (
    id uuid PRIMARY KEY,
    mode text NOT NULL CHECK (mode IN ('sandbox', 'integration')),
    product_id uuid NOT NULL,
    sku_code varchar(100) NOT NULL CHECK (length(btrim(sku_code)) > 0),
    source_namespace varchar(100) NOT NULL CHECK (length(btrim(source_namespace)) > 0),
    batch_id varchar(160) NOT NULL CHECK (length(btrim(batch_id)) > 0),
    qr_code varchar(160) NOT NULL UNIQUE CHECK (qr_code ~ '^[A-Za-z0-9_-]{22,160}$'),
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUSPENDED','ARCHIVED')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    public_revision integer,
    ever_published boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(source_namespace, batch_id)
);

CREATE TABLE traceability_revisions (
    profile_id uuid NOT NULL REFERENCES traceability_profiles(id),
    number integer NOT NULL CHECK (number > 0),
    status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','IN_REVIEW','APPROVED','REJECTED','PUBLISHED','ARCHIVED')),
    content jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(content) = 'object'),
    dependency_snapshot jsonb CHECK (dependency_snapshot IS NULL OR jsonb_typeof(dependency_snapshot) = 'object'),
    contributors uuid[] NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (profile_id, number),
    CHECK (cardinality(contributors) > 0 AND created_by = ANY(contributors) AND array_position(contributors, NULL) IS NULL)
);
CREATE UNIQUE INDEX one_working_revision ON traceability_revisions(profile_id) WHERE status IN ('DRAFT','IN_REVIEW','APPROVED');
CREATE UNIQUE INDEX one_published_revision ON traceability_revisions(profile_id) WHERE status = 'PUBLISHED';
ALTER TABLE traceability_profiles ADD CONSTRAINT public_revision_fk
    FOREIGN KEY (id, public_revision) REFERENCES traceability_revisions(profile_id, number) DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX profile_pagination ON traceability_profiles(created_at, id);

CREATE TABLE review_decisions (
    id uuid PRIMARY KEY,
    profile_id uuid NOT NULL,
    revision_number integer NOT NULL,
    actor_id uuid NOT NULL,
    decision text NOT NULL CHECK (decision IN ('APPROVE','REJECT')),
    reason text NOT NULL CHECK (length(btrim(reason)) BETWEEN 1 AND 1000),
    profile_version bigint NOT NULL CHECK (profile_version > 0),
    decided_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (profile_id, revision_number) REFERENCES traceability_revisions(profile_id, number),
    UNIQUE(profile_id, revision_number)
);

CREATE TABLE audit_entries (
    id uuid PRIMARY KEY,
    profile_id uuid NOT NULL REFERENCES traceability_profiles(id),
    revision_number integer,
    actor_id uuid NOT NULL,
    action text NOT NULL,
    reason text NOT NULL DEFAULT '',
    before_state jsonb NOT NULL CHECK (jsonb_typeof(before_state) = 'object'),
    after_state jsonb NOT NULL CHECK (jsonb_typeof(after_state) = 'object'),
    request_id text NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_profile_time ON audit_entries(profile_id, occurred_at, id);

CREATE TABLE outbox_events (
    id uuid PRIMARY KEY,
    profile_id uuid NOT NULL REFERENCES traceability_profiles(id),
    aggregate_version bigint NOT NULL CHECK (aggregate_version > 0),
    event_type text NOT NULL,
    topic text NOT NULL,
    partition_key text NOT NULL,
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','PUBLISHED','FAILED')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    lease_owner uuid,
    lease_until timestamptz,
    retry_at timestamptz NOT NULL DEFAULT now(),
    error_code text,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    UNIQUE(profile_id, aggregate_version, event_type),
    CHECK ((status = 'PUBLISHED') = (published_at IS NOT NULL)),
    CHECK ((lease_owner IS NULL) = (lease_until IS NULL))
);
CREATE INDEX outbox_pending ON outbox_events(retry_at, created_at) WHERE status = 'PENDING';

CREATE TABLE idempotency_records (
    principal_id uuid NOT NULL,
    action text NOT NULL,
    key varchar(128) NOT NULL CHECK (key ~ '^[A-Za-z0-9._~-]{1,128}$'),
    fingerprint char(64) NOT NULL CHECK (fingerprint ~ '^[a-f0-9]{64}$'),
    status_code integer NOT NULL CHECK (status_code BETWEEN 200 AND 299),
    response jsonb NOT NULL,
    resource_id uuid NOT NULL REFERENCES traceability_profiles(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(principal_id, action, key)
);

CREATE TABLE eligibility_observations (
    profile_id uuid NOT NULL REFERENCES traceability_profiles(id),
    public_revision integer NOT NULL,
    reason text NOT NULL,
    observed_state text NOT NULL,
    cutoff timestamptz NOT NULL,
    observed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(profile_id, public_revision, reason, observed_state, cutoff),
    FOREIGN KEY(profile_id, public_revision) REFERENCES traceability_revisions(profile_id, number)
);

CREATE FUNCTION reject_history_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'history is append only' USING ERRCODE = '23514';
END $$;
CREATE TRIGGER audit_append_only BEFORE UPDATE OR DELETE ON audit_entries FOR EACH ROW EXECUTE FUNCTION reject_history_mutation();
CREATE TRIGGER decision_append_only BEFORE UPDATE OR DELETE ON review_decisions FOR EACH ROW EXECUTE FUNCTION reject_history_mutation();
CREATE TRIGGER idempotency_append_only BEFORE UPDATE OR DELETE ON idempotency_records FOR EACH ROW EXECUTE FUNCTION reject_history_mutation();
CREATE TRIGGER profile_no_delete BEFORE DELETE ON traceability_profiles FOR EACH ROW EXECUTE FUNCTION reject_history_mutation();
CREATE TRIGGER revision_no_delete BEFORE DELETE ON traceability_revisions FOR EACH ROW EXECUTE FUNCTION reject_history_mutation();
CREATE TRIGGER metadata_immutable BEFORE UPDATE OR DELETE ON service_metadata FOR EACH ROW EXECUTE FUNCTION reject_history_mutation();

CREATE FUNCTION protect_profile() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE database_mode text;
BEGIN
    SELECT mode INTO database_mode FROM service_metadata WHERE singleton;
    IF database_mode IS NULL OR NEW.mode IS DISTINCT FROM database_mode THEN
        RAISE EXCEPTION 'profile mode does not match database mode' USING ERRCODE = '23514';
    END IF;
    IF TG_OP = 'UPDATE' THEN
        IF ROW(NEW.id,NEW.mode,NEW.product_id,NEW.sku_code,NEW.source_namespace,NEW.batch_id,NEW.qr_code,NEW.created_at)
            IS DISTINCT FROM ROW(OLD.id,OLD.mode,OLD.product_id,OLD.sku_code,OLD.source_namespace,OLD.batch_id,OLD.qr_code,OLD.created_at) THEN
            RAISE EXCEPTION 'binding is immutable' USING ERRCODE = '23514';
        END IF;
        IF NEW.version <> OLD.version + 1 THEN
            RAISE EXCEPTION 'profile version must advance exactly once' USING ERRCODE = '23514';
        END IF;
        IF (OLD.status = 'ARCHIVED' AND NEW.status <> 'ARCHIVED') OR (OLD.status = 'SUSPENDED' AND NEW.status = 'ACTIVE')
            OR (OLD.ever_published AND NOT NEW.ever_published) THEN
            RAISE EXCEPTION 'profile cannot be restored or erase publication history' USING ERRCODE = '23514';
        END IF;
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER profile_protection BEFORE INSERT OR UPDATE ON traceability_profiles FOR EACH ROW EXECUTE FUNCTION protect_profile();

CREATE FUNCTION protect_revision() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE root_status text;
BEGIN
    SELECT status INTO root_status FROM traceability_profiles WHERE id=NEW.profile_id;
    IF root_status <> 'ACTIVE' THEN
        RAISE EXCEPTION 'profile is not active' USING ERRCODE='23514';
    END IF;
    IF TG_OP='INSERT' THEN
        IF NEW.status <> 'DRAFT' THEN RAISE EXCEPTION 'revision must start as draft' USING ERRCODE='23514'; END IF;
    ELSE
        IF ROW(NEW.profile_id,NEW.number,NEW.created_by,NEW.created_at) IS DISTINCT FROM ROW(OLD.profile_id,OLD.number,OLD.created_by,OLD.created_at) THEN
            RAISE EXCEPTION 'revision identity is immutable' USING ERRCODE='23514';
        END IF;
        IF NOT OLD.contributors <@ NEW.contributors THEN RAISE EXCEPTION 'contributors cannot be removed' USING ERRCODE='23514'; END IF;
        IF OLD.status <> 'DRAFT' AND ROW(NEW.content,NEW.dependency_snapshot,NEW.contributors) IS DISTINCT FROM ROW(OLD.content,OLD.dependency_snapshot,OLD.contributors) THEN
            RAISE EXCEPTION 'submitted content is immutable' USING ERRCODE='23514';
        END IF;
        IF NEW.status <> OLD.status AND NOT (
            (OLD.status='DRAFT' AND NEW.status IN ('IN_REVIEW','ARCHIVED')) OR
            (OLD.status='IN_REVIEW' AND NEW.status IN ('APPROVED','REJECTED','ARCHIVED')) OR
            (OLD.status='APPROVED' AND NEW.status IN ('PUBLISHED','ARCHIVED')) OR
            (OLD.status='PUBLISHED' AND NEW.status='ARCHIVED')) THEN
            RAISE EXCEPTION 'invalid revision transition' USING ERRCODE='23514';
        END IF;
    END IF;
    IF NEW.status IN ('IN_REVIEW','APPROVED','PUBLISHED') AND NEW.dependency_snapshot IS NULL THEN
        RAISE EXCEPTION 'verification snapshot is required' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER revision_protection BEFORE INSERT OR UPDATE ON traceability_revisions FOR EACH ROW EXECUTE FUNCTION protect_revision();

CREATE FUNCTION validate_decision() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE r traceability_revisions;
BEGIN
    SELECT * INTO r FROM traceability_revisions WHERE profile_id=NEW.profile_id AND number=NEW.revision_number FOR UPDATE;
    IF r.status <> 'IN_REVIEW' OR NEW.actor_id=ANY(r.contributors) THEN
        RAISE EXCEPTION 'review must be independent and in review' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER decision_validation BEFORE INSERT ON review_decisions FOR EACH ROW EXECUTE FUNCTION validate_decision();

-- The decision and state transition must commit together. Handler checks alone
-- cannot prevent an accidental repository update from bypassing review.
CREATE FUNCTION check_review_consistency() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE pid uuid; rev integer; state text; verdict text;
BEGIN
    pid := NEW.profile_id;
    IF TG_TABLE_NAME='review_decisions' THEN rev:=NEW.revision_number; ELSE rev:=NEW.number; END IF;
    SELECT status INTO state FROM traceability_revisions WHERE profile_id=pid AND number=rev;
    SELECT decision INTO verdict FROM review_decisions WHERE profile_id=pid AND revision_number=rev;
    IF (state IN ('APPROVED','PUBLISHED') AND verdict IS DISTINCT FROM 'APPROVE')
        OR (state='REJECTED' AND verdict IS DISTINCT FROM 'REJECT')
        OR (verdict='APPROVE' AND state NOT IN ('APPROVED','PUBLISHED','ARCHIVED'))
        OR (verdict='REJECT' AND state NOT IN ('REJECTED','ARCHIVED')) THEN
        RAISE EXCEPTION 'review decision and revision state must commit together' USING ERRCODE='23514';
    END IF;
    RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER revision_review_consistency AFTER INSERT OR UPDATE ON traceability_revisions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION check_review_consistency();
CREATE CONSTRAINT TRIGGER decision_review_consistency AFTER INSERT ON review_decisions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION check_review_consistency();

CREATE FUNCTION check_public_pointer() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE pid uuid; p traceability_profiles; published integer;
BEGIN
    IF TG_TABLE_NAME='traceability_profiles' THEN pid:=NEW.id; ELSE pid:=NEW.profile_id; END IF;
    SELECT * INTO p FROM traceability_profiles WHERE id=pid;
    SELECT number INTO published FROM traceability_revisions WHERE profile_id=pid AND status='PUBLISHED';
    IF published IS DISTINCT FROM p.public_revision OR ((p.public_revision IS NOT NULL) AND NOT p.ever_published)
        OR (p.ever_published AND p.public_revision IS NULL) THEN
        RAISE EXCEPTION 'public pointer must match the sole published revision' USING ERRCODE='23514';
    END IF;
    RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER profile_public_pointer AFTER INSERT OR UPDATE ON traceability_profiles DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION check_public_pointer();
CREATE CONSTRAINT TRIGGER revision_public_pointer AFTER INSERT OR UPDATE ON traceability_revisions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION check_public_pointer();

CREATE FUNCTION protect_outbox_payload() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF ROW(NEW.id,NEW.profile_id,NEW.aggregate_version,NEW.event_type,NEW.topic,NEW.partition_key,NEW.payload,NEW.created_at)
        IS DISTINCT FROM ROW(OLD.id,OLD.profile_id,OLD.aggregate_version,OLD.event_type,OLD.topic,OLD.partition_key,OLD.payload,OLD.created_at) THEN
        RAISE EXCEPTION 'event identity and payload are immutable' USING ERRCODE='23514';
    END IF;
    IF OLD.status='PUBLISHED' AND NEW.status <> 'PUBLISHED' THEN
        RAISE EXCEPTION 'published event cannot be reset' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER outbox_payload_protection BEFORE UPDATE ON outbox_events FOR EACH ROW EXECUTE FUNCTION protect_outbox_payload();
