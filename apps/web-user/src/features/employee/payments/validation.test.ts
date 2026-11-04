import { describe, expect, it } from "vitest";
import { parseAdminPaymentPath } from "./validation";

describe("admin payment route validation", () => {
  it("accepts only reconciliation case list and detail routes", () => {
    expect(parseAdminPaymentPath(["cases"])).toEqual({ kind: "case-list" });
    expect(parseAdminPaymentPath(["cases", "pay-case-001"])).toEqual({ kind: "case-detail", caseId: "pay-case-001" });
    expect(parseAdminPaymentPath(["cases", "pay-case-001", "resolve"]).kind).toBe("invalid");
  });
});
