-- Complete the existing Inventory workbench authorization vocabulary.
-- Do not auto-grant these permissions or widen existing role assignments.
INSERT INTO iam_permissions
    (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
    ('INVENTORY_VIEW', 'INVENTORY', 'VIEW', 'MEDIUM', TRUE,
     'Read authoritative inventory balances within the assigned warehouse scope'),
    ('WAREHOUSE_WORKBENCH_VIEW', 'WAREHOUSE_WORKBENCH', 'VIEW', 'MEDIUM', TRUE,
     'Read the warehouse receipt workbench within the assigned warehouse scope')
ON CONFLICT (permission_code) DO NOTHING;
