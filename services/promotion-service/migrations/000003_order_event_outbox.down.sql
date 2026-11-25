DROP TABLE IF EXISTS promotion_consumed_events;
ALTER TABLE outbox_events
    DROP COLUMN IF EXISTS locked_until,
    DROP COLUMN IF EXISTS partition_key,
    DROP COLUMN IF EXISTS topic;
