import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { authorizeCustomerCapability, requireCustomerSession } from "./customer-access-server";

const ORIGINAL_CORE_URL = process.env.CUSTOMER_CORE_UPSTREAM_URL;
const ORIGINAL_EXTENSION_URL = process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL;

function json(body: unknown, status = 200): Response {
  return Response.json(body, { status });
}

describe("verified Customer access", () => {
  beforeEach(() => {
    process.env.CUSTOMER_CORE_UPSTREAM_URL = "http://core.test/api/v1";
    process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL = "http://capability.test/api/v1";
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    if (ORIGINAL_CORE_URL === undefined) delete process.env.CUSTOMER_CORE_UPSTREAM_URL;
    else process.env.CUSTOMER_CORE_UPSTREAM_URL = ORIGINAL_CORE_URL;
    if (ORIGINAL_EXTENSION_URL === undefined) delete process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL;
    else process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL = ORIGINAL_EXTENSION_URL;
  });

  it("returns 401 when no trusted access token exists", async () => {
    await expect(requireCustomerSession()).rejects.toMatchObject({ status: 401, code: "AUTH_REQUIRED" });
  });

  it("returns 403 when an Employee has no backend Customer mapping", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(json({
      principal_id: "PRINCIPAL-EMP-01",
      principal_type: "EMPLOYEE",
      full_name: "Nhân viên kho",
      roles: ["CUSTOMER", "WAREHOUSE_STAFF"]
    })));

    await expect(requireCustomerSession("trusted-token")).rejects.toMatchObject({
      status: 403,
      code: "CUSTOMER_MAPPING_REQUIRED"
    });
  });

  it("accepts an Employee only when Identity returns a Customer mapping", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(json({
      principal_id: "PRINCIPAL-EMP-01",
      principal_type: "EMPLOYEE",
      customer_id: "CUST-EMP-01",
      full_name: "Nhân viên kho",
      roles: ["CUSTOMER", "WAREHOUSE_STAFF"]
    })));

    await expect(requireCustomerSession("trusted-token")).resolves.toMatchObject({
      customerAccess: true,
      customer: { id: "CUST-EMP-01" },
      principal: { principalType: "EMPLOYEE", customerId: "CUST-EMP-01" }
    });
  });

  it("keeps a missing capability as 403 after principal verification", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ customer_id: "CUST-01", full_name: "Khách hàng" }))
      .mockResolvedValueOnce(json({ actor: "REGISTERED", capabilities: ["PROFILE_MANAGE"], version: 1 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(authorizeCustomerCapability("trusted-token", "ORDER_HISTORY_VIEW")).rejects.toMatchObject({
      status: 403,
      code: "CAPABILITY_FORBIDDEN"
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
