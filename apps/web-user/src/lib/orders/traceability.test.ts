import { describe, expect, it } from "vitest";
import { isOrderTraceability, matchesFrozenOrder, type OrderTraceability } from "./traceability";
const order = "11111111-1111-4111-8111-111111111111";
const line = "22222222-2222-4222-8222-222222222222";
const origin = "https://shop.example.test";
function response(): OrderTraceability {
  return { order_id: order, state: "READY", total: 2, items: [
    { order_line_id: line, sku_code: "SKU-1", batch_id: "33333333-3333-4333-8333-333333333333", batch_code: "LOT-A", packed_quantity: 3, status: "AVAILABLE", qr_url: origin + "/trace/abcdefghijklmnopqrstuv" },
    { order_line_id: line, sku_code: "SKU-1", batch_id: "44444444-4444-4444-8444-444444444444", batch_code: "LOT-B", packed_quantity: 2, status: "UNAVAILABLE", qr_url: null },
  ] };
}
describe("sealed order projection", () => {
  it("keeps mixed availability and split quantities using exact SKU fields", () => {
    const value = response();
    expect(isOrderTraceability(value, order, [origin])).toBe(true);
    expect(matchesFrozenOrder(value, [{ skuCode: "SKU-1", quantity: 5 }])).toBe(true);
    expect(matchesFrozenOrder(value, [{ skuCode: "SKU-1", quantity: 4 }])).toBe(false);
    expect(matchesFrozenOrder(value, [{ quantity: 5 }])).toBe(false);
  });
  it("rejects wrong order, duplicate rows, missing URLs, foreign URLs and ambiguous SKU mappings", () => {
    for (const change of [
      (v: OrderTraceability) => { v.order_id = line; },
      (v: OrderTraceability) => { v.items[1].batch_id = v.items[0].batch_id; },
      (v: OrderTraceability) => { v.items[0].qr_url = null; },
      (v: OrderTraceability) => { v.items[0].qr_url = "https://attacker.test/trace/token"; },
      (v: OrderTraceability) => { v.items[1].order_line_id = order; },
      (v: OrderTraceability) => { v.total = 1; },
    ]) { const value = response(); change(value); expect(isOrderTraceability(value, order, [origin])).toBe(false); }
  });
  it("requires empty pending states without manufacturing batches", () => {
    expect(isOrderTraceability({ order_id: order, state: "PACKING_PENDING", items: [], total: 0 }, order, [origin])).toBe(true);
    const value = response(); value.state = "UNAVAILABLE";
    expect(isOrderTraceability(value, order, [origin])).toBe(false);
  });
});
