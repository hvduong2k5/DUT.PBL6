import type { AdminPermissionDetail, AdminPermissionList, AdminRoleDetail, AdminRoleList, EmployeeAccessDetail, EmployeeAccessList } from "@/lib/access/types";
import { AdminAccessApiError } from "@/lib/access/types";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`/api/admin/access${path}`, { cache: "no-store", credentials: "include" });
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
  permission(permissionCode: string) { return request<AdminPermissionDetail>(`/permissions/${encodeURIComponent(permissionCode)}`); }
};
