-- Services enforce the permission together with this assignment's scope.
INSERT INTO iam_permissions (permission_code, resource_code, action_code, risk_level, scope_required, description)
VALUES
 ('EMPLOYEE_PROFILE_VIEW','EMPLOYEE_PROFILE','VIEW','MEDIUM',TRUE,'Read employee profiles within department scope'),
 ('EMPLOYEE_PROFILE_CREATE','EMPLOYEE_PROFILE','CREATE','HIGH',TRUE,'Create employee profiles within department scope'),
 ('EMPLOYEE_PROFILE_SENSITIVE_VIEW','EMPLOYEE_PROFILE','SENSITIVE_VIEW','HIGH',TRUE,'Read employee phone, identity card and certificate numbers'),
 ('EMPLOYEE_COMPLIANCE_VIEW','EMPLOYEE_COMPLIANCE','VIEW','MEDIUM',TRUE,'Read expiring employee certificates within department scope'),
 ('ORDER_VIEW','ORDER','VIEW','MEDIUM',TRUE,'Read orders within warehouse scope'),
 ('ORDER_SENSITIVE_VIEW','ORDER','SENSITIVE_VIEW','HIGH',TRUE,'Read order recipient name, phone and street'),
 ('ORDER_CANCEL_REQUEST','ORDER','CANCEL_REQUEST','HIGH',TRUE,'Request cancellation within existing lifecycle safeguards'),
 ('ORDER_HOLD_MANAGE','ORDER','HOLD_MANAGE','HIGH',TRUE,'Manage operational holds within warehouse scope'),
 ('ORDER_RECOVERY_EXECUTE','ORDER','RECOVERY_EXECUTE','HIGH',TRUE,'Resume parked operations within warehouse scope')
ON CONFLICT (permission_code) DO NOTHING;

INSERT INTO iam_role_permissions (role_id, permission_code)
SELECT 'a0000000-0000-0000-0000-000000000001', permission_code
FROM iam_permissions WHERE resource_code IN ('EMPLOYEE_PROFILE','EMPLOYEE_COMPLIANCE','ORDER')
ON CONFLICT (role_id, permission_code) DO NOTHING;
