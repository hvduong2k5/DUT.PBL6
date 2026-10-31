import { NextRequest, NextResponse } from "next/server";
import { ADMIN_PERMISSIONS, parseAdminSession, type AdminPermission, type AdminSession } from "@/lib/auth/types";
import type { AdminPermissionDetail, AdminPermissionKind, AdminPermissionList, AdminPermissionStatus, AdminPermissionSummary, AdminRoleDetail, AdminRoleKind, AdminRoleList, AdminRoleStatus, AdminRoleSummary, EffectivePermission, EmployeeAccessDetail, EmployeeAccessList, EmployeeAccessState, EmployeeAccessSummary, EmployeeAccountStatus, EmployeeRoleAssignment, EmployeeRoleSummary, RoleAssignmentStatus } from "@/lib/access/types";
import { parseAdminAccessPath } from "@/lib/access/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const ACCOUNT_STATUSES = new Set<EmployeeAccountStatus>(["INVITED", "PENDING_ACTIVATION", "ACTIVE", "LOCKED", "DISABLED"]);
const ASSIGNMENT_STATUSES = new Set<RoleAssignmentStatus>(["SCHEDULED", "ACTIVE", "EXPIRED", "REVOKED"]);
const ACCESS_STATES = new Set<EmployeeAccessState>(["ENABLED", "BLOCKED_ACCOUNT", "NO_ACTIVE_ROLE"]);
const KNOWN_PERMISSIONS = new Set<AdminPermission>(ADMIN_PERMISSIONS);
const ROLE_STATUSES = new Set<AdminRoleStatus>(["ACTIVE", "INACTIVE", "ARCHIVED"]);
const PERMISSION_STATUSES = new Set<AdminPermissionStatus>(["ACTIVE", "DEPRECATED", "ARCHIVED"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) {
  return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-ACCESS-${code}` }, { status });
}
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest) {
  const result = new Headers({ Accept: "application/json" });
  const authorization = request.headers.get("authorization");
  if (authorization) result.set("Authorization", authorization);
  const profile = mockProfile();
  if (profile) result.set("X-Admin-Profile", profile);
  return result;
}
async function upstream(request: NextRequest, path: string) {
  const response = await fetch(`${baseUrl()}/${path}`, { headers: headers(request), cache: "no-store", redirect: "manual" });
  const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {});
  return record(payload) ? payload : {};
}
async function session(request: NextRequest): Promise<AdminSession> {
  const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" });
  if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." });
  const parsed = parseAdminSession(await response.json());
  if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." });
  return parsed;
}
function permission(value: unknown): value is AdminPermission { return KNOWN_PERMISSIONS.has(string(value) as AdminPermission); }
function assignmentStatus(value: unknown): RoleAssignmentStatus {
  const status = string(value) as RoleAssignmentStatus;
  return ASSIGNMENT_STATUSES.has(status) ? status : "EXPIRED";
}
function mapRoleSummary(value: unknown): EmployeeRoleSummary {
  const role = record(value) ? value : {};
  return { roleCode: string(role.roleCode), roleName: string(role.roleName), status: assignmentStatus(role.status) };
}
function mapSummary(value: unknown, canReview: boolean): EmployeeAccessSummary {
  const item = record(value) ? value : {};
  const statusRaw = string(item.status) as EmployeeAccountStatus;
  const status = ACCOUNT_STATUSES.has(statusRaw) ? statusRaw : "DISABLED";
  const accessRaw = string(item.accessState) as EmployeeAccessState;
  const activeRoles = array(item.roles).map(mapRoleSummary);
  const accessState = ACCESS_STATES.has(accessRaw) ? accessRaw : status === "ACTIVE" ? activeRoles.some((role) => role.status === "ACTIVE") ? "ENABLED" : "NO_ACTIVE_ROLE" : "BLOCKED_ACCOUNT";
  const invitation = string(item.invitationStatus);
  return {
    accountId: string(item.accountId), employeeId: string(item.employeeId), employeeCode: string(item.employeeCode), displayName: string(item.displayName), email: string(item.email), department: string(item.department), jobTitle: string(item.jobTitle), loginIdentifier: string(item.loginIdentifier), status, statusLabel: string(item.statusLabel), invitationStatus: ["NOT_SENT", "PENDING", "DELIVERED", "FAILED", "ACCEPTED"].includes(invitation) ? invitation as EmployeeAccessSummary["invitationStatus"] : "NOT_SENT", roles: activeRoles,
    effectivePermissionCodes: canReview ? array(item.effectivePermissionCodes).filter(permission) : [], accessState, accessStateLabel: string(item.accessStateLabel), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1))
  };
}
function mapList(payload: Record<string, unknown>, canReview: boolean): EmployeeAccessList {
  const items = array(payload.items).map((item) => mapSummary(item, canReview));
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | EmployeeAccountStatus; return { status: raw === "ALL" || ACCOUNT_STATUSES.has(raw as EmployeeAccountStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  for (const status of ACCOUNT_STATUSES) if (!statusCounts.some((item) => item.status === status)) statusCounts.push({ status, label: status, count: items.filter((item) => item.status === status).length });
  return {
    items, statusCounts, departments: array(payload.departments).map((item) => string(item)).filter(Boolean),
    roles: array(payload.roles).map((value) => { const item = record(value) ? value : {}; return { roleCode: string(item.roleCode), roleName: string(item.roleName) }; }).filter((item) => item.roleCode),
    permissions: canReview ? array(payload.permissions).map((value) => { const item = record(value) ? value : {}; return permission(item.permissionCode) ? { permissionCode: item.permissionCode, permissionName: string(item.permissionName) } : null; }).filter((item): item is EmployeeAccessList["permissions"][number] => item !== null) : [], calculatedAt: date(payload.calculatedAt)
  };
}
function mapAssignment(value: unknown): EmployeeRoleAssignment {
  const item = record(value) ? value : {};
  const roleStatus = string(item.roleStatus);
  return { assignmentId: string(item.assignmentId), roleCode: string(item.roleCode), roleName: string(item.roleName), roleStatus: roleStatus === "INACTIVE" || roleStatus === "ARCHIVED" ? roleStatus : "ACTIVE", assignmentStatus: assignmentStatus(item.assignmentStatus), scopeLabel: string(item.scopeLabel), effectiveFrom: date(item.effectiveFrom), effectiveUntil: typeof item.effectiveUntil === "string" ? date(item.effectiveUntil) : null, assignedByLabel: string(item.assignedByLabel) };
}
function mapEffectivePermission(value: unknown): EffectivePermission | null {
  const item = record(value) ? value : {};
  if (!permission(item.permissionCode)) return null;
  return { permissionCode: item.permissionCode, permissionName: string(item.permissionName), action: string(item.action), resource: string(item.resource), scopeLabels: array(item.scopeLabels).map((entry) => string(entry)).filter(Boolean), sources: array(item.sources).map((value) => { const source = record(value) ? value : {}; return { roleCode: string(source.roleCode), roleName: string(source.roleName), assignmentId: string(source.assignmentId) }; }).filter((source) => source.roleCode) };
}
function mapDetail(payload: Record<string, unknown>): EmployeeAccessDetail {
  const summary = mapSummary(payload, true);
  const profile = record(payload.profileReference) ? payload.profileReference : {};
  const identity = record(payload.identity) ? payload.identity : {};
  const lock = record(payload.lock) ? payload.lock : null;
  return {
    ...summary,
    profileReference: { employeeId: string(profile.employeeId, summary.employeeId), employeeCode: string(profile.employeeCode, summary.employeeCode), profileOwner: "EPIC_02" },
    identity: { emailVerified: identity.emailVerified === true, lastAuthenticatedAt: typeof identity.lastAuthenticatedAt === "string" ? date(identity.lastAuthenticatedAt) : null, credentialReadable: false },
    lock: lock ? { effectiveAt: date(lock.effectiveAt), reason: string(lock.reason), decidedByLabel: string(lock.decidedByLabel) } : null,
    roleAssignments: array(payload.roleAssignments).map(mapAssignment),
    effectivePermissions: array(payload.effectivePermissions).map(mapEffectivePermission).filter((item): item is EffectivePermission => item !== null),
    calculatedAt: date(payload.calculatedAt)
  };
}

function roleStatus(value: unknown): AdminRoleStatus {
  const status = string(value) as AdminRoleStatus;
  return ROLE_STATUSES.has(status) ? status : "ARCHIVED";
}
function roleKind(value: unknown): AdminRoleKind { return value === "CUSTOM" ? "CUSTOM" : "SYSTEM"; }
function mapAdminRoleSummary(value: unknown): AdminRoleSummary {
  const item = record(value) ? value : {};
  return { roleCode: string(item.roleCode), roleName: string(item.roleName), description: string(item.description), scopeLabel: string(item.scopeLabel), status: roleStatus(item.status), statusLabel: string(item.statusLabel), kind: roleKind(item.kind), privileged: item.privileged === true, employeeCount: Math.max(0, number(item.employeeCount)), permissionCount: Math.max(0, number(item.permissionCount)), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1)) };
}
function mapRoleList(payload: Record<string, unknown>): AdminRoleList {
  const items = array(payload.items).map(mapAdminRoleSummary).filter((item) => item.roleCode);
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | AdminRoleStatus; return { status: raw === "ALL" || ROLE_STATUSES.has(raw as AdminRoleStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  for (const status of ROLE_STATUSES) if (!statusCounts.some((item) => item.status === status)) statusCounts.push({ status, label: status, count: items.filter((item) => item.status === status).length });
  return { items, statusCounts, calculatedAt: date(payload.calculatedAt) };
}
function mapRoleDetail(payload: Record<string, unknown>): AdminRoleDetail {
  const summary = mapAdminRoleSummary(payload);
  const impact = record(payload.impact) ? payload.impact : {};
  return { ...summary, protectedRole: payload.protectedRole === true || summary.kind === "SYSTEM", permissions: array(payload.permissions).map((value) => { const item = record(value) ? value : {}; return permission(item.permissionCode) ? { permissionCode: item.permissionCode, permissionName: string(item.permissionName), action: string(item.action), resource: string(item.resource), scopeLabel: string(item.scopeLabel) } : null; }).filter((item): item is AdminRoleDetail["permissions"][number] => item !== null), impact: { activeEmployees: Math.max(0, number(impact.activeEmployees)), scheduledAssignments: Math.max(0, number(impact.scheduledAssignments)), lockedAccounts: Math.max(0, number(impact.lockedAccounts)) } };
}
function permissionStatus(value: unknown): AdminPermissionStatus {
  const status = string(value) as AdminPermissionStatus;
  return PERMISSION_STATUSES.has(status) ? status : "ARCHIVED";
}
function permissionKind(value: unknown): AdminPermissionKind { return value === "CONFIGURABLE" ? "CONFIGURABLE" : "SYSTEM"; }
function mapPermissionSummary(value: unknown): AdminPermissionSummary | null {
  const item = record(value) ? value : {};
  if (!permission(item.permissionCode)) return null;
  return { permissionCode: item.permissionCode, permissionName: string(item.permissionName), description: string(item.description), action: string(item.action), resource: string(item.resource), scopeLabel: string(item.scopeLabel), status: permissionStatus(item.status), statusLabel: string(item.statusLabel), kind: permissionKind(item.kind), roleCount: Math.max(0, number(item.roleCount)), employeeCount: Math.max(0, number(item.employeeCount)), updatedAt: date(item.updatedAt) };
}
function mapPermissionList(payload: Record<string, unknown>): AdminPermissionList {
  const items = array(payload.items).map(mapPermissionSummary).filter((item): item is AdminPermissionSummary => item !== null);
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | AdminPermissionStatus; return { status: raw === "ALL" || PERMISSION_STATUSES.has(raw as AdminPermissionStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  for (const status of PERMISSION_STATUSES) if (!statusCounts.some((item) => item.status === status)) statusCounts.push({ status, label: status, count: items.filter((item) => item.status === status).length });
  return { items, statusCounts, resources: array(payload.resources).map((item) => string(item)).filter(Boolean), calculatedAt: date(payload.calculatedAt) };
}
function mapPermissionDetail(payload: Record<string, unknown>): AdminPermissionDetail {
  const summary = mapPermissionSummary(payload);
  if (!summary) throw new UpstreamHttpError(502, { code: "ADMIN_PERMISSION_INVALID", message: "Projection Permission không hợp lệ." });
  return { ...summary, protectedCode: payload.protectedCode === true || summary.kind === "SYSTEM", usedByModules: array(payload.usedByModules).map((item) => string(item)).filter(Boolean), rolesUsing: array(payload.rolesUsing).map((value) => { const item = record(value) ? value : {}; return { roleCode: string(item.roleCode), roleName: string(item.roleName), status: roleStatus(item.status), employeeCount: Math.max(0, number(item.employeeCount)) }; }).filter((item) => item.roleCode) };
}

export async function handleAdminAccess(request: NextRequest, path: string[]) {
  const route = parseAdminAccessPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin access route không tồn tại.", 404);
  if (request.method !== "GET") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  try {
    const currentSession = await session(request);
    if (route.kind === "role-list" || route.kind === "role-detail") {
      if (!currentSession.permissions.includes("ROLE_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem Role.", 403);
      if (route.kind === "role-list") return NextResponse.json(mapRoleList(await upstream(request, "admin/access/roles")));
      return NextResponse.json(mapRoleDetail(await upstream(request, `admin/access/roles/${encodeURIComponent(route.roleCode)}`)));
    }
    if (route.kind === "permission-list" || route.kind === "permission-detail") {
      if (!currentSession.permissions.includes("PERMISSION_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem Permission.", 403);
      if (route.kind === "permission-list") return NextResponse.json(mapPermissionList(await upstream(request, "admin/access/permissions")));
      return NextResponse.json(mapPermissionDetail(await upstream(request, `admin/access/permissions/${encodeURIComponent(route.permissionCode)}`)));
    }
    if (!currentSession.permissions.includes("EMPLOYEE_ACCOUNT_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem tài khoản nội bộ.", 403);
    const canReview = currentSession.permissions.includes("ACCESS_REVIEW_VIEW");
    if (route.kind === "list") return NextResponse.json(mapList(await upstream(request, "admin/access/employees"), canReview));
    if (!canReview) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền Access Review.", 403);
    return NextResponse.json(mapDetail(await upstream(request, `admin/access/employees/${encodeURIComponent(route.employeeId)}`)));
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_ACCESS_ERROR", cause.body.message || "Không thể xử lý yêu cầu quản trị truy cập.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_ACCESS_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ quản trị truy cập nội bộ.", 503);
  }
}
