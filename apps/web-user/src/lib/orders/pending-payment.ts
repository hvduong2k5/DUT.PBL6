import type { OrderListResponse } from "@/lib/orders/types";

export const PENDING_PAYMENT_STATUS = "PENDING_PAYMENT";
export const PENDING_PAYMENT_ORDERS_HREF = `/account/orders?status=${PENDING_PAYMENT_STATUS}`;

/** An order the customer can still pay (or cancel): the backend keeps it in PENDING_PAYMENT until paid or expired. */
export function isAwaitingPayment(order: { status: string }): boolean {
  return order.status === PENDING_PAYMENT_STATUS;
}

export function paymentHref(orderId: string): string {
  return `/payment/${encodeURIComponent(orderId)}`;
}

/** `cancel` opens the cancel dialog as soon as the order detail has loaded. */
export function orderDetailHref(orderId: string, options: { cancel?: boolean } = {}): string {
  return `/orders/${encodeURIComponent(orderId)}${options.cancel ? "?cancel=1" : ""}`;
}

export interface PendingPaymentSummary {
  count: number;
  /** True when the first page was full, so the real number may be higher. */
  hasMore: boolean;
  /** Text for a notice, e.g. "2" or "20+". */
  label: string;
}

export function summarizePendingPayment(response: Pick<OrderListResponse, "items" | "nextCursor">): PendingPaymentSummary {
  const count = response.items.filter(isAwaitingPayment).length;
  const hasMore = Boolean(response.nextCursor);
  return { count, hasMore, label: hasMore ? `${count}+` : String(count) };
}

/** Reads the `status` query value of the orders page; anything unknown falls back to "ALL". */
export function parseOrderStatusFilter<T extends string>(value: string | null, allowed: readonly T[], fallback: T): T {
  return allowed.find((candidate) => candidate === value) ?? fallback;
}
