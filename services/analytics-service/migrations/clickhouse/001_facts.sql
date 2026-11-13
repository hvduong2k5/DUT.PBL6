-- ClickHouse design candidate for Dashboard v0.1.
CREATE DATABASE IF NOT EXISTS analytics_db;

CREATE TABLE IF NOT EXISTS analytics_db.order_milestone_fact
(
    generation_id UUID,
    cohort LowCardinality(String),
    order_id UUID,
    milestone Enum8('PAID' = 1, 'COMPLETED' = 2),
    occurred_at DateTime64(6, 'UTC'),
    business_date Date,
    source_event_id String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(business_date)
ORDER BY (generation_id, cohort, milestone, business_date, order_id);

CREATE TABLE IF NOT EXISTS analytics_db.order_milestone_daily
(
    generation_id UUID,
    cohort LowCardinality(String),
    business_date Date,
    milestone Enum8('PAID' = 1, 'COMPLETED' = 2),
    order_count UInt64
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(business_date)
ORDER BY (generation_id, cohort, milestone, business_date);
