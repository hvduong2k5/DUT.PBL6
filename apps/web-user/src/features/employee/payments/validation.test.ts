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
    expect(parseAdminPaymentPath(["refunds"]).kind).toBe("invalid");
    expect(parseAdminPaymentPath(["cases", "pay-case-001", "decisions", "extra"]).kind).toBe("invalid");
  });
});
