ALTER TABLE iam_roles ADD COLUMN archived_at TIMESTAMPTZ;

-- Legacy disable and archive shared the same state. Preserve terminal roles
-- with assignment history. A never-assigned role remains editable when its
-- code has not been reused; keep only the newest such row for each code.
WITH legacy_disabled AS (
    SELECT r.role_id, r.updated_at,
           EXISTS (SELECT 1 FROM iam_role_assignments a WHERE a.role_id = r.role_id) AS has_history,
           ROW_NUMBER() OVER (
               PARTITION BY r.role_code ORDER BY r.is_enabled DESC,
               EXISTS (SELECT 1 FROM iam_role_assignments a WHERE a.role_id = r.role_id),
               r.updated_at DESC, r.role_id
           ) AS code_position
    FROM iam_roles r
)
UPDATE iam_roles r
SET archived_at = d.updated_at
FROM legacy_disabled d
WHERE r.role_id = d.role_id AND NOT r.is_enabled
  AND (d.has_history OR d.code_position > 1);

ALTER TABLE iam_roles ADD CONSTRAINT ck_iam_roles_archived_disabled
    CHECK (archived_at IS NULL OR NOT is_enabled);

DROP INDEX uq_iam_roles_role_code_enabled;
CREATE UNIQUE INDEX uq_iam_roles_role_code_unarchived
    ON iam_roles (role_code) WHERE archived_at IS NULL;
