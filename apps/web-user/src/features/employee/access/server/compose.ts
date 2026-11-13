import { AccessProjectionError } from "../live";
import type { AccessProjectionKind } from "../live";
import type {
  AdminPermissionDetail,
  AdminPermissionList,
  AdminPermissionSummary,
  AdminRoleDetail,
  AdminRoleList,
  AdminRoleSummary,
  EffectivePermission,
  EmployeeAccessDetail,
  EmployeeAccessList,
  EmployeeAccessSummary,
  EmployeeAccountStatus,
  EmployeeRoleAssignment,
  RoleAssignmentStatus,
} from "../types";

/**
 * F-37B composition: no service implements the iam-admin-read.v1 projection,
 * so in OIDC mode the BFF composes it from the real admin APIs — identity
 * users/roles/assignments and, best-effort, profile HR employee records.
 * Every field comes from one of those responses; HR fields stay empty for
 * accounts that have no profile employee record rather than being invented.
 */

export class AccessUpstreamError extends Error {
  constructor(readonly status: number) {
    super(`Admin upstream returned ${status}`);
  }
}

type Rec = Record<string, unknown>;
const rec = (v: unknown): Rec => (v && typeof v === "object" && !Array.isArray(v) ? (v as Rec) : {});
const str = (v: unknown, fallback = ""): string => (typeof v === "string" ? v : fallback);
const arr = (v: unknown): unknown[] => (Array.isArray(v) ? v : []);
const bool = (v: unknown): boolean => v === true;
const num = (v: unknown, fallback = 0): number => (typeof v === "number" && Number.isFinite(v) ? v : fallback);
const iso = (v: unknown, fallback: string): string => {
  const s = str(v);
  return s && Number.isFinite(Date.parse(s)) ? s : fallback;
};

interface Client {
  get(path: string): Promise<unknown>;
}

function makeClient(base: string, token: string): Client {
  return {
    async get(path: string) {
      let response: Response;
      try {
        response = await fetch(`${base}${path}`, {
          headers: { Accept: "application/json", Authorization: `Bearer ${token}` },
          cache: "no-store",
          redirect: "manual",
          signal: AbortSignal.timeout(10000),
        });
      } catch {
        throw new AccessUpstreamError(503);
      }
      if (!response.ok) throw new AccessUpstreamError(response.status);
      const payload: unknown = await response.json().catch(() => null);
      const envelope = rec(payload);
      return "data" in envelope ? envelope.data : payload;
    },
  };
}

interface RoleRow {
  roleId: string;
  roleCode: string;
  roleName: string;
  description: string;
  isProtected: boolean;
  isEnabled: boolean;
  version: number;
  updatedAt: string;
}

interface AssignmentRow {
  assignmentId: string;
  principalId: string;
  roleId: string;
  roleCode: string;
  roleName: string;
  scopeType: string;
  scopeId: string;
  assignmentStatus: string;
  assignedByPrincipalId: string;
  validFrom: string;
  validUntil: string | null;
}

interface HrInfo {
  employeeId: string;
  employeeCode: string;
  department: string;
  jobTitle: string;
}

const mapRole = (v: unknown, now: string): RoleRow => {
  const r = rec(v);
  return {
    roleId: str(r.roleId),
    roleCode: str(r.roleCode),
    roleName: str(r.roleName),
    description: str(r.description),
    isProtected: bool(r.isProtected),
    isEnabled: bool(r.isEnabled),
    version: Math.max(1, num(r.version, 1)),
    updatedAt: iso(r.updatedAt, iso(r.createdAt, now)),
  };
};

const mapAssignment = (v: unknown, now: string): AssignmentRow => {
  const a = rec(v);
  return {
    assignmentId: str(a.assignmentId),
    principalId: str(a.principalId),
    roleId: str(a.roleId),
    roleCode: str(a.roleCode),
    roleName: str(a.roleName),
    scopeType: str(a.scopeType, "GLOBAL"),
    scopeId: str(a.scopeId),
    assignmentStatus: str(a.assignmentStatus),
    assignedByPrincipalId: str(a.assignedByPrincipalId),
    validFrom: iso(a.validFrom, iso(a.createdAt, now)),
    validUntil: typeof a.validUntil === "string" ? iso(a.validUntil, now) : null,
  };
};

