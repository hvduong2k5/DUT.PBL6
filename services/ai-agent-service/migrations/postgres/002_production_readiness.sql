ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS guest_token_hash TEXT;

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS idempotency_owner TEXT;

ALTER TABLE conversations
    DROP CONSTRAINT IF EXISTS conversations_guest_token_check;

ALTER TABLE conversations
    ADD CONSTRAINT conversations_guest_token_check CHECK (
        (owner_type = 'guest' AND guest_token_hash IS NOT NULL)
        OR (owner_type = 'customer' AND guest_token_hash IS NULL)
    ) NOT VALID;

CREATE INDEX IF NOT EXISTS idx_conversations_expires_at ON conversations (expires_at);

CREATE UNIQUE INDEX IF NOT EXISTS uq_conversations_idempotency
    ON conversations (idempotency_owner, idempotency_key)
    WHERE idempotency_owner IS NOT NULL AND idempotency_key IS NOT NULL;

ALTER TABLE recommendation_feedback
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

ALTER TABLE recommendation_feedback
    ADD COLUMN IF NOT EXISTS idempotency_owner TEXT;

ALTER TABLE recommendation_feedback
    ADD COLUMN IF NOT EXISTS request_hash TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_recommendation_feedback_idempotency
    ON recommendation_feedback (idempotency_owner, idempotency_key)
    WHERE idempotency_owner IS NOT NULL AND idempotency_key IS NOT NULL;
