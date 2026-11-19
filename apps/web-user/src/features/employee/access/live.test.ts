import { describe, expect, it } from "vitest";
import { AccessProjectionError, decodeAccessProjection } from "./live";

const item = { accountId: "account-1", employeeId: "employee-1", employeeCode: "E1", displayName: "Fixture",
  email: "fixture@example.test", department: "HR", jobTitle: "Staff", loginIdentifier: "fixture",
  status: "ACTIVE", statusLabel: "Hoạt động", invitationStatus: "ACCEPTED", roles: [],
  effectivePermissionCodes: ["ORDER_VIEW"], accessState: "ENABLED", accessStateLabel: "Có quyền",
  updatedAt: "2026-11-07T12:00:00Z", revision: 1 };
const projection = { contractVersion: "iam-admin-read.v1", capabilities: { accessReview: false },
  data: { items: [item], statusCounts: [], departments: ["HR"], roles: [],
    permissions: [{ permissionCode: "ORDER_VIEW", permissionName: "Xem đơn" }], calculatedAt: item.updatedAt } };
describe("IAM projection validation", () => {
  it("removes access review fields when the producer denies the capability", () => {
    const result = decodeAccessProjection("list", projection);
    expect(result).toMatchObject({ permissions: [], items: [{ effectivePermissionCodes: [] }] });
    expect(decodeAccessProjection("list", { ...projection, capabilities: { accessReview: true } })).toEqual(projection.data);
  });
  it("rejects unknown states, missing dates, invalid counters and malformed codes", () => {
    for (const change of [{ status: "SUSPENDED" }, { updatedAt: null }, { revision: 0 }, { effectivePermissionCodes: [" bad code "] }]) {
      expect(() => decodeAccessProjection("list", { ...projection, data: { ...projection.data, items: [{ ...item, ...change }] } })).toThrow(AccessProjectionError);
    }
  });
  it("preserves catalog codes outside navigation permissions without aliasing them", () => {
    const result = decodeAccessProjection("list", { ...projection, capabilities: { accessReview: true }, data: { ...projection.data, items: [{ ...item, effectivePermissionCodes: ["EMPLOYEE_PROFILE_VIEW", "USER_VIEW"] }] } });
    expect(result).toMatchObject({ items: [{ effectivePermissionCodes: ["EMPLOYEE_PROFILE_VIEW", "USER_VIEW"] }] });
  });
});
