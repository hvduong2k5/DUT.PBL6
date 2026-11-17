CREATE TABLE campaign_execution_metrics (
    execution_id UUID NOT NULL REFERENCES campaign_executions(id),
    metric_date DATE NOT NULL,
    source_version BIGINT NOT NULL CHECK (source_version BETWEEN 1 AND 9007199254740991),
    impressions BIGINT NOT NULL CHECK (impressions BETWEEN 0 AND 9007199254740991),
    clicks BIGINT NOT NULL CHECK (clicks BETWEEN 0 AND 9007199254740991),
    actor_id UUID NOT NULL,
    request_id TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (execution_id, metric_date, source_version)
);
