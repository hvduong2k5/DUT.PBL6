import { describe, expect, it } from "vitest";
import { isOrderTrackingResponse } from "./mvp";

describe("minimal Order tracking contract", () => {
  it("accepts nullable shipment_id with authoritative order_status", () => {
    expect(isOrderTrackingResponse({ shipment_id: null, order_status: "PAID" })).toBe(true);
    expect(isOrderTrackingResponse({ shipment_id: "11111111-1111-4111-8111-111111111111", order_status: "SHIPPED" })).toBe(true);
  });

  it("rejects inferred timeline fields, invalid shipment IDs and missing status", () => {
    expect(isOrderTrackingResponse({ shipment_id: null, order_status: "PAID", checkpoints: [] })).toBe(false);
    expect(isOrderTrackingResponse({ shipment_id: "shipment-1", order_status: "SHIPPED" })).toBe(false);
    expect(isOrderTrackingResponse({ shipment_id: null })).toBe(false);
  });
});
