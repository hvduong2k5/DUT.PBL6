import type { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { requireEmployeeSession } from "./server";

function request(cookie?: string): NextRequest {
  return { cookies: { get: (name: string) => name === "OMAMA_SESSION" && cookie ? { name, value: cookie } : undefined } } as unknown as NextRequest;
}

function json(data: unknown, status = 200) { return Response.json({ success: true, data }, { status }); }

afterEach(() => vi.unstubAllGlobals());

describe("Employee server authorization", () => {
  it("requires the HttpOnly session cookie", async () => {
    await expect(requireEmployeeSession(request())).rejects.toMatchObject({ status: 401, code: "AUTH_REQUIRED" });
  });

  it("rejects a Customer account", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(json({ authenticated: true })).mockResolvedValueOnce(json({ principalId: "CUST-01", principalType: "CUSTOMER", email: "customer@example.com", roles: [], permissions: [] })));
    await expect(requireEmployeeSession(request("trusted"))).rejects.toMatchObject({ status: 403, code: "EMPLOYEE_ACCOUNT_REQUIRED" });
  });

  it("projects Employee roles and permissions", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(json({ authenticated: true })).mockResolvedValueOnce(json({ principalId: "EMP-01", principalType: "EMPLOYEE", email: "staff@omama.vn", roles: [{ roleCode: "PACKING_STAFF", scopeType: "WAREHOUSE", scopeId: "WH-01" }], permissions: ["PACKING_WORKBENCH_VIEW", "PACKING_TASK_COMPLETE"] })));
    await expect(requireEmployeeSession(request("trusted"))).resolves.toMatchObject({ employee: { employeeId: "EMP-01" }, roles: ["PACKING_STAFF"], permissions: ["PACKING_WORKBENCH_VIEW", "PACKING_TASK_COMPLETE"], scopes: [{ resource: "PACKING_TASK", level: "ASSIGNED_ONLY" }] });
  });
});
