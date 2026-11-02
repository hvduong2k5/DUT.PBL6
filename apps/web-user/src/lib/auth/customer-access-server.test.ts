import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { authorizeCustomerCapability, requireCustomerSession } from "./customer-access-server";

const ORIGINAL_IDENTITY_URL = process.env.IDENTITY_API_URL;
const ORIGINAL_EXTENSION_URL = process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL;

function json(body: unknown, status = 200): Response {
  return Response.json(body, { status });
}

describe("verified Customer access", () => {
  beforeEach(() => {
    process.env.IDENTITY_API_URL = "http://identity.test/api/v1";
    process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL = "http://capability.test/api/v1";
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    if (ORIGINAL_IDENTITY_URL === undefined) delete process.env.IDENTITY_API_URL;
    else process.env.IDENTITY_API_URL = ORIGINAL_IDENTITY_URL;
    if (ORIGINAL_EXTENSION_URL === undefined) delete process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL;
    else process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL = ORIGINAL_EXTENSION_URL;
  });

  it("returns 401 when no trusted session cookie exists", async () => {
    await expect(requireCustomerSession()).rejects.toMatchObject({ status: 401, code: "AUTH_REQUIRED" });
  });

  it("gives an Employee the Customer experience through the same principal", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ success: true, data: { isAuthenticated: true } }))
      .mockResolvedValueOnce(json({ success: true, data: {
        principalId: "PRINCIPAL-EMP-01",
        principalType: "EMPLOYEE",
        displayName: "Nhân viên kho",
        roles: ["CUSTOMER", "WAREHOUSE_STAFF"]
      } }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(requireCustomerSession("trusted-session")).resolves.toMatchObject({
      customerAccess: true,
      customer: { id: "PRINCIPAL-EMP-01" },
      principal: { principalType: "EMPLOYEE", customerId: "PRINCIPAL-EMP-01" }
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("keeps an explicit Customer mapping when Identity provides one", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce(json({ success: true, data: { isAuthenticated: true } }))
      .mockResolvedValueOnce(json({ success: true, data: {
        principal_id: "PRINCIPAL-EMP-01",
        principal_type: "EMPLOYEE",
        customer_id: "CUST-EMP-01",
        full_name: "Nhân viên kho",
        roles: ["CUSTOMER", "WAREHOUSE_STAFF"]
      } })));

    await expect(requireCustomerSession("trusted-session")).resolves.toMatchObject({
      customerAccess: true,
      customer: { id: "CUST-EMP-01" },
      principal: { principalType: "EMPLOYEE", customerId: "CUST-EMP-01" }
    });
  });

  it("keeps a missing capability as 403 after principal verification", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ success: true, data: { isAuthenticated: true } }))
      .mockResolvedValueOnce(json({ success: true, data: { principalId: "CUST-01", principalType: "CUSTOMER", displayName: "Khách hàng" } }))
      .mockResolvedValueOnce(json({ actor: "REGISTERED", capabilities: ["PROFILE_MANAGE"], version: 1 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(authorizeCustomerCapability("trusted-session", "ORDER_HISTORY_VIEW")).rejects.toMatchObject({
      status: 403,
      code: "CAPABILITY_FORBIDDEN"
    });
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});
