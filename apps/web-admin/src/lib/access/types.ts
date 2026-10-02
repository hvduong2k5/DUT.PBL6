import type { AdminPermission } from "@/lib/auth/types";

export type EmployeeAccountStatus = "INVITED" | "PENDING_ACTIVATION" | "ACTIVE" | "LOCKED" | "DISABLED";
export type EmployeeAccessState = "ENABLED" | "BLOCKED_ACCOUNT" | "NO_ACTIVE_ROLE";
export type RoleAssignmentStatus = "SCHEDULED" | "ACTIVE" | "EXPIRED" | "REVOKED";

export interface EmployeeRoleSummary {
  roleCode: string;
  roleName: string;
  status: RoleAssignmentStatus;
}

export interface EmployeeAccessSummary {
  accountId: string;
  employeeId: string;
  employeeCode: string;
  displayName: string;
  email: string;
  department: string;
  jobTitle: string;
  loginIdentifier: string;
  status: EmployeeAccountStatus;
  statusLabel: string;
  invitationStatus: "NOT_SENT" | "PENDING" | "DELIVERED" | "FAILED" | "ACCEPTED";
  roles: EmployeeRoleSummary[];
  effectivePermissionCodes: AdminPermission[];
  accessState: EmployeeAccessState;
  accessStateLabel: string;
  updatedAt: string;
  revision: number;
}

export interface EmployeeAccessList {
  items: EmployeeAccessSummary[];
  statusCounts: Array<{ status: "ALL" | EmployeeAccountStatus; label: string; count: number }>;
  departments: string[];
  roles: Array<{ roleCode: string; roleName: string }>;
  permissions: Array<{ permissionCode: AdminPermission; permissionName: string }>;
  calculatedAt: string;
}

export interface EmployeeRoleAssignment {
  assignmentId: string;
  roleCode: string;
  roleName: string;
  roleStatus: "ACTIVE" | "INACTIVE" | "ARCHIVED";
  assignmentStatus: RoleAssignmentStatus;
  scopeLabel: string;
  effectiveFrom: string;
  effectiveUntil: string | null;
  assignedByLabel: string;
}

export interface EffectivePermission {
  permissionCode: AdminPermission;
  permissionName: string;
  action: string;
  resource: string;
  scopeLabels: string[];
  sources: Array<{ roleCode: string; roleName: string; assignmentId: string }>;
}

export interface EmployeeAccessDetail extends EmployeeAccessSummary {
  profileReference: { employeeId: string; employeeCode: string; profileOwner: "EPIC_02" };
  identity: { emailVerified: boolean; lastAuthenticatedAt: string | null; credentialReadable: false };
  lock: null | { effectiveAt: string; reason: string; decidedByLabel: string };
  roleAssignments: EmployeeRoleAssignment[];
  effectivePermissions: EffectivePermission[];
  calculatedAt: string;
}

export type AdminRoleStatus = "ACTIVE" | "INACTIVE" | "ARCHIVED";
export type AdminRoleKind = "SYSTEM" | "CUSTOM";

export interface AdminRoleSummary {
  roleCode: string;
  roleName: string;
  description: string;
  scopeLabel: string;
  status: AdminRoleStatus;
  statusLabel: string;
  kind: AdminRoleKind;
  privileged: boolean;
  employeeCount: number;
  permissionCount: number;
  updatedAt: string;
  revision: number;
}

export interface AdminRoleList {
  items: AdminRoleSummary[];
  statusCounts: Array<{ status: "ALL" | AdminRoleStatus; label: string; count: number }>;
  calculatedAt: string;
}

export interface AdminRoleDetail extends AdminRoleSummary {
  protectedRole: boolean;
  permissions: Array<{ permissionCode: AdminPermission; permissionName: string; action: string; resource: string; scopeLabel: string }>;
  impact: { activeEmployees: number; scheduledAssignments: number; lockedAccounts: number };
}

export type AdminPermissionStatus = "ACTIVE" | "DEPRECATED" | "ARCHIVED";
export type AdminPermissionKind = "SYSTEM" | "CONFIGURABLE";

export interface AdminPermissionSummary {
  permissionCode: AdminPermission;
  permissionName: string;
  description: string;
  action: string;
  resource: string;
  scopeLabel: string;
  status: AdminPermissionStatus;
  statusLabel: string;
  kind: AdminPermissionKind;
  roleCount: number;
  employeeCount: number;
  updatedAt: string;
}

export interface AdminPermissionList {
  items: AdminPermissionSummary[];
  statusCounts: Array<{ status: "ALL" | AdminPermissionStatus; label: string; count: number }>;
  resources: string[];
  calculatedAt: string;
}

export interface AdminPermissionDetail extends AdminPermissionSummary {
  protectedCode: boolean;
  usedByModules: string[];
  rolesUsing: Array<{ roleCode: string; roleName: string; status: AdminRoleStatus; employeeCount: number }>;
}

interface AdminAccessErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminAccessApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: AdminAccessErrorBody) {
    super(body.message || "Không thể xử lý yêu cầu quản trị truy cập.");
    this.name = "AdminAccessApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
  }
}
