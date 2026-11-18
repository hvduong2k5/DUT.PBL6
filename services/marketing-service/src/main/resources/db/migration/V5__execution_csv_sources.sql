CREATE TABLE campaign_execution_csv_sources (
    execution_id UUID PRIMARY KEY REFERENCES campaign_executions(id),
    external_key VARCHAR(160) NOT NULL UNIQUE
);