function assignmentStatus(raw: string): RoleAssignmentStatus {
  if (raw === "ACTIVE") return "ACTIVE";
  if (raw === "REVOKED") return "REVOKED";
  if (raw === "SCHEDULED" || raw === "PENDING") return "SCHEDULED";
  return "EXPIRED";
}

function accountStatus(provisioningStatus: string): EmployeeAccountStatus {
  if (provisioningStatus === "ACTIVE") return "ACTIVE";
  if (provisioningStatus === "PENDING_VERIFICATION") return "PENDING_ACTIVATION";
  if (provisioningStatus === "SUSPENDED") return "DISABLED";
  return "DISABLED";
}

const ACCOUNT_LABELS: Record<EmployeeAccountStatus, string> = {
  INVITED: "Đã mời",
  PENDING_ACTIVATION: "Chờ kích hoạt",
  ACTIVE: "Đang hoạt động",
  LOCKED: "Đã khóa",
  DISABLED: "Đã vô hiệu hóa",
};

function scopeLabel(scopeType: string, scopeId: string): string {
  if (scopeType === "GLOBAL") return "Toàn hệ thống";
  if (scopeType === "WAREHOUSE") return scopeId ? `Kho ${scopeId}` : "Kho";
  if (scopeType === "DEPARTMENT") return scopeId ? `Phòng ban ${scopeId}` : "Phòng ban";
  if (scopeType === "STORE") return scopeId ? `Cửa hàng ${scopeId}` : "Cửa hàng";
  return scopeType;
}

const TWO_WORD_ACTIONS = ["READ_METADATA", "ASSIGN_SELF", "READ_HISTORY", "RESOLVE_BLOCK", "CANCEL_REQUEST", "PASSWORD_RESET", "SESSION_REVOKE", "SENSITIVE_VIEW", "COMPLIANCE_VIEW"];

/** Display derivation from the code itself: last segment(s) are the action, the rest is the resource. */
export function splitPermissionCode(code: string): { action: string; resource: string } {
  for (const suffix of TWO_WORD_ACTIONS) {
    if (code.endsWith(`_${suffix}`)) return { action: suffix, resource: code.slice(0, code.length - suffix.length - 1) };
  }
  const idx = code.lastIndexOf("_");
  if (idx <= 0) return { action: "", resource: code };
  return { action: code.slice(idx + 1), resource: code.slice(0, idx) };
}

/** Humanised label derived from the code (the catalogue stores no separate names). */
export function permissionName(code: string): string {
  return code
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0) + part.slice(1).toLowerCase())
    .join(" ");
}

const MODULE_BY_RESOURCE: Record<string, string> = {
  ORDER: "Đơn hàng",
  CATALOG: "Danh mục",
  FULFILLMENT: "Đóng gói & giao vận",
  USER: "Quản trị truy cập",
  ROLE: "Quản trị truy cập",
  PERMISSION: "Quản trị truy cập",
  EMPLOYEE: "Quản trị truy cập",
  AUDIT: "Nhật ký kiểm toán",
};

interface Catalogue {
  roles: RoleRow[];
  rolePermissions: Map<string, string[]>; // roleId -> codes
  assignments: AssignmentRow[];
}

async function loadCatalogue(client: Client, now: string): Promise<Catalogue> {
  const roles = arr(await client.get("/admin/roles")).map((r) => mapRole(r, now)).filter((r) => r.roleId);
  const rolePermissions = new Map<string, string[]>();
  await Promise.all(
    roles.map(async (role) => {
      const data = await client.get(`/admin/roles/${encodeURIComponent(role.roleId)}/permissions`);
      const codes = arr(data).map((c) => str(c)).filter(Boolean);
      rolePermissions.set(role.roleId, codes);
    }),
  );
  const assignments = arr(await client.get("/admin/role-assignments?size=200")).map((a) => mapAssignment(a, now));
  return { roles, rolePermissions, assignments };
}

