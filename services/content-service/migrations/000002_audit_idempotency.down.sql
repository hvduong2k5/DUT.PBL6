BEGIN;
DROP TABLE idempotency_keys;
DROP TRIGGER immutable_published_status ON article_revisions;
DROP FUNCTION protect_published_status();
DROP TABLE audit_events;
DROP FUNCTION forbid_audit_change();
COMMIT;
