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
  "B2B_ORDER_CONVERT"
] as const;

export type AdminPermission = (typeof ADMIN_PERMISSIONS)[number];
export type AdminRole = "SALES_MANAGER" | "CUSTOMER_SERVICE" | "B2B_VIEWER";

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
  const roles = source.roles.filter((item): item is AdminRole => ["SALES_MANAGER", "CUSTOMER_SERVICE", "B2B_VIEWER"].includes(String(item)));
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
