import { describe, expect, it } from "vitest";
import { mapCustomerCoreOrderDetail, mapCustomerCoreOrderListItem } from "./customer-core";

describe("customer core order mapping", () => {
  it("maps SHIPPING list status to the existing SHIPPED UI state", () => {
    expect(mapCustomerCoreOrderListItem({ order_id: "ORD-1", status: "SHIPPING", total_items: 2, final_amount: { currency_code: "VND", units: 345000, nanos: 0 }, created_at: "2026-10-15T08:30:00.000Z" })).toMatchObject({ orderId: "ORD-1", status: "SHIPPED", shippingStatus: "IN_TRANSIT" });
  });

  it("combines detail and tracking checkpoints", () => {
    const detail = mapCustomerCoreOrderDetail({
      order_id: "ORD-1", status: "PAID", shipping_address: { recipient_name: "An", phone_number: "0905", street_address: "123 Lê Duẩn", ward: "Thuận Hòa", district: "Huế", province: "Huế" },
      items: [{ sku_code: "MX-500", quantity: 2, price: { currency_code: "VND", units: 345000, nanos: 0 } }],
      subtotal_amount: { currency_code: "VND", units: 690000, nanos: 0 }, discount_amount: { currency_code: "VND", units: 0, nanos: 0 }, shipping_fee: { currency_code: "VND", units: 30000, nanos: 0 }, final_amount: { currency_code: "VND", units: 720000, nanos: 0 }, tracking_code: "GHN-1", created_at: "2026-10-15T08:30:00.000Z"
    }, { tracking_code: "GHN-1", carrier_name: "GHN", current_status: "IN_TRANSIT", checkpoints: [{ status: "IN_TRANSIT", timestamp: "2026-10-15T14:20:00.000Z", location: "Kho Huế", description: "Đã nhập kho" }] });
    expect(detail).toMatchObject({ orderId: "ORD-1", shipping: { status: "IN_TRANSIT", trackingCode: "GHN-1" } });
    expect(detail.timeline).toHaveLength(2);
  });
});
