-- Stop variance actions first. Never discard completed decisions for rollback.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM receipt_variance_resolution_results) THEN
        RAISE EXCEPTION 'cannot roll back: completed variance resolution evidence exists';
    END IF;
END $$;
DROP TABLE receipt_variance_resolution_results;
