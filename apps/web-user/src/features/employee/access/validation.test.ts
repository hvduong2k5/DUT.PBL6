import { describe, expect, it } from "vitest";
import { parseAdminAccessPath } from "./validation";

describe("admin access route validation", () => {
  it("accepts Employee, Role and Permission catalog routes only", () => {
    expect(parseAdminAccessPath(["employees"]).kind).toBe("list");
    expect(parseAdminAccessPath(["employees", "EMP-001"]).kind).toBe("detail");
    expect(parseAdminAccessPath(["roles"]).kind).toBe("role-list");
    expect(parseAdminAccessPath(["roles", "SYSTEM_ADMIN"]).kind).toBe("role-detail");
    expect(parseAdminAccessPath(["permissions"]).kind).toBe("permission-list");
    expect(parseAdminAccessPath(["permissions", "ROLE_VIEW"]).kind).toBe("permission-detail");
    expect(parseAdminAccessPath(["employees", "EMP-001", "permissions"]).kind).toBe("invalid");
  });
});
