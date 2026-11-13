import { describe, expect, it } from "vitest";
import type { OwnedOrder } from "@/lib/api-contract";
import { mapMvpOrderDetail, mapMvpOrderListResponse } from "./mvp";

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

describe("MVP order mappers (F-36)", () => {
  it("list items carry preview lines and a total item count from the snapshot", () => {
    const list = mapMvpOrderListResponse({ items: [ownedOrder()], next_cursor: "" });
    expect(list.items).toHaveLength(1);
    expect(list.items[0].itemCount).toBe(3);
    expect(list.items[0].previewItems).toEqual([
      { name: "Mè Xửng Dẻo Ngự Trà", quantity: 2 },
      { name: "Mè Xửng giòn", quantity: 1 }
    ]);
  });

  it("detail renders the recipient address with ward/province names when present", () => {
    const detail = mapMvpOrderDetail(ownedOrder());
    expect(detail.recipient?.addressDisplay).toBe("2 Le Loi, Phường Phú Nhuận, Thành phố Huế");
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

  it("cancellation block exposes customer labels instead of raw codes", () => {
    const detail = mapMvpOrderDetail(ownedOrder());
    expect(detail.cancellation.statusLabel).toBe("Không có yêu cầu hủy");
    expect(detail.cancellation.compensationLabel).toBe("Không áp dụng");
    expect(detail.cancellation.message).not.toMatch(/backend|allowed_actions/i);
  });

  it("cancel policy copy speaks to the customer, not the backend", () => {
    const cancellable = mapMvpOrderDetail(ownedOrder({
      status: "PENDING_PAYMENT",
      payment_status: "UNPAID",
      allowed_actions: ["PAY", "CANCEL", "CONTACT_SUPPORT"]
    }));
    expect(cancellable.cancellation.message).toBe("Bạn có thể hủy đơn trực tiếp ở bước này.");
    const paid = mapMvpOrderDetail(ownedOrder({ allowed_actions: ["REQUEST_CANCELLATION", "CONTACT_SUPPORT"] }));
    expect(paid.cancellation.message).toContain("đội chăm sóc");
    expect(paid.cancellation.message).not.toMatch(/backend|capability/i);
  });
});
