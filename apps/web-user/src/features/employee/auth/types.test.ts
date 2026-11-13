import { describe, expect, it } from "vitest";
import { parseAdminSession, toEmployeeSession } from "./types";
import type { IdentityProfile } from "@/lib/auth/identity";

const employee = { employeeId: "EMP-001", displayName: "Lê Thị Đoan Trang", email: "trang@oma.vn", jobTitle: "Sales Manager", department: "Kinh doanh B2B" };

describe("admin session projection", () => {
  it("keeps roles and effective permissions separate", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["SALES_MANAGER"], permissions: ["B2B_REQUEST_VIEW", "B2B_QUOTE_ISSUE", "PRODUCT_VIEW"], scopes: [{ resource: "B2B_REQUEST", level: "ALL" }], version: 1 }))
      .toMatchObject({ roles: ["SALES_MANAGER"], permissions: ["B2B_REQUEST_VIEW", "B2B_QUOTE_ISSUE", "PRODUCT_VIEW"] });
  });

  it("rejects unknown permissions", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["SALES_MANAGER"], permissions: ["ADMIN_ALL"], scopes: [], version: 1 })).toBeUndefined();
  });

  it("accepts role codes supplied by Access Management without hardcoding them in the UI", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["PACKING"], permissions: ["PACKING_TASK_VIEW"], scopes: [{ resource: "PACKING_TASK", level: "ASSIGNED_ONLY" }], version: 1 }))
      .toMatchObject({ roles: ["PACKING"] });
    expect(parseAdminSession({ authenticated: true, employee, roles: ["CUSTOM_OPERATIONS_ROLE"], permissions: ["ADMIN_DASHBOARD_VIEW"], scopes: [], version: 1 }))
      .toMatchObject({ roles: ["CUSTOM_OPERATIONS_ROLE"] });
  });

  it("accepts a system administrator with atomic administration permissions", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["SYSTEM_ADMIN"], permissions: ["EMPLOYEE_ACCOUNT_VIEW", "ACCESS_REVIEW_VIEW", "ROLE_ASSIGN"], scopes: [{ resource: "EMPLOYEE_ACCOUNT", level: "ALL" }], version: 1 }))
      .toMatchObject({ roles: ["SYSTEM_ADMIN"], permissions: ["EMPLOYEE_ACCOUNT_VIEW", "ACCESS_REVIEW_VIEW", "ROLE_ASSIGN"] });
  });

  it("uses one Warehouse role while permissions distinguish management from workbench", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["WAREHOUSE"], permissions: ["INVENTORY_VIEW", "INVENTORY_MANAGEMENT_VIEW", "BATCH_VIEW"], scopes: [{ resource: "INVENTORY", level: "ALL" }], version: 1 }))
      .toMatchObject({ roles: ["WAREHOUSE"], permissions: ["INVENTORY_VIEW", "INVENTORY_MANAGEMENT_VIEW", "BATCH_VIEW"] });
    expect(parseAdminSession({ authenticated: true, employee, roles: ["WAREHOUSE"], permissions: ["INVENTORY_VIEW", "WAREHOUSE_WORKBENCH_VIEW", "INVENTORY_RECEIPT_CREATE", "INVENTORY_ISSUE_CREATE", "INVENTORY_ADJUSTMENT_CREATE"], scopes: [{ resource: "INVENTORY", level: "ALL" }], version: 1 }))
      .toMatchObject({ roles: ["WAREHOUSE"], permissions: ["INVENTORY_VIEW", "WAREHOUSE_WORKBENCH_VIEW", "INVENTORY_RECEIPT_CREATE", "INVENTORY_ISSUE_CREATE", "INVENTORY_ADJUSTMENT_CREATE"] });
  });

  it("uses one Packing role while permissions and scopes distinguish management from workbench", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["PACKING"], permissions: ["PACKING_TASK_VIEW", "PACKING_MANAGEMENT_VIEW"], scopes: [{ resource: "PACKING_TASK", level: "ALL" }], version: 1 }))
      .toMatchObject({ roles: ["PACKING"], permissions: ["PACKING_TASK_VIEW", "PACKING_MANAGEMENT_VIEW"] });
    expect(parseAdminSession({ authenticated: true, employee, roles: ["PACKING"], permissions: ["PACKING_TASK_VIEW", "PACKING_WORKBENCH_VIEW", "PACKING_CHECKLIST_UPDATE", "PACKING_TASK_COMPLETE"], scopes: [{ resource: "PACKING_TASK", level: "ASSIGNED_ONLY" }], version: 1 }))
      .toMatchObject({ roles: ["PACKING"], permissions: ["PACKING_TASK_VIEW", "PACKING_WORKBENCH_VIEW", "PACKING_CHECKLIST_UPDATE", "PACKING_TASK_COMPLETE"] });
  });

  it("keeps Shipment view and sensitive-recipient permissions separate", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["SALES_MANAGER"], permissions: ["SHIPMENT_VIEW", "SHIPMENT_MANAGEMENT_VIEW"], scopes: [{ resource: "SHIPMENT", level: "ALL" }], version: 1 }))
      .toMatchObject({ permissions: ["SHIPMENT_VIEW", "SHIPMENT_MANAGEMENT_VIEW"] });
    expect(parseAdminSession({ authenticated: true, employee, roles: ["WAREHOUSE"], permissions: ["SHIPMENT_VIEW", "SHIPMENT_SENSITIVE_VIEW"], scopes: [{ resource: "SHIPMENT", level: "ASSIGNED_ONLY" }], version: 1 }))
      .toMatchObject({ permissions: ["SHIPMENT_VIEW", "SHIPMENT_SENSITIVE_VIEW"] });
    expect(parseAdminSession({ authenticated: true, employee, roles: ["DELIVERY"], permissions: ["SHIPMENT_VIEW", "DELIVERY_WORKBENCH_VIEW", "SHIPMENT_SENSITIVE_VIEW"], scopes: [{ resource: "SHIPMENT", level: "ASSIGNED_ONLY" }], version: 1 }))
      .toMatchObject({ roles: ["DELIVERY"], permissions: ["SHIPMENT_VIEW", "DELIVERY_WORKBENCH_VIEW", "SHIPMENT_SENSITIVE_VIEW"] });
  });

  it("accepts Accountant with separate Payment permissions", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["ACCOUNTANT"], permissions: ["PAYMENT_VIEW", "PAYMENT_RECONCILIATION_VIEW", "PAYMENT_SENSITIVE_VIEW"], scopes: [{ resource: "PAYMENT", level: "ALL" }], version: 1 }))
      .toMatchObject({ roles: ["ACCOUNTANT"], permissions: ["PAYMENT_VIEW", "PAYMENT_RECONCILIATION_VIEW", "PAYMENT_SENSITIVE_VIEW"] });
  });

  it("keeps Return Case, evidence and financial permissions separate", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["CUSTOMER_SERVICE"], permissions: ["RETURN_CASE_VIEW", "RETURN_EVIDENCE_VIEW"], scopes: [{ resource: "RETURN_CASE", level: "ASSIGNED_ONLY" }], version: 1 }))
      .toMatchObject({ permissions: ["RETURN_CASE_VIEW", "RETURN_EVIDENCE_VIEW"] });
    expect(parseAdminSession({ authenticated: true, employee, roles: ["ACCOUNTANT"], permissions: ["RETURN_CASE_VIEW", "RETURN_FINANCIAL_VIEW"], scopes: [{ resource: "RETURN_CASE", level: "ALL" }], version: 1 }))
      .toMatchObject({ permissions: ["RETURN_CASE_VIEW", "RETURN_FINANCIAL_VIEW"] });
  });

  it("accepts atomic Customer Service Ticket permissions", () => {
    expect(parseAdminSession({ authenticated: true, employee, roles: ["CUSTOMER_SERVICE"], permissions: ["TICKET_QUEUE_VIEW", "TICKET_CONVERSATION_VIEW", "TICKET_CONTEXT_VIEW", "TICKET_INTERNAL_NOTE_VIEW"], scopes: [{ resource: "SUPPORT_TICKET", level: "ASSIGNED_ONLY" }], version: 1 }))
      .toMatchObject({ permissions: ["TICKET_QUEUE_VIEW", "TICKET_CONVERSATION_VIEW", "TICKET_CONTEXT_VIEW", "TICKET_INTERNAL_NOTE_VIEW"] });
  });
});

