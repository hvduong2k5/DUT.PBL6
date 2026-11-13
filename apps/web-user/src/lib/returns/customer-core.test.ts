import { describe, expect, it } from "vitest";
import { mapCustomerCoreReturnCreated, mapCustomerCoreReturnDetail, mapOrderToReturnEligibility, toCustomerCoreReturnRequest } from "./customer-core";

const order = {
  order_id: "ORD-20261015-0042",
  status: "DELIVERED",
  shipping_address: { recipient_name: "Nguyễn Văn An", phone_number: "0905123456", street_address: "12 Lê Lợi", ward: "Phú Hội", district: "Thuận Hóa", province: "Huế" },
  items: [{ sku_code: "MX-GION-500G", quantity: 2, price: { currency_code: "VND", units: 345000, nanos: 0 } }],
  subtotal_amount: { currency_code: "VND", units: 690000, nanos: 0 },
  discount_amount: { currency_code: "VND", units: 0, nanos: 0 },
  shipping_fee: { currency_code: "VND", units: 0, nanos: 0 },
  final_amount: { currency_code: "VND", units: 690000, nanos: 0 },
  tracking_code: null,
  created_at: "2026-10-15T08:30:00.000Z"
};

const ticket = {
  return_id: "RET-20261016-0012",
  order_id: "ORD-20261015-0042",
  status: "REQUESTED" as const,
  reason: "Bao bì bị rách",
  refund_amount: { currency_code: "VND", units: 345000, nanos: 0 },
  created_at: "2026-10-16T10:15:00.000Z"
};

describe("Customer API return adapter", () => {
  it("uses SKU as the single selectable return line", () => {
    const result = mapOrderToReturnEligibility(order);
    expect(result.lines[0]).toMatchObject({ lineId: "MX-GION-500G", maxReturnQty: 2 });
    expect(result.policyWindowEndsAt).toBeNull();
  });

  it("writes the customer API request shape", () => {
    expect(toCustomerCoreReturnRequest({
      orderId: order.order_id,
      lines: [{ lineId: "MX-GION-500G", quantity: 1 }],
      reasonCode: "QUALITY_ISSUE",
      details: "Bao bì sản phẩm bị rách.",
      evidenceMediaUrls: ["https://cdn.example.vn/evidence.jpg"],
      refundBankCode: "MBBANK",
      refundAccountNumber: "0905123456",
      refundAccountHolder: "NGUYEN VAN AN",
      idempotencyKey: "return-case-550e8400-e29b-41d4-a716-446655440000"
    })).toMatchObject({ order_id: order.order_id, sku_code: "MX-GION-500G", quantity: 1, evidence_media_urls: ["https://cdn.example.vn/evidence.jpg"] });
  });

  it("maps created and detail projections without inventing unavailable data", () => {
    expect(mapCustomerCoreReturnCreated(ticket).casePath).toBe("/after-sales/RET-20261016-0012");
    const detail = mapCustomerCoreReturnDetail(ticket);
    expect(detail.status).toBe("RETURN_REQUESTED");
    expect(detail.reason).toBe("Bao bì bị rách");
    expect(detail.items).toEqual([]);
    expect(detail.refund.amountVnd).toBe(345000);
  });
});
