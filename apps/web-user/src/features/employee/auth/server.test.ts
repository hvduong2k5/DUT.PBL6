import type { NextRequest } from "next/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { requireEmployeeSession } from "./server";

function request(cookie?: string): NextRequest {
  return { cookies: { get: (name: string) => name === "OMAMA_SESSION" && cookie ? { name, value: cookie } : undefined } } as unknown as NextRequest;
}

function json(data: unknown, status = 200) { return Response.json({ success: true, data }, { status }); }

afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

describe("Employee server authorization", () => {
  beforeEach(() => { vi.stubEnv("WEB_AUTH_MODE", "identity-bff"); });
  it("requires the HttpOnly session cookie", async () => {
    await expect(requireEmployeeSession(request())).rejects.toMatchObject({ status: 401, code: "AUTH_REQUIRED" });
  });

  it("rejects a Customer account", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(json({ authenticated: true })).mockResolvedValueOnce(json({ principalId: "CUST-01", principalType: "CUSTOMER", email: "customer@example.com", roles: [], permissions: [] })));
    await expect(requireEmployeeSession(request("trusted"))).rejects.toMatchObject({ status: 403, code: "STAFF_ACCESS_REQUIRED" });
  });

  it("projects Employee roles and permissions", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(json({ authenticated: true })).mockResolvedValueOnce(json({ principalId: "EMP-01", principalType: "EMPLOYEE", email: "staff@omama.vn", roles: [{ roleCode: "PACKING_STAFF", scopeType: "WAREHOUSE", scopeId: "WH-01" }], permissions: ["PACKING_WORKBENCH_VIEW", "PACKING_TASK_COMPLETE"] })));
    await expect(requireEmployeeSession(request("trusted"))).resolves.toMatchObject({ employee: { employeeId: "EMP-01" }, roles: ["PACKING_STAFF"], permissions: ["PACKING_WORKBENCH_VIEW", "PACKING_TASK_COMPLETE"], scopes: [{ resource: "PACKING_TASK", level: "ASSIGNED_ONLY" }] });
  });
});

describe("Employee server authorization (OIDC mode)", () => {
  afterEach(() => vi.unstubAllEnvs());
  function oidcRequest(token?: string): NextRequest {
    return { cookies: { get: (name: string) => name === "oma_access_token" && token ? { name, value: token } : undefined } } as unknown as NextRequest;
  }
  function setupOidc() {
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "http://identity.test/api/v1/customer-api");
  }

  it("requires the access-token cookie", async () => {
    setupOidc();
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    await expect(requireEmployeeSession(oidcRequest())).rejects.toMatchObject({ status: 401, code: "AUTH_REQUIRED" });
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("projects Employee roles and permissions from the bearer profile", async () => {
    setupOidc();
    const fetcher = vi.fn().mockResolvedValue(json({ principalId: "EMP-02", principalType: "EMPLOYEE", email: "staff@omama.vn", roles: [{ roleCode: "ADMIN", scopeType: "GLOBAL" }], permissions: ["ORDER_VIEW", "PACKING_TASK_VIEW"] }));
    vi.stubGlobal("fetch", fetcher);
    await expect(requireEmployeeSession(oidcRequest("staff-token"))).resolves.toMatchObject({ employee: { employeeId: "EMP-02" }, roles: ["ADMIN"], permissions: ["ORDER_VIEW", "PACKING_TASK_VIEW"] });
    expect(fetcher).toHaveBeenCalledTimes(1);
    const [url, init] = fetcher.mock.calls[0];
    expect(url).toBe("http://identity.test/api/v1/customer-api/users/me");
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer staff-token");
  });

  it("rejects a Customer account", async () => {
    setupOidc();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(json({ principalId: "CUST-02", principalType: "CUSTOMER", email: "customer@example.com", roles: [], permissions: [] })));
    await expect(requireEmployeeSession(oidcRequest("customer-token"))).rejects.toMatchObject({ status: 403, code: "STAFF_ACCESS_REQUIRED" });
  });

  it("maps an expired token to AUTH_REQUIRED", async () => {
    setupOidc();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 401 })));
    await expect(requireEmployeeSession(oidcRequest("expired-token"))).rejects.toMatchObject({ status: 401, code: "AUTH_REQUIRED" });
  });
});
