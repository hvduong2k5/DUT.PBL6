import { describe, expect, it } from "vitest";
import { parseAdminReturnPath } from "./validation";

describe("admin return route validation", () => {
  it("accepts only Return Case list and detail routes", () => {
    expect(parseAdminReturnPath(["cases"])).toEqual({ kind: "case-list" });
    expect(parseAdminReturnPath(["cases", "ret-001"])).toEqual({ kind: "case-detail", caseId: "ret-001" });
    expect(parseAdminReturnPath(["cases", "ret-001", "approve"]).kind).toBe("invalid");
  });
});