function activePrincipalsByRole(catalogue: Catalogue): Map<string, Set<string>> {
  const map = new Map<string, Set<string>>();
  for (const a of catalogue.assignments) {
    if (assignmentStatus(a.assignmentStatus) !== "ACTIVE" || !a.principalId) continue;
    const set = map.get(a.roleId) ?? new Set<string>();
    set.add(a.principalId);
    map.set(a.roleId, set);
  }
  return map;
}

async function loadHr(client: Client): Promise<Map<string, HrInfo>> {
  const map = new Map<string, HrInfo>();
  try {
    const data = await client.get("/admin/employees");
    const items = Array.isArray(data) ? data : arr(rec(data).items);
    for (const raw of items) {
      const e = rec(raw);
      const userId = str(e.user_id ?? e.userId);
      if (!userId) continue;
      map.set(userId, {
        employeeId: str(e.id),
        employeeCode: str(e.employee_code ?? e.employeeCode),
        department: str(e.department_name ?? e.departmentName ?? e.department_id ?? e.departmentId),
        jobTitle: str(e.position ?? e.jobTitle),
      });
    }
  } catch {
    // HR enrichment is best-effort: identity data alone still composes a truthful projection.
  }
  return map;
}

interface UserDetail {
  principalId: string;
  email: string;
  displayName: string;
  provisioningStatus: string;
  createdAt: string;
  updatedAt: string;
  permissions: string[];
  roles: AssignmentRow[];
}

function mapUserDetail(v: unknown, now: string): UserDetail {
  const u = rec(v);
  const principalId = str(u.principalId);
  return {
    principalId,
    email: str(u.email),
    displayName: [str(u.firstName), str(u.lastName)].filter(Boolean).join(" ").trim(),
    provisioningStatus: str(u.provisioningStatus),
    createdAt: iso(u.createdAt, now),
    updatedAt: iso(u.updatedAt, iso(u.createdAt, now)),
    permissions: arr(u.permissions).map((p) => str(p)).filter(Boolean),
    roles: arr(u.roles).map((r) =>
      // The paged list endpoint carries role codes only; the detail endpoint
      // carries full assignment rows. Support both shapes.
      typeof r === "string"
        ? { ...mapAssignment({}, now), principalId, roleCode: r, roleName: r, assignmentStatus: "ACTIVE" }
        : { ...mapAssignment(r, now), principalId },
    ),
  };
}

function summarize(user: UserDetail, hr: HrInfo | undefined): EmployeeAccessSummary {
  const status = accountStatus(user.provisioningStatus);
  const roles = user.roles.map((a) => ({ roleCode: a.roleCode, roleName: a.roleName, status: assignmentStatus(a.assignmentStatus) }));
  const hasActiveRole = roles.some((r) => r.status === "ACTIVE");
  const accessState = status !== "ACTIVE" ? "BLOCKED_ACCOUNT" : hasActiveRole ? "ENABLED" : "NO_ACTIVE_ROLE";
  return {
    accountId: user.principalId,
    employeeId: user.principalId,
    employeeCode: hr?.employeeCode ?? "",
    displayName: user.displayName || user.email,
    email: user.email,
    department: hr?.department ?? "",
    jobTitle: hr?.jobTitle ?? "",
    loginIdentifier: user.email,
    status,
    statusLabel: ACCOUNT_LABELS[status],
    invitationStatus: status === "ACTIVE" ? "ACCEPTED" : status === "PENDING_ACTIVATION" ? "PENDING" : "NOT_SENT",
    roles,
    effectivePermissionCodes: user.permissions,
    accessState,
    accessStateLabel: accessState === "ENABLED" ? "Đang được cấp quyền" : accessState === "BLOCKED_ACCOUNT" ? "Bị chặn bởi tài khoản" : "Chưa có Role hiệu lực",
    updatedAt: user.updatedAt,
    revision: 1,
  };
}

