import { describe, expect, it } from "vitest";
import { parseAdminSession } from "./types";

const employee = { employeeId: "EMP-001", displayName: "Lê Thị Đoan Trang", email: "trang@oma.vn", jobTitle: "Sales Manager", department: "Kinh doanh B2B" };

describe("admin session projection", () => {
  it("keeps roles and effective permissions separate", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["SALES_MANAGER"], permissions: ["B2B_REQUEST_VIEW", "B2B_QUOTE_ISSUE"], scopes: [{ resource: "B2B_REQUEST", level: "ALL" }], version: 1 }))
      .toMatchObject({ roles: ["SALES_MANAGER"], permissions: ["B2B_REQUEST_VIEW", "B2B_QUOTE_ISSUE"] });
  });

  it("rejects unknown permissions", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["SALES_MANAGER"], permissions: ["ADMIN_ALL"], scopes: [], version: 1 })).toBeUndefined();
  });
});
