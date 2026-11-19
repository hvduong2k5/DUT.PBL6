-- PostgreSQL design candidate for Dashboard v0.1.
BEGIN;

CREATE SCHEMA analytics_config;
SET LOCAL search_path = analytics_config, public;

CREATE TABLE partition_cursor (
    pipeline text NOT NULL,
    topic text NOT NULL,
    partition_id integer NOT NULL CHECK (partition_id >= 0),
    topic_identity text NOT NULL,
    next_offset bigint NOT NULL CHECK (next_offset >= 0),
    lease_epoch bigint NOT NULL DEFAULT 0 CHECK (lease_epoch >= 0),
    lease_owner text,
    lease_until timestamptz,
    gap_code text,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (pipeline, topic, partition_id),
    CHECK ((lease_owner IS NULL) = (lease_until IS NULL))
);

CREATE TABLE source_event (
    pipeline text NOT NULL,
    source text NOT NULL,
    event_id text NOT NULL,
    event_type text NOT NULL,
    adapter_version text NOT NULL,
    normalized_digest bytea NOT NULL CHECK (octet_length(normalized_digest) = 32),
    occurred_at timestamptz NOT NULL,
    ingested_at timestamptz NOT NULL,
    topic_identity text NOT NULL,
    topic text NOT NULL,
    partition_id integer NOT NULL CHECK (partition_id >= 0),
    source_offset bigint NOT NULL CHECK (source_offset >= 0),
    PRIMARY KEY (pipeline, source, event_id),
    UNIQUE (pipeline, topic_identity, partition_id, source_offset)
);

CREATE TABLE order_milestone (
    pipeline text NOT NULL,
    cohort text NOT NULL,
    order_id uuid NOT NULL,
    milestone text NOT NULL CHECK (milestone IN ('PAID', 'COMPLETED')),
    occurred_at timestamptz NOT NULL,
    order_version bigint NOT NULL CHECK (order_version > 0),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    source text NOT NULL,
    event_id text NOT NULL,
    PRIMARY KEY (pipeline, cohort, order_id, milestone),
    UNIQUE (pipeline, source, event_id),
    FOREIGN KEY (pipeline, source, event_id) REFERENCES source_event
);

CREATE INDEX order_milestone_by_period
    ON order_milestone (pipeline, cohort, occurred_at, order_id, milestone);

CREATE TABLE source_quarantine (
    pipeline text NOT NULL,
    topic_identity text NOT NULL,
    topic text NOT NULL,
    partition_id integer NOT NULL CHECK (partition_id >= 0),
    source_offset bigint NOT NULL CHECK (source_offset >= 0),
    reason_code text NOT NULL,
    record_digest bytea NOT NULL CHECK (octet_length(record_digest) = 32),
    detected_at timestamptz NOT NULL,
    resolved_at timestamptz,
    resolution_ref text,
    PRIMARY KEY (pipeline, topic_identity, partition_id, source_offset),
    CHECK ((resolved_at IS NULL) = (resolution_ref IS NULL))
);

CREATE TABLE metric_definition (
    metric_id text NOT NULL,
    definition_version text NOT NULL,
    unit text NOT NULL,
    owner text NOT NULL,
    policy jsonb NOT NULL CHECK (jsonb_typeof(policy) = 'object'),
    effective_from timestamptz NOT NULL,
    approval_ref text,
    PRIMARY KEY (metric_id, definition_version)
);

CREATE TABLE serving_generation (
    generation_id uuid PRIMARY KEY,
    scope_key text NOT NULL,
    pipeline text NOT NULL,
    expected_generation_id uuid,
    state text NOT NULL CHECK (state IN ('BUILDING', 'PUBLISHED', 'ABANDONED', 'RETIRED')),
    created_at timestamptz NOT NULL,
    published_at timestamptz,
    serve_until timestamptz,
    fact_count bigint CHECK (fact_count >= 0),
    fact_digest bytea CHECK (octet_length(fact_digest) = 32),
    UNIQUE (scope_key, generation_id),
    CHECK (state <> 'PUBLISHED' OR
        (published_at IS NOT NULL AND fact_count IS NOT NULL AND fact_digest IS NOT NULL))
);

CREATE TABLE generation_cutoff (
    generation_id uuid NOT NULL REFERENCES serving_generation,
    topic_identity text NOT NULL,
    topic text NOT NULL,
    partition_id integer NOT NULL CHECK (partition_id >= 0),
    next_offset bigint NOT NULL CHECK (next_offset >= 0),
    PRIMARY KEY (generation_id, topic_identity, partition_id)
);

CREATE TABLE generation_metric (
    generation_id uuid NOT NULL REFERENCES serving_generation,
    metric_id text NOT NULL,
    definition_version text NOT NULL,
    source_as_of timestamptz,
    coverage jsonb NOT NULL CHECK (jsonb_typeof(coverage) = 'object'),
    PRIMARY KEY (generation_id, metric_id),
    FOREIGN KEY (metric_id, definition_version) REFERENCES metric_definition
);

CREATE TABLE serving_pointer (
    scope_key text PRIMARY KEY,
    generation_id uuid NOT NULL,
    updated_at timestamptz NOT NULL,
    FOREIGN KEY (scope_key, generation_id)
        REFERENCES serving_generation (scope_key, generation_id)
);

COMMIT;
