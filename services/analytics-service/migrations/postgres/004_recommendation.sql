CREATE SCHEMA analytics_recommendation;
SET LOCAL search_path = analytics_recommendation, public;

CREATE TABLE source_snapshot (
    snapshot_id uuid PRIMARY KEY,
    scope_id text NOT NULL,
    dataset text NOT NULL CHECK (dataset IN ('catalog','sellability','completed_baskets')),
    source_revision bigint NOT NULL CHECK (source_revision > 0),
    source_as_of timestamptz NOT NULL,
    manifest jsonb NOT NULL,
    records jsonb NOT NULL CHECK (jsonb_typeof(records)='array'),
    digest text NOT NULL,
    received_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(scope_id,dataset,source_revision),
    UNIQUE(scope_id,dataset,snapshot_id)
);
CREATE TABLE source_pointer (
    scope_id text NOT NULL,
    dataset text NOT NULL,
    snapshot_id uuid NOT NULL,
    PRIMARY KEY(scope_id,dataset),
    FOREIGN KEY(scope_id,dataset,snapshot_id) REFERENCES source_snapshot(scope_id,dataset,snapshot_id)
);
CREATE TABLE consent (
    subject_key text PRIMARY KEY,
    enabled boolean NOT NULL DEFAULT false,
    version bigint NOT NULL DEFAULT 0 CHECK(version>=0),
    policy_version text NOT NULL DEFAULT 'personalization-v1',
    granted_at timestamptz,
    revoked_at timestamptz
);
CREATE TABLE consent_request (
    subject_key text NOT NULL REFERENCES consent,
    request_key uuid NOT NULL,
    digest text NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY(subject_key,request_key)
);
CREATE TABLE privacy_state (
    id integer PRIMARY KEY CHECK(id=1),
    epoch bigint NOT NULL DEFAULT 0
);
INSERT INTO privacy_state(id) VALUES(1);
CREATE TABLE erasure (
    subject_key text PRIMARY KEY,
    requested_at timestamptz NOT NULL,
    through_version bigint NOT NULL
);
CREATE TABLE behavior_event (
    subject_key text NOT NULL REFERENCES consent,
    event_id uuid NOT NULL,
    scope_id text NOT NULL,
    consent_version bigint NOT NULL,
    digest text NOT NULL,
    event_type text NOT NULL CHECK(event_type IN ('VIEW_ITEM','ADD_TO_CART')),
    sku_code text NOT NULL,
    occurred_at timestamptz NOT NULL,
    effective_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL,
    recommendation_id uuid,
    PRIMARY KEY(subject_key,event_id)
);
CREATE INDEX behavior_history ON behavior_event(scope_id,subject_key,effective_at);
CREATE TABLE generation (
    generation_id uuid PRIMARY KEY,
    scope_id text NOT NULL,
    input_digest text NOT NULL,
    privacy_epoch bigint NOT NULL,
    built_at timestamptz NOT NULL,
    sources jsonb NOT NULL,
    payload jsonb NOT NULL,
    UNIQUE(scope_id,generation_id)
);
CREATE TABLE serving_pointer (
    scope_id text PRIMARY KEY,
    generation_id uuid NOT NULL,
    FOREIGN KEY(scope_id,generation_id) REFERENCES generation(scope_id,generation_id)
);
CREATE TABLE decision_trace (
    recommendation_id uuid PRIMARY KEY,
    scope_id text NOT NULL,
    placement text NOT NULL,
    generation_id uuid,
    strategy text NOT NULL,
    items jsonb NOT NULL,
    decided_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
