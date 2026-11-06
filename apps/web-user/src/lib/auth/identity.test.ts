import { describe, expect, it } from "vitest";
import { parseIdentityProfile, parseIdentitySession } from "./identity";

describe("Identity contract parsers", () => {
  it("reads the wrapped session response", () => {
    expect(parseIdentitySession({ success: true, data: { authenticated: true, principalId: "P-01", email: "a@omama.vn" } })).toEqual({ authenticated: true, principalId: "P-01", email: "a@omama.vn" });
    expect(parseIdentitySession({ success: true, data: { isAuthenticated: false } })).toEqual({ authenticated: false });
  });

  it("keeps Customer and Employee as mutually exclusive principal types", () => {
    expect(parseIdentityProfile({ data: { principalId: "EMP-01", principalType: "EMPLOYEE", email: "staff@omama.vn", roles: [{ roleCode: "WAREHOUSE_STAFF" }], permissions: ["PACKING_WORKBENCH_VIEW"] } })).toMatchObject({ principalId: "EMP-01", principalType: "EMPLOYEE" });
    expect(parseIdentityProfile({ data: { principalId: "BAD-01", principalType: "BOTH" } })).toBeUndefined();
  });
});
