import { describe, expect, it } from "vitest";
import { parseAdminOrderPath } from "./validation";

describe("admin order route validation", () => {
  it("accepts only the list and single-order detail routes", () => {
    expect(parseAdminOrderPath([])).toEqual({ kind: "order-list" });
    expect(parseAdminOrderPath(["ord-001"])).toEqual({ kind: "order-detail", orderId: "ord-001" });
    expect(parseAdminOrderPath(["ord-001", "actions"]).kind).toBe("invalid");
  });
});
