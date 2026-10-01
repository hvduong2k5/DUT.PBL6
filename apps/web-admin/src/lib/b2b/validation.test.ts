import { describe, expect, it } from "vitest";
import { parseAdminB2BPath, validateAdminB2BAction } from "./validation";

describe("admin B2B validation", () => {
  it("allows list, detail and popup action routes only", () => {
    expect(parseAdminB2BPath(["quote-requests"]).kind).toBe("list");
    expect(parseAdminB2BPath(["quote-requests", "req-1"]).kind).toBe("detail");
    expect(parseAdminB2BPath(["quote-requests", "req-1", "versions"]).kind).toBe("versions");
    expect(parseAdminB2BPath(["quote-requests", "req-1", "actions"]).kind).toBe("action");
    expect(parseAdminB2BPath(["quotes"]).kind).toBe("invalid");
  });
  it("requires a reason for request-info and reject", () => {
    expect(validateAdminB2BAction({ action: "REQUEST_INFO", reason: "ngắn", expectedRevision: 7, idempotencyKey: "admin-action-12345678" }).ok).toBe(false);
    expect(validateAdminB2BAction({ action: "REQUEST_INFO", reason: "Vui lòng bổ sung file logo gốc.", expectedRevision: 7, idempotencyKey: "admin-action-12345678" }).ok).toBe(true);
  });
  it("requires business confirmations before converting an accepted quote", () => {
    const base = { action: "CONVERT_ORDER", expectedRevision: 7, idempotencyKey: "admin-action-12345678" };
    expect(validateAdminB2BAction(base).ok).toBe(false);
    expect(validateAdminB2BAction({ ...base, invoiceSnapshotConfirmed: true, availabilityCheckAcknowledged: true }).ok).toBe(true);
  });
  it("requires a reason to withdraw an issued quote", () => {
    expect(validateAdminB2BAction({ action: "WITHDRAW", reason: "ngắn", expectedRevision: 7, idempotencyKey: "admin-action-12345678" }).ok).toBe(false);
    expect(validateAdminB2BAction({ action: "WITHDRAW", reason: "Khách yêu cầu điều chỉnh lại phạm vi giao hàng.", expectedRevision: 7, idempotencyKey: "admin-action-12345678" }).ok).toBe(true);
  });
  it("rejects a mutation without a positive expected revision", () => {
    expect(validateAdminB2BAction({ action: "ASSIGN", assigneeId: "EMP-01", expectedRevision: 0, idempotencyKey: "admin-action-12345678" }).ok).toBe(false);
    expect(validateAdminB2BAction({ action: "ASSIGN", assigneeId: "EMP-01", expectedRevision: 3, idempotencyKey: "admin-action-12345678" }).ok).toBe(true);
  });
});
