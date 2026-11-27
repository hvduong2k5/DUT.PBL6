DO $$ BEGIN
    IF EXISTS(SELECT 1 FROM master_data_create_results) THEN
        RAISE EXCEPTION 'cannot roll back: completed master data create evidence exists';
    END IF;
END $$;
DROP TABLE master_data_create_results;
