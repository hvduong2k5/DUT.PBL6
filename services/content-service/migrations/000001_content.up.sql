BEGIN;
CREATE TABLE articles (
    id UUID PRIMARY KEY,
    version BIGINT NOT NULL CHECK (version >= 1),
    current_revision_id UUID NOT NULL,
    published_revision_id UUID,
    visible BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (NOT visible OR published_revision_id IS NOT NULL)
);
CREATE TABLE article_revisions (
    id UUID PRIMARY KEY,
    article_id UUID NOT NULL REFERENCES articles(id),
    revision_number INTEGER NOT NULL CHECK (revision_number > 0),
    status TEXT NOT NULL CHECK (status IN ('DRAFT','PENDING','APPROVED','REJECTED','PUBLISHED')),
    title VARCHAR(300) NOT NULL CHECK (length(trim(title)) > 0),
    slug VARCHAR(200) NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    body TEXT NOT NULL CHECK (length(trim(body)) > 0),
    category VARCHAR(100) NOT NULL DEFAULT '',
    seo JSONB NOT NULL CHECK (jsonb_typeof(seo) = 'object'),
    media JSONB NOT NULL CHECK (jsonb_typeof(media) = 'array'),
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE(article_id, revision_number),
    UNIQUE(article_id, id)
);
ALTER TABLE articles ADD CONSTRAINT current_revision_owner
    FOREIGN KEY (id, current_revision_id) REFERENCES article_revisions(article_id,id)
    DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE articles ADD CONSTRAINT published_revision_owner
    FOREIGN KEY (id, published_revision_id) REFERENCES article_revisions(article_id,id)
    DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE approval_decisions (
    id UUID PRIMARY KEY,
    article_id UUID NOT NULL REFERENCES articles(id),
    revision_id UUID NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('APPROVED','REJECTED')),
    reason TEXT NOT NULL DEFAULT '',
    actor_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (article_id,revision_id) REFERENCES article_revisions(article_id,id),
    CHECK (status <> 'REJECTED' OR length(trim(reason)) > 0)
);
-- Every used slug is reserved globally, even after a URL changes.
CREATE TABLE slug_history (
    slug VARCHAR(200) PRIMARY KEY,
    article_id UUID NOT NULL REFERENCES articles(id),
    created_at TIMESTAMPTZ NOT NULL,
    CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$')
);
CREATE INDEX articles_public_page ON articles(created_at DESC,id DESC) WHERE visible;
CREATE INDEX articles_admin_page ON articles(created_at DESC,id DESC);
CREATE INDEX revisions_slug ON article_revisions(slug);
CREATE INDEX revisions_filter ON article_revisions(status,category);
CREATE INDEX decisions_article ON approval_decisions(article_id,created_at,id);
CREATE INDEX slugs_article ON slug_history(article_id);

CREATE FUNCTION protect_revision_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF ROW(NEW.id,NEW.article_id,NEW.revision_number,NEW.title,NEW.slug,NEW.body,
           NEW.category,NEW.seo,NEW.media,NEW.created_by,NEW.created_at)
       IS DISTINCT FROM
       ROW(OLD.id,OLD.article_id,OLD.revision_number,OLD.title,OLD.slug,OLD.body,
           OLD.category,OLD.seo,OLD.media,OLD.created_by,OLD.created_at) THEN
        RAISE EXCEPTION 'revision snapshots are immutable' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER immutable_revision BEFORE UPDATE ON article_revisions
    FOR EACH ROW EXECUTE FUNCTION protect_revision_snapshot();
COMMIT;
