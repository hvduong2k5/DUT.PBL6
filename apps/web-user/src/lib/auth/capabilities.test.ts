import { describe, expect, it } from "vitest";
import { parseCustomerCapabilityProjection } from "./capabilities";

describe("customer capability projection", () => {
  it("accepts known actors and deduplicates known capabilities", () => {
    expect(parseCustomerCapabilityProjection({ actor: "REGISTERED", capabilities: ["PROFILE_MANAGE", "PROFILE_MANAGE", "REVIEW_CREATE"], version: 1 }))
      .toEqual({ actor: "REGISTERED", capabilities: ["PROFILE_MANAGE", "REVIEW_CREATE"], version: 1 });
  });

  it("rejects unknown actors and capabilities", () => {
    expect(parseCustomerCapabilityProjection({ actor: "ADMIN", capabilities: [], version: 1 })).toBeUndefined();
    expect(parseCustomerCapabilityProjection({ actor: "GUEST", capabilities: ["ADMIN_ALL"], version: 1 })).toBeUndefined();
  });
});
