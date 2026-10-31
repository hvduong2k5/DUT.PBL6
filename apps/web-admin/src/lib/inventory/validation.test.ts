import { describe, expect, it } from "vitest";
import { parseAdminInventoryPath, validateInventoryAdjustment, validateInventoryIssue, validateInventoryReceipt } from "./validation";

describe("admin inventory route validation", () => {
  it("accepts management and Warehouse Workbench routes", () => {
    expect(parseAdminInventoryPath(["skus"]).kind).toBe("sku-list");
    expect(parseAdminInventoryPath(["skus", "sku-001"])).toEqual({ kind: "sku-detail", skuId: "sku-001" });
    expect(parseAdminInventoryPath(["workbench"]).kind).toBe("workbench-context");
    expect(parseAdminInventoryPath(["workbench", "receipts"]).kind).toBe("receipt-create");
    expect(parseAdminInventoryPath(["workbench", "issues"]).kind).toBe("issue-create");
    expect(parseAdminInventoryPath(["workbench", "adjustments"]).kind).toBe("adjustment-create");
    expect(parseAdminInventoryPath(["receipts"]).kind).toBe("invalid");
  });

  it("validates receipt batch data and chronological dates", () => {
    expect(validateInventoryReceipt({ skuId: "sku-1", batchMode: "NEW", batchCode: "LOT-01", manufacturingDate: "2026-10-01", expiresAt: "2027-01-01", quantity: 10, sourceReference: "PO-100", expectedRevision: 2, idempotencyKey: "receipt-operation-0001" }).ok).toBe(true);
    expect(validateInventoryReceipt({ skuId: "sku-1", batchMode: "NEW", batchCode: "LOT-01", manufacturingDate: "2027-01-01", expiresAt: "2026-10-01", quantity: 10, sourceReference: "PO-100", expectedRevision: 2, idempotencyKey: "receipt-operation-0002" }).ok).toBe(false);
  });

  it("requires an approved reason for non-order issue", () => {
    expect(validateInventoryIssue({ skuId: "sku-1", batchId: "batch-1", purpose: "ORDER", quantity: 2, reference: "ORD-100", expectedRevision: 2, idempotencyKey: "issue-operation-0001" }).ok).toBe(true);
    expect(validateInventoryIssue({ skuId: "sku-1", batchId: "batch-1", purpose: "APPROVED_NON_ORDER", quantity: 2, reference: "DOC-100", reason: "ngắn", expectedRevision: 2, idempotencyKey: "issue-operation-0002" }).ok).toBe(false);
  });

  it("uses counted quantity for inventory count adjustment", () => {
    expect(validateInventoryAdjustment({ skuId: "sku-1", batchId: "batch-1", kind: "COUNT", countedQuantity: 18, reference: "COUNT-100", reason: "Kiểm kê định kỳ tại kho", expectedRevision: 2, idempotencyKey: "adjust-operation-0001" }).ok).toBe(true);
    expect(validateInventoryAdjustment({ skuId: "sku-1", batchId: "batch-1", kind: "DAMAGE", quantity: 0, reference: "DMG-100", reason: "Bao bì hư hỏng do vận chuyển", expectedRevision: 2, idempotencyKey: "adjust-operation-0002" }).ok).toBe(false);
  });
});