async function composeEmployees(client: Client, now: string): Promise<EmployeeAccessList> {
  const page = rec(await client.get("/admin/users?type=EMPLOYEE&size=100"));
  const items = arr(page.items);
  const [hr, catalogue] = await Promise.all([loadHr(client), loadCatalogue(client, now)]);
  const users: UserDetail[] = [];
  for (const raw of items) {
    const item = rec(raw);
    const id = str(item.principalId);
    if (!id) continue;
    try {
      users.push(mapUserDetail(await client.get(`/admin/users/${encodeURIComponent(id)}`), now));
    } catch (cause) {
      if (cause instanceof AccessUpstreamError && (cause.status === 401 || cause.status === 403)) throw cause;
      users.push(mapUserDetail(item, now));
    }
  }
  const summaries = users.map((u) => summarize(u, hr.get(u.principalId)));
  const statusCounts: EmployeeAccessList["statusCounts"] = [
    { status: "ALL", label: "Tất cả", count: summaries.length },
    ...(Object.keys(ACCOUNT_LABELS) as EmployeeAccountStatus[]).map((s) => ({
      status: s,
      label: ACCOUNT_LABELS[s],
      count: summaries.filter((x) => x.status === s).length,
    })),
  ];
  const permissions: Array<{ permissionCode: string; permissionName: string }> = [];
  const seen = new Set<string>();
  for (const codes of catalogue.rolePermissions.values()) {
    for (const code of codes) {
      if (seen.has(code)) continue;
      seen.add(code);
      permissions.push({ permissionCode: code, permissionName: permissionName(code) });
    }
  }
  return {
    items: summaries,
    statusCounts,
    departments: [...new Set(summaries.map((s) => s.department).filter(Boolean))],
    roles: catalogue.roles.map((r) => ({ roleCode: r.roleCode, roleName: r.roleName })),
    permissions,
    calculatedAt: now,
  };
}

async function composeEmployeeDetail(client: Client, principalId: string, now: string): Promise<EmployeeAccessDetail> {
  const [detailRaw, hr, catalogue, page] = await Promise.all([
    client.get(`/admin/users/${encodeURIComponent(principalId)}`),
    loadHr(client),
    loadCatalogue(client, now),
    client.get("/admin/users?type=EMPLOYEE&size=100").catch(() => null),
  ]);
  const user = mapUserDetail(detailRaw, now);
  const summary = summarize(user, hr.get(principalId));
  const nameByPrincipal = new Map<string, string>();
  for (const raw of arr(rec(page).items)) {
    const item = rec(raw);
    const name = [str(item.firstName), str(item.lastName)].filter(Boolean).join(" ").trim();
    if (str(item.principalId)) nameByPrincipal.set(str(item.principalId), name || str(item.email));
  }
  const roleById = new Map(catalogue.roles.map((r) => [r.roleId, r]));
  const roleAssignments: EmployeeRoleAssignment[] = user.roles.map((a) => {
    const role = roleById.get(a.roleId);
    return {
      assignmentId: a.assignmentId,
      roleCode: a.roleCode,
      roleName: a.roleName,
      roleStatus: role ? (role.isEnabled ? "ACTIVE" : "INACTIVE") : "ACTIVE",
      assignmentStatus: assignmentStatus(a.assignmentStatus),
      scopeLabel: scopeLabel(a.scopeType, a.scopeId),
      effectiveFrom: a.validFrom,
      effectiveUntil: a.validUntil,
      assignedByLabel: nameByPrincipal.get(a.assignedByPrincipalId) ?? a.assignedByPrincipalId,
    };
  });
  const sourcesByCode = new Map<string, EffectivePermission>();
  for (const a of user.roles) {
    if (assignmentStatus(a.assignmentStatus) !== "ACTIVE") continue;
    for (const code of catalogue.rolePermissions.get(a.roleId) ?? []) {
      const entry = sourcesByCode.get(code) ?? {
        permissionCode: code,
        permissionName: permissionName(code),
        ...splitPermissionCode(code),
        scopeLabels: [],
        sources: [],
      };
      entry.sources.push({ roleCode: a.roleCode, roleName: a.roleName, assignmentId: a.assignmentId });
      const label = scopeLabel(a.scopeType, a.scopeId);
      if (!entry.scopeLabels.includes(label)) entry.scopeLabels.push(label);
      sourcesByCode.set(code, entry);
    }
  }
  const effectivePermissions = user.permissions.map(
    (code) =>
      sourcesByCode.get(code) ?? {
        permissionCode: code,
        permissionName: permissionName(code),
        ...splitPermissionCode(code),
        scopeLabels: [],
        sources: [],
      },
  );
  const hrInfo = hr.get(principalId);
  return {
    ...summary,
    profileReference: { employeeId: hrInfo?.employeeId || principalId, employeeCode: hrInfo?.employeeCode ?? "", profileOwner: "EPIC_02" },
    identity: { credentialReadable: false },
    lock: null,
    roleAssignments,
    effectivePermissions,
    calculatedAt: now,
  };
}

