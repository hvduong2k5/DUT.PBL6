import { describe, expect, it } from "vitest";
import type { OwnedOrder } from "@/lib/api-contract";
import { mapMvpOrderDetail } from "./mvp";

const ORDER_ID = "11111111-1111-4111-8111-111111111111";
const DIGEST = "a".repeat(64);

function ownedOrder(overrides: Partial<OwnedOrder> = {}): OwnedOrder {
  return {
    order_id: ORDER_ID,
    operation_id: "22222222-2222-4222-8222-222222222222",
    order_code: "OM-TEST-1",
    customer_id: "33333333-3333-4333-8333-333333333333",
    warehouse_id: "HUE",
    payment_method: "VIETQR",
    status: "PAID",
    payment_status: "CONFIRMED",
    stock_status: "COMMITTED",
    subtotal: 255000,
    shipping_fee: 27000,
    final_amount: 282000,
    version: 4,
    ready_generation: 1,
    ready_authorized: true,
    operational_hold: false,
    has_active_case: false,
    payment_reference: "OMP0D8XDP3O74WJ",
    task_id: null,
    shipment_id: null,
    seal_code: null,
    payment_expires_at: "2026-11-14T06:00:00.000Z",
    updated_at: "2026-11-13T07:00:00.000Z",
    created_at: "2026-11-13T06:15:00.000Z",
    sla_due_at: null,
    completion_due_at: null,
    items_digest: DIGEST,
    compensation_status: "NOT_REQUIRED",
    cancellation_status: "NONE",
    allowed_actions: ["CONTACT_SUPPORT"],
    snapshot: {
      catalog_snapshot_id: "cat-1",
      catalog_expires_at: "2026-11-13T06:30:00.000Z",
      shipping_quote_reference: "ship-1",
      quote_id: "quote-1",
      cart_revision: 3,
      payment_method: "VIETQR",
      items: [
        { sku_code: "SKU-002-200", product_name: "Mè Xửng Dẻo Ngự Trà", quantity: 2, unit_price: 85000 },
        { sku_code: "MX-500", product_name: "Mè Xửng giòn", quantity: 1, unit_price: 85000 }
      ],
      address: {
        recipient_name: "Golden Audit",
        phone: "0900000333",
        street: "2 Le Loi",
        ward_code: "WARD-PN-016",
        ward_name: "Phường Phú Nhuận",
        province_code: "75",
        province_name: "Thành phố Huế"
      },
      subtotal: 255000,
      shipping_fee: 27000,
      final_amount: 282000,
      expires_at: "2026-11-13T06:30:00.000Z",
      shipping_expires_at: "2026-11-13T06:30:00.000Z"
    },
    ...overrides
  } as OwnedOrder;
}

describe("MVP order recipient address", () => {
  it("detail renders the recipient address with ward/province names when present", () => {
    const detail = mapMvpOrderDetail(ownedOrder());
    expect(detail.recipient?.addressDisplay).toBe("2 Le Loi, Phường Phú Nhuận, Thành phố Huế");
  });

  it("renders a province once when a historical snapshot has the same suffix on the ward", () => {
    const order = ownedOrder();
    order.snapshot!.address.ward_name = "Phường Phú Nhuận, Thành phố Huế";
    expect(mapMvpOrderDetail(order).recipient?.addressDisplay).toBe("2 Le Loi, Phường Phú Nhuận, Thành phố Huế");
  });

  it("detail falls back to address codes for orders frozen before names were captured", () => {
    const order = ownedOrder();
    const snapshot = order.snapshot!;
    order.snapshot = {
      ...snapshot,
      address: {
        recipient_name: snapshot.address.recipient_name,
        phone: snapshot.address.phone,
        street: snapshot.address.street,
        ward_code: snapshot.address.ward_code,
        province_code: snapshot.address.province_code
      }
    };
    const detail = mapMvpOrderDetail(order);
    expect(detail.recipient?.addressDisplay).toBe("2 Le Loi, WARD-PN-016, 75");
  });

});
