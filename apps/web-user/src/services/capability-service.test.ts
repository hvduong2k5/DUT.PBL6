import { describe, expect, it } from "vitest";
import { capabilityFailureAction } from "./capability-service";

describe("capabilityFailureAction", () => {
  it("treats every 403 as a genuine authorization denial", () => {
    expect(capabilityFailureAction(403, "CAPABILITY_FORBIDDEN")).toBe("forbidden");
    expect(capabilityFailureAction(403, "AUTHORIZATION_DENIED")).toBe("forbidden");
    expect(capabilityFailureAction(403, "ANYTHING_ELSE")).toBe("forbidden");
  });

  it("treats only the explicit session codes on a 401 as a session failure", () => {
    expect(capabilityFailureAction(401, "AUTH_REQUIRED")).toBe("session");
    expect(capabilityFailureAction(401, "SESSION_EXPIRED")).toBe("session");
    expect(capabilityFailureAction(401, "SESSION_INVALID")).toBe("session");
  });

  it("never maps a bare or unknown 401 to forbidden or session", () => {
    expect(capabilityFailureAction(401, "UPSTREAM_REJECTED")).toBe("error");
    expect(capabilityFailureAction(401, "CAPABILITY_UNAVAILABLE")).toBe("error");
    expect(capabilityFailureAction(401, "")).toBe("error");
  });

  it("treats upstream and provider failures as generic errors", () => {
    expect(capabilityFailureAction(502, "UPSTREAM_AUTH_FAILED")).toBe("error");
    expect(capabilityFailureAction(503, "AUTH_PROVIDER_UNAVAILABLE")).toBe("error");
    expect(capabilityFailureAction(503, "CAPABILITY_PROJECTION_UNAVAILABLE")).toBe("error");
    expect(capabilityFailureAction(0, "")).toBe("error");
  });
});
