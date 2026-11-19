import { afterEach, describe, expect, it, vi } from "vitest";
import { decodeAccessProjection } from "../live";
import type { AdminPermissionDetail, AdminPermissionList, AdminRoleDetail, AdminRoleList, EmployeeAccessDetail, EmployeeAccessList } from "../types";
import { AccessUpstreamError, composeAccessProjection, splitPermissionCode } from "./compose";

const BASE = "http://identity.test/api/v1";
const T0 = "2026-11-01T00:00:00.000Z";

const adminItem = { principalId: "p-admin", idpSubject: "kc-admin", email: "admin@example.com", firstName: "An", lastName: "Admin", principalType: "EMPLOYEE", provisioningStatus: "ACTIVE", createdAt: T0, roles: ["ADMIN"] };
const limitedItem = { principalId: "p-limited", idpSubject: "kc-limited", email: "limited@example.com", firstName: "Lim", lastName: "Ited", principalType: "EMPLOYEE", provisioningStatus: "SUSPENDED", createdAt: T0, roles: ["F01_LIMITED"] };

const adminDetail = {
  ...adminItem, updatedAt: "2026-11-10T00:00:00.000Z", permissions: ["USER_VIEW", "ROLE_VIEW"],
  roles: [{ assignmentId: "a1", roleId: "r-admin", roleCode: "ADMIN", roleName: "Administrator", scopeType: "GLOBAL", assignmentStatus: "ACTIVE", validFrom: T0, assignedByPrincipalId: "p-admin", createdAt: T0 }],
};
const limitedDetail = {
  ...limitedItem, updatedAt: "2026-11-10T00:00:00.000Z", permissions: ["USER_VIEW"],
  roles: [{ assignmentId: "a2", roleId: "r-limited", roleCode: "F01_LIMITED", roleName: "Limited", scopeType: "GLOBAL", assignmentStatus: "ACTIVE", validFrom: T0, assignedByPrincipalId: "p-admin", createdAt: T0 }],
};

const rolesPayload = [
  { roleId: "r-admin", roleCode: "ADMIN", roleName: "Administrator", description: "Full access", isProtected: true, isEnabled: true, version: 3, createdAt: T0, updatedAt: "2026-11-09T00:00:00.000Z", createdByPrincipalId: "p-admin", updatedByPrincipalId: "p-admin" },
  { roleId: "r-limited", roleCode: "F01_LIMITED", roleName: "Limited", description: "", isProtected: false, isEnabled: true, version: 1, createdAt: T0, updatedAt: "2026-11-08T00:00:00.000Z", createdByPrincipalId: "p-admin", updatedByPrincipalId: "p-admin" },
];

const assignmentsPayload = [
  { assignmentId: "a1", principalId: "p-admin", roleId: "r-admin", roleCode: "ADMIN", roleName: "Administrator", scopeType: "GLOBAL", assignmentStatus: "ACTIVE", validFrom: T0, assignedByPrincipalId: "p-admin", version: 1, createdAt: T0, updatedAt: T0 },
  { assignmentId: "a2", principalId: "p-limited", roleId: "r-limited", roleCode: "F01_LIMITED", roleName: "Limited", scopeType: "GLOBAL", assignmentStatus: "ACTIVE", validFrom: T0, assignedByPrincipalId: "p-admin", version: 1, createdAt: T0, updatedAt: T0 },
];

const envelope = (data: unknown) => ({ success: true, code: "OK", message: "", timestamp: T0, data });
const json = (body: unknown) => new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });

function stubUpstream() {
  const fetcher = vi.fn(async (input: unknown) => {
    const url = String(input);
    if (url === `${BASE}/admin/users?type=EMPLOYEE&size=100`) return json(envelope({ items: [adminItem, limitedItem], page: 0, size: 100, totalItems: 2, totalPages: 1 }));
    if (url === `${BASE}/admin/users/p-admin`) return json(envelope(adminDetail));
    if (url === `${BASE}/admin/users/p-limited`) return json(envelope(limitedDetail));
    if (url === `${BASE}/admin/roles`) return json(envelope(rolesPayload));
    if (url === `${BASE}/admin/roles/r-admin/permissions`) return json(envelope(["USER_VIEW", "ROLE_VIEW"]));
    if (url === `${BASE}/admin/roles/r-limited/permissions`) return json(envelope(["USER_VIEW"]));
    if (url === `${BASE}/admin/role-assignments?size=200`) return json(envelope(assignmentsPayload));
    // The live profile service answers 200 with a JSON null body while it holds no HR records.
    if (url === `${BASE}/admin/employees`) return new Response("null", { status: 200, headers: { "Content-Type": "application/json" } });
    throw new Error(`unexpected fetch: ${url}`);
  });
  vi.stubGlobal("fetch", fetcher);
  return fetcher;
}

afterEach(() => { vi.unstubAllGlobals(); });

