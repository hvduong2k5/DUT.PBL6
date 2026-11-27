CREATE TABLE media_bind_jobs (
    profile_id uuid NOT NULL REFERENCES traceability_profiles(id),
    revision integer NOT NULL,
    file_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    retry_at timestamptz NOT NULL DEFAULT now(),
    lease_token uuid,
    lease_until timestamptz,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    bound_at timestamptz,
    PRIMARY KEY(profile_id, revision, file_id),
    FOREIGN KEY(profile_id, revision) REFERENCES traceability_revisions(profile_id, number)
);
CREATE INDEX media_bind_jobs_pending ON media_bind_jobs(retry_at) WHERE bound_at IS NULL;
CREATE FUNCTION protect_media_bind_job() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP <> 'UPDATE' THEN
  RAISE EXCEPTION 'media binding obligations cannot be deleted' USING ERRCODE='23514';
 END IF;
 IF ROW(NEW.profile_id,NEW.revision,NEW.file_id,NEW.created_at) IS DISTINCT FROM ROW(OLD.profile_id,OLD.revision,OLD.file_id,OLD.created_at)
    OR (OLD.bound_at IS NOT NULL AND NEW IS DISTINCT FROM OLD) THEN
  RAISE EXCEPTION 'media binding identity and completion are immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER media_bind_job_protection BEFORE UPDATE OR DELETE ON media_bind_jobs FOR EACH ROW EXECUTE FUNCTION protect_media_bind_job();
