import { expect, it } from "vitest";
import { isOwnedOrder, mapMvpOrderDetail } from "./mvp";

const order = {
  order_id: "10000000-0000-4000-8000-000000000053", operation_id: "20000000-0000-4000-8000-000000000053", order_code: "ORD-COD-53", customer_id: null, warehouse_id: "HUE", payment_method: "COD", status: "CONFIRMED_COD", payment_status: "UNPAID", stock_status: "COMMITTED", subtotal: 1000, shipping_fee: 100, final_amount: 1100, version: 1, ready_generation: 0, ready_authorized: false, operational_hold: false, has_active_case: false, payment_reference: "", task_id: null, shipment_id: null, seal_code: null, payment_expires_at: null, updated_at: "2026-11-21T12:00:00Z", created_at: "2026-11-21T12:00:00Z", sla_due_at: null, completion_due_at: null, items_digest: "a".repeat(64), compensation_status: "NOT_REQUIRED", cancellation_status: "NONE", allowed_actions: ["CONTACT_SUPPORT"]
};

it("reads guest and registered COD orders without a bank-transfer deadline", () => {
  for (const customer_id of [null, "30000000-0000-4000-8000-000000000053"]) {
    const value = { ...order, customer_id };
    expect(isOwnedOrder(value)).toBe(true);
    if (isOwnedOrder(value)) expect(mapMvpOrderDetail(value).payment.expiresAt).toBeNull();
  }
});
it("still requires a valid VietQR payment deadline", () => {
  expect(isOwnedOrder({ ...order, payment_method: "VIETQR", status: "PENDING_PAYMENT" })).toBe(false);
  expect(isOwnedOrder({ ...order, payment_expires_at: "invalid" })).toBe(false);
});
