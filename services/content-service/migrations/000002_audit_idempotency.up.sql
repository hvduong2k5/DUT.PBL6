BEGIN;
-- Append-only trail of every editorial action (create, edit, submit, approve,
-- reject, publish, hide, restore).
CREATE TABLE audit_events (
    id UUID PRIMARY KEY,
    article_id UUID NOT NULL REFERENCES articles(id),
    revision_id UUID NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('create','edit','submit','approve','reject','publish','hide','restore')),
    actor_id UUID NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (article_id,revision_id) REFERENCES article_revisions(article_id,id)
);
CREATE INDEX audit_article ON audit_events(article_id,created_at,id);
CREATE FUNCTION forbid_audit_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit events are append-only' USING ERRCODE = '23514';
END $$;
CREATE TRIGGER audit_append_only BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION forbid_audit_change();

-- A published snapshot never changes status; hide only flips articles.visible.
CREATE FUNCTION protect_published_status() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'published revision status is immutable' USING ERRCODE = '23514';
END $$;
CREATE TRIGGER immutable_published_status BEFORE UPDATE OF status ON article_revisions
    FOR EACH ROW WHEN (OLD.status = 'PUBLISHED' AND NEW.status IS DISTINCT FROM OLD.status)
    EXECUTE FUNCTION protect_published_status();

-- Idempotency-Key replay store, scoped to the verified principal.
CREATE TABLE idempotency_keys (
    principal_id UUID NOT NULL,
    key TEXT NOT NULL CHECK (length(key) BETWEEN 1 AND 255),
    request_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('IN_PROGRESS','COMPLETED')),
    response_status INTEGER,
    response_headers JSONB,
    response_body BYTEA,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (principal_id,key)
);
CREATE INDEX idempotency_created ON idempotency_keys(created_at);
COMMIT;
