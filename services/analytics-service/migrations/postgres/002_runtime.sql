SET LOCAL search_path = analytics_config, public;

ALTER TABLE order_milestone
    ADD COLUMN submicro_ns smallint NOT NULL DEFAULT 0 CHECK (submicro_ns BETWEEN 0 AND 999);

ALTER TABLE partition_cursor
    ADD COLUMN bootstrap_offset bigint NOT NULL CHECK (bootstrap_offset >= 0),
    ADD COLUMN last_record_digest bytea CHECK (octet_length(last_record_digest) = 32);

ALTER TABLE serving_generation
    ADD COLUMN input_digest bytea CHECK (octet_length(input_digest) = 32),
    ADD COLUMN purged_at timestamptz;

CREATE TABLE coverage_approval (
    approval_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pipeline text NOT NULL,
    business_date date NOT NULL,
    topic_identity text NOT NULL,
    cutoffs jsonb NOT NULL,
    paid_count bigint NOT NULL CHECK (paid_count >= 0),
    completed_count bigint NOT NULL CHECK (completed_count >= 0),
    source_as_of timestamptz NOT NULL,
    approved_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    approved_by text NOT NULL,
    evidence_ref text NOT NULL,
    invalidated_at timestamptz,
    invalidation_reason text
);

CREATE INDEX coverage_by_day ON coverage_approval (pipeline, business_date, approval_id DESC);