describe("employee session projection from identity profiles", () => {
  const profileWith = (permissions: string[]): IdentityProfile => ({
    principalId: "emp-1",
    principalType: "EMPLOYEE",
    email: "staff@oma.vn",
    displayName: "Staff One",
    provisioningStatus: "ACTIVE",
    roles: [{ roleCode: "ADMIN", scopeType: "GLOBAL" }],
    permissions
  });

  it("expands catalog IAM grants into the catalog workspace draft permission names", () => {
    const session = toEmployeeSession(profileWith(["CATALOG_VIEW", "CATALOG_MANAGE", "ORDER_VIEW", "USER_VIEW"]));
    expect(session?.permissions).toEqual(expect.arrayContaining(["CATALOG_VIEW", "CATALOG_MANAGE", "PRODUCT_VIEW", "PRODUCT_MANAGE", "ORDER_VIEW"]));
    // USER_VIEW is outside the web session vocabulary and stays out.
    expect(session?.permissions).not.toContain("USER_VIEW");
  });

  it("does not invent draft permissions without the underlying IAM grant", () => {
    const session = toEmployeeSession(profileWith(["ORDER_VIEW"]));
    expect(session?.permissions).not.toContain("PRODUCT_VIEW");
    expect(session?.permissions).not.toContain("PRODUCT_MANAGE");
  });
});
