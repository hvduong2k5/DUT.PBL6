import { describe, expect, it } from "vitest";
import { parseAdminShippingPath } from "./validation";

describe("admin shipping route validation", () => {
  it("separates management and assigned Delivery Workbench routes", () => {
    expect(parseAdminShippingPath(["shipments"])).toEqual({ kind: "shipment-list" });
    expect(parseAdminShippingPath(["shipments", "ship-001"])).toEqual({ kind: "shipment-detail", shipmentId: "ship-001" });
    expect(parseAdminShippingPath(["workbench", "shipments"])).toEqual({ kind: "workbench-list" });
    expect(parseAdminShippingPath(["workbench", "shipments", "ship-001"])).toEqual({ kind: "workbench-detail", shipmentId: "ship-001" });
    expect(parseAdminShippingPath(["shipments", "ship-001", "cancel"]).kind).toBe("invalid");
  });
});
