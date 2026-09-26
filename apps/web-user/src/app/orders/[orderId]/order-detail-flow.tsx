"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { CancelReasonCode, OrderDetail } from "@/lib/orders/types";
import { OrderApiError } from "@/lib/orders/types";
import { orderService } from "@/services/order-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export function OrderDetailFlow({ orderId }: { orderId: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [order, setOrder] = useState<OrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [reasonCode, setReasonCode] = useState<CancelReasonCode | "">("");
  const [note, setNote] = useState("");
  const [cancelError, setCancelError] = useState("");
  const [cancelling, setCancelling] = useState(false);
  const [notice, setNotice] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try { setOrder(await orderService.detail(orderId, scenario)); }
    catch (cause) { setError(cause instanceof OrderApiError ? cause.message : "Không thể tải chi tiết Order."); }
    finally { setLoading(false); }
  }, [orderId, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  async function cancelOrder(event: FormEvent) {
    event.preventDefault();
    if (!reasonCode) { setCancelError("Hãy chọn lý do hủy."); return; }
    if (reasonCode === "OTHER" && note.trim().length < 5) { setCancelError("Hãy mô tả lý do khác tối thiểu 5 ký tự."); return; }
    setCancelling(true); setCancelError("");
    try {
      const result = await orderService.cancel(orderId, { reasonCode, note: note.trim() || undefined, idempotencyKey: `order-cancel-${globalThis.crypto.randomUUID()}` }, scenario);
      setOrder((current) => current ? { ...current, status: "CANCELLED", statusLabel: "Đã hủy", statusDescription: result.message, payment: result.refundRequired ? { ...current.payment, status: "REFUND_PENDING", label: "Chờ xử lý hoàn tiền" } : current.payment, cancellation: { canCancel: false, cancelBy: null, message: result.message, allowedReasons: [] }, timeline: [...current.timeline.map((item) => ({ ...item, state: "COMPLETED" as const })), { code: "ORDER_CANCELLED", label: "Order đã hủy", description: result.message, occurredAt: result.cancelledAt, state: "CURRENT" }] } : current);
      setNotice(result.message); setDialogOpen(false);
    } catch (cause) { setCancelError(cause instanceof OrderApiError ? cause.message : "Không thể hủy Order lúc này."); }
    finally { setCancelling(false); }
  }

  if (loading) return <main className="order-detail-page"><div className="order-detail-loading" aria-busy="true"><div /><div /><div /></div></main>;
  if (!order) return <main className="order-detail-page"><section className="orders-state error"><span>!</span><h1>Chưa thể mở Order</h1><p>{error || "Order không tồn tại hoặc bạn không có quyền xem."}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href="/track-order">Tra cứu Guest Order</Link></div></section></main>;

  return <main className="order-detail-page">
    <nav className="detail-breadcrumbs" aria-label="Đường dẫn"><Link href="/">Trang chủ</Link><span>›</span><Link href="/account/orders">Đơn hàng của tôi</Link><span>›</span><strong>#{order.orderNumber}</strong></nav>
    <section className="order-detail-hero"><div><span className="eyebrow">Order snapshot • {order.source}</span><h1>Order #{order.orderNumber}</h1><p>Đặt lúc {new Date(order.placedAt).toLocaleString("vi-VN", { dateStyle: "long", timeStyle: "short" })}</p></div><div className="order-detail-actions"><button type="button" className="secondary-button" onClick={() => window.print()}>In thông tin</button>{order.cancellation.canCancel ? <button type="button" className="order-danger-button" onClick={() => { setDialogOpen(true); setCancelError(""); }}>Hủy Order</button> : null}</div></section>
    {notice ? <div className="order-notice" role="status">{notice}</div> : null}
    <section className="order-status-grid"><article><span>▤</span><div><small>Trạng thái Order</small><strong>{order.statusLabel}</strong><p>{order.statusDescription}</p></div></article><article><span>◎</span><div><small>Trạng thái Payment</small><strong>{order.payment.label}</strong><p>{order.payment.method === "COD" ? "Payment thu khi giao hàng." : "Payment và Order được theo dõi riêng."}</p></div></article><article><span>▱</span><div><small>Trạng thái giao hàng</small><strong>{order.shipping.label}</strong><p>{order.shipping.estimatedDelivery ?? "Chưa có dự kiến giao hàng."}</p></div></article></section>

    <section className="order-journey"><header><div><span className="eyebrow">Hành trình Order</span><h2>Các mốc được phép công khai</h2></div>{order.shipping.trackingCode ? <b>{order.shipping.trackingCode}</b> : null}</header><ol>{order.timeline.map((event) => <li className={event.state.toLocaleLowerCase()} key={`${event.code}-${event.occurredAt}`}><span>{event.state === "UPCOMING" ? "○" : "✓"}</span><div><time>{new Date(event.occurredAt).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" })}</time><strong>{event.label}</strong><p>{event.description}</p></div></li>)}</ol></section>

    <div className="order-detail-layout"><div className="order-detail-main"><section className="order-detail-card"><header><span>◈</span><div><h2>Danh mục thức quà</h2><p>Snapshot không thay đổi theo Catalog hiện tại.</p></div></header><div className="order-lines">{order.lines.map((line) => <article key={line.lineId}><span aria-hidden="true">◇</span><div><strong>{line.productName}</strong><small>{line.skuLabel}</small><em>{VND.format(line.unitPriceVnd)} × {line.quantity}</em></div><b>{VND.format(line.lineSubtotalVnd)}</b></article>)}</div></section><section className="order-detail-card"><header><span>⌖</span><div><h2>Thông tin nhận hàng</h2><p>Dữ liệu đã được chốt khi tạo Order.</p></div></header><dl className="order-recipient"><div><dt>Người nhận</dt><dd>{order.recipient.fullName}</dd></div><div><dt>Điện thoại</dt><dd>{order.recipient.phoneDisplay}</dd></div><div><dt>Email</dt><dd>{order.recipient.emailDisplay}</dd></div><div><dt>Địa chỉ</dt><dd>{order.recipient.addressDisplay}</dd></div>{order.recipient.deliveryNote ? <div><dt>Ghi chú</dt><dd>{order.recipient.deliveryNote}</dd></div> : null}</dl></section></div>
      <aside><section className="order-total-card"><span className="eyebrow">Quyết toán Order</span><dl><div><dt>Tạm tính</dt><dd>{VND.format(order.pricing.subtotalVnd)}</dd></div><div><dt>Phí vận chuyển</dt><dd>{VND.format(order.pricing.shippingFeeVnd)}</dd></div><div><dt>Ưu đãi</dt><dd>-{VND.format(order.pricing.discountVnd)}</dd></div><div className="grand"><dt>Tổng thanh toán</dt><dd>{VND.format(order.pricing.totalVnd)}</dd></div></dl></section><section className="order-policy-card"><h2>Chính sách hủy hiện tại</h2><p>{order.cancellation.message}</p>{order.cancellation.cancelBy ? <small>Server sẽ kiểm tra lại eligibility khi gửi yêu cầu.</small> : null}<Link href="/track-order">Cần hỗ trợ tra cứu?</Link>{["DELIVERED", "COMPLETED"].includes(order.status) ? <Link className="order-after-sales-link" href={`/orders/${encodeURIComponent(order.orderId)}/after-sales/request`}>Tạo yêu cầu hậu mãi / đổi trả</Link> : null}</section></aside>
    </div>

    {dialogOpen ? <div className="order-dialog-backdrop" role="presentation" onMouseDown={() => !cancelling && setDialogOpen(false)}><section className="order-dialog" role="dialog" aria-modal="true" aria-labelledby="cancel-order-title" onMouseDown={(event) => event.stopPropagation()}><span className="order-dialog-icon">!</span><h2 id="cancel-order-title">Xác nhận hủy Order</h2><p>{order.cancellation.message}</p><form onSubmit={cancelOrder}><label><span>Lý do hủy <b>*</b></span><select value={reasonCode} onChange={(event) => { setReasonCode(event.target.value as CancelReasonCode); setCancelError(""); }}><option value="">Chọn lý do</option>{order.cancellation.allowedReasons.map((reason) => <option key={reason.code} value={reason.code}>{reason.label}</option>)}</select></label><label><span>Ghi chú</span><textarea value={note} maxLength={300} onChange={(event) => setNote(event.target.value)} placeholder="Thông tin bổ sung cho yêu cầu hủy…" /></label><div className="order-cancel-warning"><strong>Hủy Order đã paid không đồng nghĩa đã hoàn tiền.</strong><p>Nếu cần, hệ thống chỉ tạo nghĩa vụ hoàn tiền cho quy trình riêng.</p></div>{cancelError ? <div className="status-notice error" role="alert">{cancelError}</div> : null}<footer><button type="button" className="secondary-button" disabled={cancelling} onClick={() => setDialogOpen(false)}>Giữ Order</button><button className="order-danger-button" disabled={cancelling}>{cancelling ? "Đang kiểm tra…" : "Xác nhận hủy"}</button></footer></form></section></div> : null}
  </main>;
}