function roleSummary(role: RoleRow, catalogue: Catalogue, principalsByRole: Map<string, Set<string>>): AdminRoleSummary {
  const status = role.isEnabled ? "ACTIVE" : "INACTIVE";
  return {
    roleCode: role.roleCode,
    roleName: role.roleName,
    description: role.description,
    scopeLabel: "Theo phân công",
    status,
    statusLabel: status === "ACTIVE" ? "Đang hoạt động" : "Đã vô hiệu hóa",
    kind: role.isProtected ? "SYSTEM" : "CUSTOM",
    privileged: role.isProtected,
    employeeCount: principalsByRole.get(role.roleId)?.size ?? 0,
    permissionCount: catalogue.rolePermissions.get(role.roleId)?.length ?? 0,
    updatedAt: role.updatedAt,
    revision: role.version,
  };
}

async function composeRoles(client: Client, now: string): Promise<AdminRoleList> {
  const catalogue = await loadCatalogue(client, now);
  const principalsByRole = activePrincipalsByRole(catalogue);
  const items = catalogue.roles.map((r) => roleSummary(r, catalogue, principalsByRole));
  return {
    items,
    statusCounts: [
      { status: "ALL", label: "Tất cả", count: items.length },
      { status: "ACTIVE", label: "Đang hoạt động", count: items.filter((i) => i.status === "ACTIVE").length },
      { status: "INACTIVE", label: "Đã vô hiệu hóa", count: items.filter((i) => i.status === "INACTIVE").length },
      { status: "ARCHIVED", label: "Đã lưu trữ", count: items.filter((i) => i.status === "ARCHIVED").length },
    ],
    calculatedAt: now,
  };
}

async function composeRoleDetail(client: Client, roleCode: string, now: string): Promise<AdminRoleDetail> {
  const catalogue = await loadCatalogue(client, now);
  const role = catalogue.roles.find((r) => r.roleCode === roleCode);
  if (!role) throw new AccessUpstreamError(404);
  const principalsByRole = activePrincipalsByRole(catalogue);
  const summary = roleSummary(role, catalogue, principalsByRole);
  const mine = catalogue.assignments.filter((a) => a.roleId === role.roleId);
  const scheduled = mine.filter((a) => assignmentStatus(a.assignmentStatus) === "ACTIVE" && Date.parse(a.validFrom) > Date.parse(now)).length;
  const page = rec(await client.get("/admin/users?type=EMPLOYEE&size=100").catch(() => null));
  const suspended = new Set(arr(page.items).filter((raw) => str(rec(raw).provisioningStatus) === "SUSPENDED").map((raw) => str(rec(raw).principalId)));
  const lockedAccounts = new Set(mine.filter((a) => suspended.has(a.principalId)).map((a) => a.principalId)).size;
  return {
    ...summary,
    protectedRole: role.isProtected,
    permissions: (catalogue.rolePermissions.get(role.roleId) ?? []).map((code) => ({
      permissionCode: code,
      permissionName: permissionName(code),
      ...splitPermissionCode(code),
      scopeLabel: "Theo phân công",
    })),
    impact: { activeEmployees: summary.employeeCount, scheduledAssignments: scheduled, lockedAccounts },
  };
}

interface PermissionAggregate {
  code: string;
  roles: RoleRow[];
  updatedAt: string;
}

