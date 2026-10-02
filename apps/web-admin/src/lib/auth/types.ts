export const ADMIN_PERMISSIONS = [
  "ADMIN_DASHBOARD_VIEW",
  "B2B_REQUEST_VIEW",
  "B2B_REQUEST_ASSIGN",
  "B2B_REQUEST_INFO_REQUEST",
  "B2B_FILE_VIEW",
  "B2B_QUOTE_DRAFT",
  "B2B_QUOTE_ISSUE",
  "B2B_QUOTE_WITHDRAW",
  "B2B_QUOTE_REJECT",
  "B2B_ORDER_CONVERT",
  "PRODUCT_VIEW",
  "INVENTORY_VIEW",
  "INVENTORY_MANAGEMENT_VIEW",
  "BATCH_VIEW",
  "WAREHOUSE_WORKBENCH_VIEW",
  "INVENTORY_RECEIPT_CREATE",
  "INVENTORY_ISSUE_CREATE",
  "INVENTORY_ADJUSTMENT_CREATE",
  "ORDER_VIEW",
  "ORDER_SENSITIVE_VIEW",
  "PACKING_TASK_VIEW",
  "PACKING_MANAGEMENT_VIEW",
  "PACKING_WORKBENCH_VIEW",
  "PACKING_CHECKLIST_UPDATE",
  "PACKING_TASK_COMPLETE",
  "SHIPMENT_VIEW",
  "SHIPMENT_MANAGEMENT_VIEW",
  "DELIVERY_WORKBENCH_VIEW",
  "SHIPMENT_SENSITIVE_VIEW",
  "PAYMENT_VIEW",
  "PAYMENT_RECONCILIATION_VIEW",
  "PAYMENT_SENSITIVE_VIEW",
  "RETURN_CASE_VIEW",
  "RETURN_EVIDENCE_VIEW",
  "RETURN_FINANCIAL_VIEW",
  "TICKET_QUEUE_VIEW",
  "TICKET_CONVERSATION_VIEW",
  "TICKET_CONTEXT_VIEW",
  "TICKET_INTERNAL_NOTE_VIEW",
  "EMPLOYEE_ACCOUNT_VIEW",
  "EMPLOYEE_ACCOUNT_CREATE",
  "EMPLOYEE_ACCOUNT_UPDATE",
  "EMPLOYEE_ACCOUNT_LOCK",
  "ROLE_VIEW",
  "ROLE_MANAGE",
  "PERMISSION_VIEW",
  "PERMISSION_MANAGE",
  "ROLE_ASSIGN",
  "ACCESS_REVIEW_VIEW"
] as const;

export type AdminPermission = (typeof ADMIN_PERMISSIONS)[number];
// Role codes are data-driven by Access Management. UI authorization must use atomic permissions, not this value.
export type AdminRole = string;

export interface AdminSession {
  authenticated: true;
  employee: { employeeId: string; displayName: string; email: string; jobTitle: string; department: string };
  roles: AdminRole[];
  permissions: AdminPermission[];
  scopes: Array<{ resource: string; level: "ALL" | "ASSIGNED_ONLY" }>;
  version: number;
}

export function parseAdminSession(value: unknown): AdminSession | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  const source = value as Record<string, unknown>;
  const employee = source.employee && typeof source.employee === "object" && !Array.isArray(source.employee) ? source.employee as Record<string, unknown> : undefined;
  if (source.authenticated !== true || !employee || !Array.isArray(source.roles) || !Array.isArray(source.permissions) || !Array.isArray(source.scopes) || !Number.isInteger(source.version)) return undefined;
  const roles = source.roles.filter((item): item is AdminRole => typeof item === "string" && Boolean(item.trim()));
  const permissions = source.permissions.filter((item): item is AdminPermission => ADMIN_PERMISSIONS.includes(item as AdminPermission));
  if (roles.length !== source.roles.length || permissions.length !== source.permissions.length) return undefined;
  const scopes: AdminSession["scopes"] = source.scopes.flatMap((item): AdminSession["scopes"] => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const scope = item as Record<string, unknown>;
    return typeof scope.resource === "string" && (scope.level === "ALL" || scope.level === "ASSIGNED_ONLY") ? [{ resource: scope.resource, level: scope.level }] : [];
  });
  if (scopes.length !== source.scopes.length || [employee.employeeId, employee.displayName, employee.email, employee.jobTitle, employee.department].some((item) => typeof item !== "string")) return undefined;
  return { authenticated: true, employee: employee as AdminSession["employee"], roles, permissions: [...new Set(permissions)], scopes, version: source.version as number };
}
