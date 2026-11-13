import type { AdminPermissionDetail, AdminPermissionList, AdminRoleDetail, AdminRoleList, EmployeeAccessDetail, EmployeeAccessList } from "./types";

export class AccessProjectionError extends Error {}
type Decoder = (value: unknown) => unknown;
const invalid = (): never => { throw new AccessProjectionError("Invalid iam-admin-read.v1 projection"); };
const text: Decoder = (v) => typeof v === "string" && v.length <= 4096 ? v : invalid();
const boolean: Decoder = (v) => typeof v === "boolean" ? v : invalid();
const count: Decoder = (v) => typeof v === "number" && Number.isSafeInteger(v) && v >= 0 ? v : invalid();
const revision: Decoder = (v) => typeof v === "number" && Number.isSafeInteger(v) && v >= 1 ? v : invalid();
const timestamp: Decoder = (v) => typeof v === "string" && /^\d{4}-\d{2}-\d{2}T/u.test(v) && Number.isFinite(Date.parse(v)) ? v : invalid();
const nullable = (decode: Decoder): Decoder => (v) => v === null ? null : decode(v);
const choice = (...values: readonly unknown[]): Decoder => (v) => values.includes(v) ? v : invalid();
const array = (decode: Decoder): Decoder => (v) => Array.isArray(v) && v.length <= 1000 ? v.map(decode) : invalid();
const object = (fields: Record<string, Decoder>): Decoder => (v) => {
  if (!v || typeof v !== "object" || Array.isArray(v)) return invalid();
  const source = v as Record<string, unknown>;
  // Return only contracted fields. Never relay credentials or extra producer fields.
  return Object.fromEntries(Object.entries(fields).map(([key, decode]) => [key, decode(source[key])]));
};
// Catalog codes are producer data; this does not grant access to any UI route.
const code: Decoder = (v) => typeof v === "string" && /^[A-Za-z][A-Za-z0-9_.:-]{0,159}$/u.test(v) ? v : invalid();
const accountStatus = choice("INVITED", "PENDING_ACTIVATION", "ACTIVE", "LOCKED", "DISABLED");
const assignmentStatus = choice("SCHEDULED", "ACTIVE", "EXPIRED", "REVOKED");
const roleStatus = choice("ACTIVE", "INACTIVE", "ARCHIVED");
const permissionStatus = choice("ACTIVE", "DEPRECATED", "ARCHIVED");
const summaryFields = {
  accountId: text, employeeId: text, employeeCode: text, displayName: text, email: text,
  department: text, jobTitle: text, loginIdentifier: text, status: accountStatus, statusLabel: text,
  invitationStatus: choice("NOT_SENT", "PENDING", "DELIVERED", "FAILED", "ACCEPTED"),
  roles: array(object({ roleCode: text, roleName: text, status: assignmentStatus })),
  effectivePermissionCodes: array(code), accessState: choice("ENABLED", "BLOCKED_ACCOUNT", "NO_ACTIVE_ROLE"),
  accessStateLabel: text, updatedAt: timestamp, revision,
};
const roleFields = {
  roleCode: text, roleName: text, description: text, scopeLabel: text, status: roleStatus, statusLabel: text,
  kind: choice("SYSTEM", "CUSTOM"), privileged: boolean, employeeCount: count, permissionCount: count,
  updatedAt: timestamp, revision,
};
const permissionFields = {
  permissionCode: code, permissionName: text, description: text, action: text, resource: text,
  scopeLabel: text, status: permissionStatus, statusLabel: text, kind: choice("SYSTEM", "CONFIGURABLE"),
  roleCount: count, employeeCount: count, updatedAt: timestamp,
};
const statusCounts = (status: Decoder) => array(object({ status: (v) => v === "ALL" ? v : status(v), label: text, count }));
const employeeList = object({
  items: array(object(summaryFields)), statusCounts: statusCounts(accountStatus), departments: array(text),
  roles: array(object({ roleCode: text, roleName: text })),
  permissions: array(object({ permissionCode: code, permissionName: text })), calculatedAt: timestamp,
});
const employeeDetail = object({
  ...summaryFields,
  profileReference: object({ employeeId: text, employeeCode: text, profileOwner: choice("EPIC_02") }),
  identity: object({ emailVerified: boolean, lastAuthenticatedAt: nullable(timestamp), credentialReadable: choice(false) }),
  lock: nullable(object({ effectiveAt: timestamp, reason: text, decidedByLabel: text })),
  roleAssignments: array(object({ assignmentId: text, roleCode: text, roleName: text, roleStatus,
    assignmentStatus, scopeLabel: text, effectiveFrom: timestamp, effectiveUntil: nullable(timestamp), assignedByLabel: text })),
  effectivePermissions: array(object({ permissionCode: code, permissionName: text, action: text, resource: text,
    scopeLabels: array(text), sources: array(object({ roleCode: text, roleName: text, assignmentId: text })) })),
  calculatedAt: timestamp,
});
const roleList = object({ items: array(object(roleFields)), statusCounts: statusCounts(roleStatus), calculatedAt: timestamp });
const roleDetail = object({ ...roleFields, protectedRole: boolean,
  permissions: array(object({ permissionCode: code, permissionName: text, action: text, resource: text, scopeLabel: text })),
  impact: object({ activeEmployees: count, scheduledAssignments: count, lockedAccounts: count }),
});
const permissionList = object({ items: array(object(permissionFields)), statusCounts: statusCounts(permissionStatus), resources: array(text), calculatedAt: timestamp });
const permissionDetail = object({ ...permissionFields, protectedCode: boolean, usedByModules: array(text),
  rolesUsing: array(object({ roleCode: text, roleName: text, status: roleStatus, employeeCount: count })),
});
const decoders = { list: employeeList, detail: employeeDetail, "role-list": roleList, "role-detail": roleDetail, "permission-list": permissionList, "permission-detail": permissionDetail };
export type AccessProjectionKind = keyof typeof decoders;
type Projection = EmployeeAccessList | EmployeeAccessDetail | AdminRoleList | AdminRoleDetail | AdminPermissionList | AdminPermissionDetail;

/** This is current configured access, not an inventory of users' unexpired JWTs. */
export function decodeAccessProjection(kind: AccessProjectionKind, value: unknown): Projection {
  const envelope = object({ contractVersion: choice("iam-admin-read.v1"), capabilities: object({ accessReview: boolean }),
    data: decoders[kind] })(value) as { capabilities: { accessReview: boolean }; data: Projection };
  if (kind === "detail" && !envelope.capabilities.accessReview) return invalid();
  if (kind === "list" && !envelope.capabilities.accessReview) {
    const data = envelope.data as EmployeeAccessList;
    return { ...data, permissions: [], items: data.items.map((item) => ({ ...item, effectivePermissionCodes: [] })) };
  }
  return envelope.data;
}