function aggregatePermissions(catalogue: Catalogue): PermissionAggregate[] {
  const map = new Map<string, PermissionAggregate>();
  for (const role of catalogue.roles) {
    for (const code of catalogue.rolePermissions.get(role.roleId) ?? []) {
      const entry = map.get(code) ?? { code, roles: [], updatedAt: role.updatedAt };
      entry.roles.push(role);
      if (role.updatedAt > entry.updatedAt) entry.updatedAt = role.updatedAt;
      map.set(code, entry);
    }
  }
  return [...map.values()].sort((a, b) => a.code.localeCompare(b.code));
}

function permissionSummary(entry: PermissionAggregate, principalsByRole: Map<string, Set<string>>): AdminPermissionSummary {
  const employees = new Set<string>();
  for (const role of entry.roles) for (const p of principalsByRole.get(role.roleId) ?? []) employees.add(p);
  const { action, resource } = splitPermissionCode(entry.code);
  return {
    permissionCode: entry.code,
    permissionName: permissionName(entry.code),
    description: "",
    action,
    resource,
    scopeLabel: "Theo phân công",
    status: "ACTIVE",
    statusLabel: "Đang hoạt động",
    kind: "SYSTEM",
    roleCount: entry.roles.length,
    employeeCount: employees.size,
    updatedAt: entry.updatedAt,
  };
}

async function composePermissions(client: Client, now: string): Promise<AdminPermissionList> {
  const catalogue = await loadCatalogue(client, now);
  const principalsByRole = activePrincipalsByRole(catalogue);
  const aggregates = aggregatePermissions(catalogue);
  const items = aggregates.map((e) => permissionSummary(e, principalsByRole));
  return {
    items,
    statusCounts: [
      { status: "ALL", label: "Tất cả", count: items.length },
      { status: "ACTIVE", label: "Đang hoạt động", count: items.length },
      { status: "DEPRECATED", label: "Ngừng khuyến nghị", count: 0 },
      { status: "ARCHIVED", label: "Đã lưu trữ", count: 0 },
    ],
    resources: [...new Set(items.map((i) => i.resource))].sort(),
    calculatedAt: now,
  };
}

async function composePermissionDetail(client: Client, code: string, now: string): Promise<AdminPermissionDetail> {
  const catalogue = await loadCatalogue(client, now);
  const principalsByRole = activePrincipalsByRole(catalogue);
  const entry = aggregatePermissions(catalogue).find((e) => e.code === code);
  if (!entry) throw new AccessUpstreamError(404);
  const summary = permissionSummary(entry, principalsByRole);
  const module = MODULE_BY_RESOURCE[summary.resource.split("_")[0]];
  return {
    ...summary,
    protectedCode: true,
    usedByModules: module ? [module] : [],
    rolesUsing: entry.roles.map((role) => ({
      roleCode: role.roleCode,
      roleName: role.roleName,
      status: role.isEnabled ? "ACTIVE" : "INACTIVE",
      employeeCount: principalsByRole.get(role.roleId)?.size ?? 0,
    })),
  };
}

export async function composeAccessProjection(
  kind: AccessProjectionKind,
  target: string | undefined,
  base: string,
  token: string,
): Promise<{ contractVersion: "iam-admin-read.v1"; capabilities: { accessReview: boolean }; data: unknown }> {
  const client = makeClient(base, token);
  const now = new Date().toISOString();
  let data: unknown;
  switch (kind) {
    case "list":
      data = await composeEmployees(client, now);
      break;
    case "detail":
      if (!target) throw new AccessProjectionError("missing employee id");
      data = await composeEmployeeDetail(client, target, now);
      break;
    case "role-list":
      data = await composeRoles(client, now);
      break;
    case "role-detail":
      if (!target) throw new AccessProjectionError("missing role code");
      data = await composeRoleDetail(client, target, now);
      break;
    case "permission-list":
      data = await composePermissions(client, now);
      break;
    case "permission-detail":
      if (!target) throw new AccessProjectionError("missing permission code");
      data = await composePermissionDetail(client, target, now);
      break;
  }
  return { contractVersion: "iam-admin-read.v1", capabilities: { accessReview: true }, data };
}
