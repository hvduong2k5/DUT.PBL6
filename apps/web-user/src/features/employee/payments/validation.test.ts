import { describe, expect, it } from "vitest";
import { parseAdminPaymentPath } from "./validation";

describe("admin payment route validation", () => {
  it("accepts only reconciliation case routes", () => {
    expect(parseAdminPaymentPath(["cases"])).toEqual({ kind: "case-list" });
    expect(parseAdminPaymentPath(["cases", "pay-case-001"])).toEqual({ kind: "case-detail", caseId: "pay-case-001" });
    expect(parseAdminPaymentPath(["cases", "pay-case-001", "assign"])).toEqual({ kind: "case-assign", caseId: "pay-case-001" });
    expect(parseAdminPaymentPath(["cases", "pay-case-001", "resolve"])).toEqual({ kind: "case-resolve", caseId: "pay-case-001" });
    expect(parseAdminPaymentPath(["cases", "pay-case-001", "decisions"])).toEqual({ kind: "case-decisions", caseId: "pay-case-001" });
    expect(parseAdminPaymentPath(["cases", "pay-case-001", "bogus"]).kind).toBe("invalid");
    expect(parseAdminPaymentPath(["cases", "pay-case-001", "decisions", "extra"]).kind).toBe("invalid");
  });

  it("accepts the refund authority routes (F-42 slice 2)", () => {
    expect(parseAdminPaymentPath(["refunds"])).toEqual({ kind: "refund-list" });
    expect(parseAdminPaymentPath(["refunds", "rf-001"])).toEqual({ kind: "refund-detail", refundId: "rf-001" });
    expect(parseAdminPaymentPath(["refunds", "rf-001", "claim"])).toEqual({ kind: "refund-action", refundId: "rf-001", action: "claim" });
    expect(parseAdminPaymentPath(["refunds", "rf-001", "cancel-obligation"])).toEqual({ kind: "refund-action", refundId: "rf-001", action: "cancel-obligation" });
    expect(parseAdminPaymentPath(["refunds", "rf-001", "evidence"])).toEqual({ kind: "refund-evidence", refundId: "rf-001" });
    expect(parseAdminPaymentPath(["refunds", "rf-001", "verify"])).toEqual({ kind: "refund-verify", refundId: "rf-001" });
    expect(parseAdminPaymentPath(["refunds", "rf-001", "bogus"]).kind).toBe("invalid");
    expect(parseAdminPaymentPath(["refund-approvals"])).toEqual({ kind: "refund-approvals" });
    expect(parseAdminPaymentPath(["refund-approvals", "ap-001", "withdraw"])).toEqual({ kind: "refund-approval-withdraw", approvalId: "ap-001" });
    expect(parseAdminPaymentPath(["refund-approvals", "ap-001", "accept"])).toEqual({ kind: "refund-approval-accept", approvalId: "ap-001" });
    expect(parseAdminPaymentPath(["refund-approvals", "ap-001"]).kind).toBe("invalid");
    expect(parseAdminPaymentPath(["refund-evidence"])).toEqual({ kind: "refund-evidence-records" });
  });
});
