INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('traceability.read_internal', 'TRACEABILITY', 'READ_INTERNAL', 'LOW', FALSE, 'Read internal traceability profiles'),
    ('traceability.read_evidence', 'TRACEABILITY', 'READ_EVIDENCE', 'MEDIUM', FALSE, 'Read traceability evidence references'),
    ('traceability.create', 'TRACEABILITY', 'CREATE', 'LOW', FALSE, 'Create traceability profiles'),
    ('traceability.edit', 'TRACEABILITY', 'EDIT', 'LOW', FALSE, 'Edit traceability drafts'),
    ('traceability.submit', 'TRACEABILITY', 'SUBMIT', 'MEDIUM', FALSE, 'Submit traceability revisions'),
    ('traceability.approve', 'TRACEABILITY', 'APPROVE', 'HIGH', FALSE, 'Approve or reject traceability revisions'),
    ('traceability.publish', 'TRACEABILITY', 'PUBLISH', 'HIGH', FALSE, 'Publish traceability revisions'),
    ('traceability.suspend', 'TRACEABILITY', 'SUSPEND', 'HIGH', FALSE, 'Suspend traceability profiles'),
    ('traceability.archive', 'TRACEABILITY', 'ARCHIVE', 'HIGH', FALSE, 'Archive traceability profiles')
ON CONFLICT (permission_code) DO NOTHING;
