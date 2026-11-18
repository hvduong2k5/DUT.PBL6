"use client";

import Link from "next/link";
import { PENDING_PAYMENT_ORDERS_HREF } from "@/lib/orders/pending-payment";
import { usePendingPaymentOrders } from "./use-pending-payment-orders";

/**
 * "You have unpaid orders" notice.
 * - variant "home": links to the orders page already filtered to unpaid orders.
 * - variant "orders": switches the list filter in place through `onShow`.
 */
export function PendingPaymentNotice({ variant, onShow }: { variant: "home" | "orders"; onShow?: () => void }) {
  const summary = usePendingPaymentOrders();
  if (!summary || summary.count === 0) return null;

  const message = variant === "home"
    ? <p><strong>Bạn có {summary.label} đơn hàng chưa thanh toán.</strong> Hoàn tất thanh toán để Gói Huế bắt đầu chuẩn bị quà cho bạn.</p>
    : <p><strong>Bạn có {summary.label} đơn hàng chưa thanh toán.</strong></p>;

  return <aside className={`gh-notice gh-notice--${variant}`} role="status" aria-live="polite">
    <span className="gh-emblem" aria-hidden="true" />
    {message}
    {variant === "orders" && onShow
      ? <button type="button" className="gh-notice__action" onClick={onShow}>Xem ngay</button>
      : <Link className="gh-notice__action" href={PENDING_PAYMENT_ORDERS_HREF}>Xem ngay</Link>}
  </aside>;
}
