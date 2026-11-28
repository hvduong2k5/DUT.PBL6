-- Maintenance only: do not remove pending binding obligations during rollback.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM media_bind_jobs WHERE bound_at IS NULL) THEN
  RAISE EXCEPTION 'pending media bindings must be drained before rollback';
 END IF;
END $$;
DROP TABLE media_bind_jobs;
DROP FUNCTION protect_media_bind_job();
