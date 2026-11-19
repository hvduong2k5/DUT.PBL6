-- V9: Replace full-table UNIQUE(role_code) with a partial unique index for active roles only.
-- Soft-deleted roles (is_enabled = FALSE) keep their original role_code and cannot be reactivated.
-- This allows a new role to reuse a role_code only after the previous one is deleted (not soft-deleted).

DO $$
DECLARE
    v_constraint_name text;
BEGIN
    -- Find the unique constraint on role_code (single-column unique constraint on iam_roles)
    SELECT conname INTO v_constraint_name
    FROM pg_constraint
    WHERE conrelid = 'iam_roles'::regclass
      AND contype = 'u'
      AND array_length(conkey, 1) = 1
      AND conkey[1] = (
          SELECT attnum
          FROM pg_attribute
          WHERE attrelid = 'iam_roles'::regclass
            AND attname = 'role_code'
      );

    IF v_constraint_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE iam_roles DROP CONSTRAINT ' || quote_ident(v_constraint_name);
        RAISE NOTICE 'Dropped full-table unique constraint: %', v_constraint_name;
    ELSE
        RAISE NOTICE 'No full-table unique constraint on role_code found; skipping drop.';
    END IF;
END $$;

-- Partial unique index: role_code must be unique among ACTIVE (is_enabled = TRUE) roles only.
-- Soft-deleted roles (is_enabled = FALSE) are excluded and their role_code is "freed" logically,
-- but reactivation of a soft-deleted role is blocked at the application layer.
CREATE UNIQUE INDEX IF NOT EXISTS uq_iam_roles_role_code_enabled
    ON iam_roles (role_code)
    WHERE is_enabled = TRUE;
