-- Initial migration rollback is intentionally refused: financial obligations must not be erased.
DO $$ BEGIN RAISE EXCEPTION 'Use an audited restore/forward migration; destructive rollback refused'; END $$;
