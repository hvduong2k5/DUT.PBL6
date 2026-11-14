import type { AdminPermissionDetail, AdminPermissionList, AdminRoleDetail, AdminRoleList, AssignRoleInput, CreateRoleInput, EmployeeAccessDetail, EmployeeAccessList, InvitationCreated, ResetPasswordInput, UpdateRoleInput } from "@/features/employee/access/types";
import { AdminAccessApiError } from "@/features/employee/access/types";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`/api/admin/access${path}`, { cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminAccessApiError(response.status, payload);
  return payload as T;
}

async function send<T>(method: string, path: string, body?: unknown): Promise<T> {
  const response = await fetch(`/api/admin/access${path}`, {
    method,
    cache: "no-store",
    credentials: "include",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminAccessApiError(response.status, payload);
  return payload as T;
}

export const adminAccessService = {
  list() { return request<EmployeeAccessList>("/employees"); },
  detail(employeeId: string) { return request<EmployeeAccessDetail>(`/employees/${encodeURIComponent(employeeId)}`); },
  roles() { return request<AdminRoleList>("/roles"); },
  role(roleCode: string) { return request<AdminRoleDetail>(`/roles/${encodeURIComponent(roleCode)}`); },
  permissions() { return request<AdminPermissionList>("/permissions"); },
  permission(permissionCode: string) { return request<AdminPermissionDetail>(`/permissions/${encodeURIComponent(permissionCode)}`); },
  createRole(input: CreateRoleInput) { return send<{ roleId: string }>("POST", "/roles", input); },
  updateRole(roleCode: string, input: UpdateRoleInput) { return send<{ updated: boolean }>("PATCH", `/roles/${encodeURIComponent(roleCode)}`, input); },
  deleteRole(roleCode: string) { return send<{ deleted: boolean }>("DELETE", `/roles/${encodeURIComponent(roleCode)}`); },
  replaceRolePermissions(roleCode: string, permissions: string[]) { return send<{ updated: boolean; permissionCount: number }>("PUT", `/roles/${encodeURIComponent(roleCode)}/permissions`, { permissions }); },
  assignRole(employeeId: string, input: AssignRoleInput) { return send<{ assignmentId: string }>("POST", `/employees/${encodeURIComponent(employeeId)}/assignments`, input); },
  revokeAssignment(employeeId: string, assignmentId: string) { return send<{ revoked: boolean }>("DELETE", `/employees/${encodeURIComponent(employeeId)}/assignments/${encodeURIComponent(assignmentId)}`); },
  resetPassword(employeeId: string, input: ResetPasswordInput) { return send<{ reset: boolean }>("POST", `/employees/${encodeURIComponent(employeeId)}/password-reset`, input); },
  revokeSessions(employeeId: string) { return send<{ revoked: boolean }>("POST", `/employees/${encodeURIComponent(employeeId)}/sessions-revoke`); },
  createInvitation(validDays: number) { return send<InvitationCreated>("POST", "/invitations", { validDays }); },
  revokeInvitation(invitationId: string) { return send<{ revoked: boolean }>("DELETE", `/invitations/${encodeURIComponent(invitationId)}`); }
};
