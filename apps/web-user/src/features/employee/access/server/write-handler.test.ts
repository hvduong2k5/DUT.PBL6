import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { handleLiveAdminAccess } from "./live-handler";

const BASE = "http://identity.test/api/v1";
const ROLE_ID = "6377edec-51e2-4d3f-ae8f-926b26d0e5ca";
const EMPLOYEE_ID = "de99e25d-c8bf-4e99-8438-87134d906195";

function writeRequest(method: string, path: string, body?: unknown, cookie = "oma_access_token=test-token") {
  return new NextRequest(`http://localhost:3000/api/admin/access/${path}`, {
    method,
    headers: { Cookie: cookie, ...(body === undefined ? {} : { "Content-Type": "application/json" }) },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

interface Call { url: string; method: string; body: unknown }
function router(responses: Record<string, { status: number; payload: unknown }>) {
  const calls: Call[] = [];
  const fetcher = vi.fn(async (url: string, options: { method?: string; body?: string }) => {
    const method = options.method ?? "GET";
    calls.push({ url, method, body: options.body ? JSON.parse(options.body) : undefined });
    const hit = responses[`${method} ${url}`];
    if (!hit) return new Response(JSON.stringify({ code: "UNEXPECTED", message: `${method} ${url}` }), { status: 500 });
    return Response.json(hit.payload, { status: hit.status });
  });
  return { fetcher, calls };
}

describe("live IAM write forwarding (F-47)", () => {
  beforeEach(() => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    vi.stubEnv("IDENTITY_UPSTREAM_URL", BASE);
  });
  afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

  it("creates a role and returns the new role id", async () => {
    const { fetcher, calls } = router({ [`POST ${BASE}/admin/roles`]: { status: 201, payload: { success: true, code: "SUCCESS", data: ROLE_ID } } });
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("POST", "roles", { roleCode: "WTEST", roleName: "Write Test", description: "demo" }), ["roles"]);
    expect(response.status).toBe(201);
    expect(await response.json()).toEqual({ roleId: ROLE_ID });
    expect(calls).toEqual([{ url: `${BASE}/admin/roles`, method: "POST", body: { roleCode: "WTEST", roleName: "Write Test", description: "demo" } }]);
  });

  it("rejects an invalid role code before calling upstream", async () => {
    const { fetcher } = router({});
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("POST", "roles", { roleCode: "lower case", roleName: "X" }), ["roles"]);
    expect(response.status).toBe(400);
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("resolves the role code to an id before replacing permissions", async () => {
    const { fetcher, calls } = router({
      [`GET ${BASE}/admin/roles`]: { status: 200, payload: { success: true, data: [{ roleId: ROLE_ID, roleCode: "WTEST" }] } },
      [`PUT ${BASE}/admin/roles/${ROLE_ID}/permissions`]: { status: 200, payload: { success: true, data: null } },
    });
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("PUT", "roles/WTEST/permissions", { permissions: ["USER_VIEW", "USER_VIEW", "CATALOG_VIEW"] }), ["roles", "WTEST", "permissions"]);
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ updated: true, permissionCount: 2 });
    expect(calls[1]).toEqual({ url: `${BASE}/admin/roles/${ROLE_ID}/permissions`, method: "PUT", body: { permissions: ["USER_VIEW", "CATALOG_VIEW"] } });
  });

  it("refuses the isEnabled toggle instead of silently dropping it (F-48)", async () => {
    const { fetcher } = router({});
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("PATCH", "roles/WTEST", { isEnabled: false }), ["roles", "WTEST"]);
    expect(response.status).toBe(400);
    expect((await response.json()).code).toBe("INVALID_ARGUMENT");
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("assigns a role by code with GLOBAL scope normalised to a null scopeId", async () => {
    const { fetcher, calls } = router({
      [`GET ${BASE}/admin/roles`]: { status: 200, payload: { success: true, data: [{ roleId: ROLE_ID, roleCode: "WTEST" }] } },
      [`POST ${BASE}/admin/role-assignments`]: { status: 200, payload: { success: true, data: "7141d9af-1cb9-45c9-a9af-68857ca4c253" } },
    });
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("POST", `employees/${EMPLOYEE_ID}/assignments`, { roleCode: "WTEST", scopeType: "GLOBAL" }), ["employees", EMPLOYEE_ID, "assignments"]);
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ assignmentId: "7141d9af-1cb9-45c9-a9af-68857ca4c253" });
    expect(calls[1].body).toEqual({ principalId: EMPLOYEE_ID, roleId: ROLE_ID, scopeType: "GLOBAL", scopeId: null });
  });

  it("requires a scopeId for scoped assignments", async () => {
    const { fetcher } = router({});
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("POST", `employees/${EMPLOYEE_ID}/assignments`, { roleCode: "WTEST", scopeType: "WAREHOUSE" }), ["employees", EMPLOYEE_ID, "assignments"]);
    expect(response.status).toBe(400);
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("propagates an upstream conflict with its code and message", async () => {
    const { fetcher } = router({
      [`GET ${BASE}/admin/roles`]: { status: 200, payload: { success: true, data: [{ roleId: ROLE_ID, roleCode: "WTEST" }] } },
      [`DELETE ${BASE}/admin/roles/${ROLE_ID}`]: { status: 409, payload: { success: false, code: "STATE_CONFLICT", message: "Cannot delete role WTEST because it is currently assigned to 1 active user(s)" } },
    });
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("DELETE", "roles/WTEST"), ["roles", "WTEST"]);
    expect(response.status).toBe(409);
    const payload = await response.json();
    expect(payload.code).toBe("STATE_CONFLICT");
    expect(payload.message).toContain("currently assigned");
  });

  it("requires the staff token and never calls upstream without one", async () => {
    const { fetcher } = router({});
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("POST", "roles", { roleCode: "WTEST", roleName: "X" }, ""), ["roles"]);
    expect(response.status).toBe(401);
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("creates an invitation and returns its tracking fields", async () => {
    const { fetcher } = router({
      [`POST ${BASE}/admin/employee-invitations`]: { status: 200, payload: { success: true, data: { invitationId: "814fd90b-08ac-4fb4-a04e-52eb444ad95c", principalId: "9821918d-de1f-4420-b66e-57274264f85", status: "PENDING", expiresAt: "2026-11-20T13:46:37Z" } } },
    });
    vi.stubGlobal("fetch", fetcher);
    const response = await handleLiveAdminAccess(writeRequest("POST", "invitations", { validDays: 7 }), ["invitations"]);
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ invitationId: "814fd90b-08ac-4fb4-a04e-52eb444ad95c", principalId: "9821918d-de1f-4420-b66e-57274264f85", status: "PENDING", expiresAt: "2026-11-20T13:46:37Z" });
  });
});
