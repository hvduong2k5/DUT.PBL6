import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { authorizeCustomerCapability, requireCustomerSession } from "./customer-access-server";

const ORIGINAL_IDENTITY_URL = process.env.IDENTITY_UPSTREAM_URL;
const ORIGINAL_EXTENSION_URL = process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL;

function json(body: unknown, status = 200): Response {
  return Response.json(body, { status });
}

function session(authenticated = true) {
  return json({ success: true, data: { authenticated, principalId: authenticated ? "PRINCIPAL-01" : null } });
}

function profile(principalType: "CUSTOMER" | "EMPLOYEE") {
  return json({ success: true, data: {
    principalId: principalType === "CUSTOMER" ? "CUST-01" : "EMP-01",
    principalType,
    email: principalType === "CUSTOMER" ? "customer@example.com" : "staff@omama.vn",
    displayName: principalType === "CUSTOMER" ? "Khách hàng" : "Nhân viên kho",
    roles: [{ roleCode: principalType }],
    permissions: principalType === "EMPLOYEE" ? ["INVENTORY_VIEW"] : []
  } });
}

describe("verified Customer access", () => {
  beforeEach(() => {
    process.env.IDENTITY_UPSTREAM_URL = "http://identity.test/api/v1";
    process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL = "http://capability.test/api/v1";
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    if (ORIGINAL_IDENTITY_URL === undefined) delete process.env.IDENTITY_UPSTREAM_URL;
    else process.env.IDENTITY_UPSTREAM_URL = ORIGINAL_IDENTITY_URL;
    if (ORIGINAL_EXTENSION_URL === undefined) delete process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL;
    else process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL = ORIGINAL_EXTENSION_URL;
  });

  it("returns 401 when no trusted session cookie exists", async () => {
    await expect(requireCustomerSession()).rejects.toMatchObject({ status: 401, code: "AUTH_REQUIRED" });
  });

  it("always rejects an Employee account from Customer operations", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(session()).mockResolvedValueOnce(profile("EMPLOYEE"));
    vi.stubGlobal("fetch", fetchMock);

    await expect(requireCustomerSession("trusted-session")).rejects.toMatchObject({
      status: 403,
      code: "EMPLOYEE_PURCHASE_NOT_ALLOWED"
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("accepts a Customer account from the verified Identity profile", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(session()).mockResolvedValueOnce(profile("CUSTOMER"));
    vi.stubGlobal("fetch", fetchMock);

    await expect(requireCustomerSession("trusted-session")).resolves.toMatchObject({
      customerAccess: true,
      customer: { id: "CUST-01" },
      principal: { principalType: "CUSTOMER", customerId: "CUST-01" }
    });
  });

  it("keeps a missing capability as 403 after Customer verification", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(session())
      .mockResolvedValueOnce(profile("CUSTOMER"))
      .mockResolvedValueOnce(json({ actor: "REGISTERED", capabilities: ["PROFILE_MANAGE"], version: 1 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(authorizeCustomerCapability("trusted-session", "ORDER_HISTORY_VIEW")).rejects.toMatchObject({
      status: 403,
      code: "CAPABILITY_FORBIDDEN"
    });
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});
