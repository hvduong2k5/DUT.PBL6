BEGIN;
ALTER TABLE articles DROP CONSTRAINT published_revision_owner;
ALTER TABLE articles DROP CONSTRAINT current_revision_owner;
DROP TABLE slug_history;
DROP TABLE approval_decisions;
DROP TABLE article_revisions;
DROP TABLE articles;
DROP FUNCTION protect_revision_snapshot();
COMMIT;