describe("composeAccessProjection (F-37B administration composition)", () => {
  it("composes the employee list from identity users + details, tolerating the empty HR source", async () => {
    stubUpstream();
    const data = decodeAccessProjection("list", await composeAccessProjection("list", undefined, BASE, "token")) as EmployeeAccessList;
    expect(data.items).toHaveLength(2);
    const admin = data.items.find((i) => i.accountId === "p-admin")!;
    expect(admin.status).toBe("ACTIVE");
    expect(admin.accessState).toBe("ENABLED");
    expect(admin.effectivePermissionCodes).toEqual(["USER_VIEW", "ROLE_VIEW"]);
    expect(admin.roles).toEqual([{ roleCode: "ADMIN", roleName: "Administrator", status: "ACTIVE" }]);
    expect(admin.department).toBe("");
    expect(admin.employeeCode).toBe("");
    const limited = data.items.find((i) => i.accountId === "p-limited")!;
    expect(limited.status).toBe("DISABLED");
    expect(limited.accessState).toBe("BLOCKED_ACCOUNT");
    expect(data.statusCounts).toContainEqual({ status: "ALL", label: "Tất cả", count: 2 });
    expect(data.roles).toEqual([{ roleCode: "ADMIN", roleName: "Administrator" }, { roleCode: "F01_LIMITED", roleName: "Limited" }]);
    expect(data.permissions.map((p) => p.permissionCode).sort()).toEqual(["ROLE_VIEW", "USER_VIEW"]);
  });

  it("composes the employee detail with assignment sources for effective permissions", async () => {
    stubUpstream();
    const data = decodeAccessProjection("detail", await composeAccessProjection("detail", "p-admin", BASE, "token")) as EmployeeAccessDetail;
    expect(data.identity).toEqual({ credentialReadable: false });
    expect(data.lock).toBeNull();
    expect(data.roleAssignments).toHaveLength(1);
    expect(data.roleAssignments[0].scopeLabel).toBe("Toàn hệ thống");
    expect(data.roleAssignments[0].assignedByLabel).toBe("An Admin");
    const userView = data.effectivePermissions.find((p) => p.permissionCode === "USER_VIEW")!;
    expect(userView.action).toBe("VIEW");
    expect(userView.resource).toBe("USER");
    expect(userView.sources).toEqual([{ roleCode: "ADMIN", roleName: "Administrator", assignmentId: "a1" }]);
  });

  it("composes the role list with permission and employee counts from real assignments", async () => {
    stubUpstream();
    const data = decodeAccessProjection("role-list", await composeAccessProjection("role-list", undefined, BASE, "token")) as AdminRoleList;
    const admin = data.items.find((r) => r.roleCode === "ADMIN")!;
    expect(admin.kind).toBe("SYSTEM");
    expect(admin.privileged).toBe(true);
    expect(admin.permissionCount).toBe(2);
    expect(admin.employeeCount).toBe(1);
    const limited = data.items.find((r) => r.roleCode === "F01_LIMITED")!;
    expect(limited.kind).toBe("CUSTOM");
    expect(limited.employeeCount).toBe(1);
  });

  it("composes the role detail with its permissions and impact", async () => {
    stubUpstream();
    const data = decodeAccessProjection("role-detail", await composeAccessProjection("role-detail", "ADMIN", BASE, "token")) as AdminRoleDetail;
    expect(data.protectedRole).toBe(true);
    expect(data.permissions.map((p) => p.permissionCode)).toEqual(["USER_VIEW", "ROLE_VIEW"]);
    expect(data.impact.activeEmployees).toBe(1);
    expect(data.impact.lockedAccounts).toBe(0);
  });

  it("composes the permission catalogue as the union of role grants with usage counts", async () => {
    stubUpstream();
    const data = decodeAccessProjection("permission-list", await composeAccessProjection("permission-list", undefined, BASE, "token")) as AdminPermissionList;
    const userView = data.items.find((p) => p.permissionCode === "USER_VIEW")!;
    expect(userView.roleCount).toBe(2);
    expect(userView.employeeCount).toBe(2);
    const roleView = data.items.find((p) => p.permissionCode === "ROLE_VIEW")!;
    expect(roleView.roleCount).toBe(1);
    expect(roleView.employeeCount).toBe(1);
    expect(data.resources).toEqual(["ROLE", "USER"]);
  });

  it("composes the permission detail with the roles using it", async () => {
    stubUpstream();
    const data = decodeAccessProjection("permission-detail", await composeAccessProjection("permission-detail", "USER_VIEW", BASE, "token")) as AdminPermissionDetail;
    expect(data.rolesUsing.map((r) => r.roleCode).sort()).toEqual(["ADMIN", "F01_LIMITED"]);
    expect(data.usedByModules).toEqual(["Quản trị truy cập"]);
  });

  it("propagates an upstream 403 instead of fabricating data", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(null, { status: 403 })));
    await expect(composeAccessProjection("role-list", undefined, BASE, "token")).rejects.toMatchObject({ status: 403 });
    await expect(composeAccessProjection("role-list", undefined, BASE, "token")).rejects.toBeInstanceOf(AccessUpstreamError);
  });
});

describe("splitPermissionCode", () => {
  it("splits single and two-word actions", () => {
    expect(splitPermissionCode("ORDER_VIEW")).toEqual({ action: "VIEW", resource: "ORDER" });
    expect(splitPermissionCode("FULFILLMENT_TASK_READ_HISTORY")).toEqual({ action: "READ_HISTORY", resource: "FULFILLMENT_TASK" });
    expect(splitPermissionCode("USER_PASSWORD_RESET")).toEqual({ action: "PASSWORD_RESET", resource: "USER" });
  });
});
