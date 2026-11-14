import { describe, expect, it } from "vitest";
import { AdminOrderProjectionError, liveOrderDetail, liveOrderList } from "./live";

export function orderFixture() {
  return { order_id: "11111111-1111-4111-8111-111111111111", order_code: "O-TEST", status: "CONFIRMED_COD", payment_status: "CONFIRMED", payment_method: "COD", stock_status: "COMMITTED", customer_id: null, task_id: "task-reference", shipment_id: null, final_amount: 1200, item_count: 2, version: 3, operational_hold: false, has_active_case: false, sla_due_at: null, created_at: "2026-11-07T10:00:00Z", updated_at: "2026-11-07T11:00:00Z" };
}
function detailFixture() {
  return { order: orderFixture(), calculated_at: "2026-11-07T12:00:00Z", subtotal: 1000, shipping_fee: 200, discount: 0, recipient: null, lines: [{ sku_code: "SKU-1", product_name: "Snapshot name", quantity: 2, unit_price: 500, line_total: 1000 }], history: [] };
}
describe("live employee order projection", () => {
  it("uses provider states and snapshot values without claiming global totals or invented SLA", () => {
    const result = liveOrderList({ items: [orderFixture()], calculated_at: "2026-11-07T12:00:00Z", next_cursor: "signed-boundary" });
    expect(result.items[0]).toMatchObject({ status: "CONFIRMED", paymentStatus: "PAID", totalVnd: 1200, itemCount: 2, slaState: "UNAVAILABLE" });
    expect(result.nextCursor).toBe("signed-boundary");
    expect(result.statusCounts[0].count).toBe(1);
    const unknown = liveOrderList({ items: [{ ...orderFixture(), status: "NEW_PROVIDER_STATE", payment_status: "NEW_PAYMENT_STATE" }], calculated_at: "2026-11-07T12:00:00Z", next_cursor: null });
    expect(unknown.items[0]).toMatchObject({ status: "UNKNOWN", paymentStatus: "UNKNOWN" });
  });
  it("keeps recipients masked unless the owning service supplies authorized details", () => {
    const payload = detailFixture();
    const masked = liveOrderDetail(payload);
    expect(masked.recipient.masked).toBe(true);
    expect(masked.recipient.phoneDisplay).toContain("Đã ẩn");
    expect(masked.lines[0].productName).toBe("Snapshot name");
    expect(masked.packing.reference).toBe("task-reference");
    const visible = liveOrderDetail({ ...payload, recipient: { name: "Authorized name", phone: "0901234567", address: "Authorized street" } });
    expect(visible.recipient).toMatchObject({ masked: false, displayName: "Authorized name", phoneDisplay: "0901234567" });
  });
  it("labels the payment method on the detail projection", () => {
    expect(liveOrderDetail(detailFixture()).payment.methodLabel).toBe("COD");
    const vietqr = detailFixture();
    vietqr.order = { ...vietqr.order, payment_method: "VIETQR" };
    expect(liveOrderDetail(vietqr).payment.methodLabel).toBe("VietQR");
    const future = detailFixture();
    future.order = { ...future.order, payment_method: "B2B_BANK_TRANSFER" };
    expect(liveOrderDetail(future).payment.methodLabel).toBe("B2B_BANK_TRANSFER");
  });
  it("rejects malformed data instead of showing fallback success, zero totals or a fabricated timestamp", () => {
    for (const order of [{ ...orderFixture(), final_amount: "1200" }, { ...orderFixture(), version: 0 }, { ...orderFixture(), created_at: "" }, { ...orderFixture(), final_amount: 1.5 }]) {
      expect(() => liveOrderList({ items: [order], calculated_at: "2026-11-07T12:00:00Z", next_cursor: null })).toThrow(AdminOrderProjectionError);
    }
    expect(() => liveOrderList({ items: [], next_cursor: null })).toThrow(AdminOrderProjectionError);
    expect(() => liveOrderDetail({ ...detailFixture(), recipient: { name: "Incomplete" } })).toThrow(AdminOrderProjectionError);
  });
});
