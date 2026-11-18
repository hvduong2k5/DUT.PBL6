import { describe, expect, it } from "vitest";
import {
  isAwaitingPayment,
  orderDetailHref,
  parseOrderStatusFilter,
  paymentHref,
  PENDING_PAYMENT_ORDERS_HREF,
  summarizePendingPayment
} from "@/lib/orders/pending-payment";

describe("pending payment helpers", () => {
  it("treats only PENDING_PAYMENT orders as payable", () => {
    expect(isAwaitingPayment({ status: "PENDING_PAYMENT" })).toBe(true);
    for (const status of ["PAYMENT_FINALIZING", "PAID", "CONFIRMED_COD", "CANCELLED_TIMEOUT", "CANCELLED_BY_USER"]) {
      expect(isAwaitingPayment({ status })).toBe(false);
    }
  });

  it("builds links that stay on the same order", () => {
    expect(paymentHref("ord/1 2")).toBe("/payment/ord%2F1%202");
    expect(orderDetailHref("abc")).toBe("/orders/abc");
    expect(orderDetailHref("abc", { cancel: true })).toBe("/orders/abc?cancel=1");
    expect(PENDING_PAYMENT_ORDERS_HREF).toBe("/account/orders?status=PENDING_PAYMENT");
  });

  it("counts unpaid orders and marks a possibly longer list", () => {
    const item = (status: string) => ({ status }) as never;
    expect(summarizePendingPayment({ items: [], nextCursor: "" })).toEqual({ count: 0, hasMore: false, label: "0" });
    expect(summarizePendingPayment({ items: [item("PENDING_PAYMENT"), item("PAID"), item("PENDING_PAYMENT")], nextCursor: "" }))
      .toEqual({ count: 2, hasMore: false, label: "2" });
    expect(summarizePendingPayment({ items: [item("PENDING_PAYMENT")], nextCursor: "next" }))
      .toEqual({ count: 1, hasMore: true, label: "1+" });
  });

  it("parses the status filter from the URL and ignores unknown values", () => {
    const allowed = ["ALL", "PENDING_PAYMENT", "PAID"] as const;
    expect(parseOrderStatusFilter("PENDING_PAYMENT", allowed, "ALL")).toBe("PENDING_PAYMENT");
    expect(parseOrderStatusFilter("nope", allowed, "ALL")).toBe("ALL");
    expect(parseOrderStatusFilter(null, allowed, "ALL")).toBe("ALL");
  });
});
