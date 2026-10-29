"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { createIdempotencyKey, PublicApiError } from "@/lib/api-contract";
import type { CancelReasonCode, OrderDetail } from "@/lib/orders/types";
import { orderService } from "@/services/order-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

function lineAmount(value: number | undefined): string {
  return value === undefined ? "Không có trong snapshot" : VND.format(value);
}

export function OrderDetailFlow({ orderId }: { orderId: string }) {
  const { hasCapability } = useCustomerSession();
  const canManageOrder = hasCapability("ORDER_HISTORY_VIEW");
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const cancelCommandRef = useRef<{ signature: string; key: string } | undefined>(undefined);
  const [order, setOrder] = useState<OrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [trackingState, setTrackingState] = useState<"idle" | "loading" | "available" | "unavailable">("idle");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [reasonCode, setReasonCode] = useState<CancelReasonCode | "">("");
  const [note, setNote] = useState("");
  const [cancelError, setCancelError] = useState("");
  const [cancelling, setCancelling] = useState(false);
  const [notice, setNotice] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const detail = await orderService.detail(orderId, scenario);
      setOrder(detail);
      if (detail.cancellation.canViewTracking) {
        setTrackingState("loading");
        try {
          await orderService.tracking(orderId, scenario);
          setTrackingState("available");
        } catch {
          setTrackingState("unavailable");
        }
      } else {
        setTrackingState("idle");
      }
    } catch (cause) {
      setError(cause instanceof PublicApiError ? cause.message : "Không thể tải chi tiết Order.");
      setOrder(null);
    } finally {
      setLoading(false);
    }
  }, [orderId, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  async function cancelOrder(event: FormEvent) {
    event.preventDefault();
    if (!order || !reasonCode) {
      setCancelError("Hãy chọn lý do hủy.");
      return;
    }
    if (reasonCode === "OTHER" && note.trim().length < 5) {
      setCancelError("Hãy mô tả lý do khác tối thiểu 5 ký tự.");
      return;
    }
    const signature = `${order.version}:${reasonCode}:${note.trim()}`;
    if (!cancelCommandRef.current || cancelCommandRef.current.signature !== signature) {
      cancelCommandRef.current = { signature, key: createIdempotencyKey("order-cancel") };
    }
    setCancelling(true);
    setCancelError("");
    try {
      const result = await orderService.cancel(orderId, {
        reasonCode,
        note: note.trim() || undefined,
        version: order.version,
        idempotencyKey: cancelCommandRef.current.key
      }, scenario);
      setNotice(result.message);
      setDialogOpen(false);
      cancelCommandRef.current = undefined;
      await load();
    } catch (cause) {
      if (cause instanceof PublicApiError) {
        setCancelError(cause.status === 409
          ? "Order đã đổi version hoặc không còn đủ điều kiện hủy. Dữ liệu đang được tải lại."
          : cause.message);
        if (cause.status === 409) {
          cancelCommandRef.current = undefined;
          await load();
        }
      } else {
        setCancelError("Không thể xác định kết quả hủy. Hãy thử lại với cùng nội dung để giữ nguyên Idempotency-Key.");
      }
    } finally {
      setCancelling(false);
    }
  }

  if (loading) {
    return <main className="order-detail-page"><div className="order-detail-loading" aria-busy="true"><div /><div /><div /></div></main>;
  }
  if (!order) {
    return <main className="order-detail-page"><section className="orders-state error"><span>!</span><h1>Chưa thể mở Order</h1><p>{error || "Order không tồn tại hoặc bạn không có quyền xem."}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href="/login?returnUrl=/account/orders">Đăng nhập lại</Link></div></section></main>;
  }

  return <main className="order-detail-page">
    <nav className="detail-breadcrumbs" aria-label="Đường dẫn"><Link href="/">Trang chủ</Link><span>›</span><Link href="/account/orders">Đơn hàng của tôi</Link><span>›</span><strong>#{order.orderNumber}</strong></nav>
    <section className="order-detail-hero"><div><span className="eyebrow">Order version {order.version}</span><h1>Order #{order.orderNumber}</h1><p>Tạo lúc {new Date(order.placedAt).toLocaleString("vi-VN", { dateStyle: "long", timeStyle: "short" })}</p></div><div className="order-detail-actions"><button type="button" className="secondary-button" onClick={() => window.print()}>In thông tin</button>{canManageOrder && order.cancellation.canCancel ? <button type="button" className="order-danger-button" onClick={() => { setDialogOpen(true); setCancelError(""); }}>Hủy Order</button> : null}</div></section>
    {notice ? <div className="order-notice" role="status">{notice}</div> : null}
    {order.operationalHold || order.hasActiveCase ? <div className="checkout-inline-error" role="status"><strong>Order đang cần lưu ý</strong><p>{order.statusDescription}</p></div> : null}

    <section className="order-status-grid">
      <article><span>▤</span><div><small>Trạng thái Order</small><strong>{order.statusLabel}</strong><p>{order.statusDescription}</p></div></article>
      <article><span>◎</span><div><small>Trạng thái Payment</small><strong>{order.payment.label}</strong><p>{order.payment.method} · hết hạn {new Date(order.payment.expiresAt).toLocaleString("vi-VN")}</p></div></article>
      <article><span>▱</span><div><small>Trạng thái Inventory</small><strong>{order.stock.label}</strong><p>{order.stock.status}</p></div></article>
    </section>

    <section className="order-journey"><header><div><span className="eyebrow">Hành trình Order</span><h2>Các mốc có trong public contract</h2></div>{order.shipmentId ? <b>Shipment: {order.shipmentId}</b> : null}</header><ol>{order.timeline.map((event) => <li className={event.state.toLocaleLowerCase()} key={`${event.code}-${event.occurredAt}`}><span>✓</span><div><time>{new Date(event.occurredAt).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" })}</time><strong>{event.label}</strong><p>{event.description}</p></div></li>)}</ol>{order.cancellation.canViewTracking ? <p className="cart-checkout-note" role="status">{trackingState === "loading" ? "Đang đọc tracking…" : trackingState === "available" ? "Tracking endpoint khả dụng. Baseline hiện chỉ định response là object nên UI không tự suy diễn checkpoint/carrier fields." : "Tracking hiện chưa khả dụng; Order detail vẫn được giữ độc lập."}</p> : null}</section>

    <div className="order-detail-layout"><div className="order-detail-main">
      <section className="order-detail-card"><header><span>◈</span><div><h2>Danh mục thức quà</h2><p>{order.snapshotAvailable ? "Immutable Checkout snapshot." : "Backend chưa trả snapshot trong Order này."}</p></div></header>{order.lines.length ? <div className="order-lines">{order.lines.map((line) => <article key={line.lineId}><span aria-hidden="true">◇</span><div><strong>{line.productName}</strong><small>{line.skuLabel}</small><em>{lineAmount(line.unitPriceVnd)} × {line.quantity}</em></div><b>{lineAmount(line.lineSubtotalVnd)}</b></article>)}</div> : <div className="orders-state"><p>Không có dòng sản phẩm trong snapshot public.</p></div>}</section>
      <section className="order-detail-card"><header><span>⌖</span><div><h2>Thông tin nhận hàng</h2><p>Dữ liệu chỉ hiển thị khi Order có immutable snapshot.</p></div></header>{order.recipient ? <dl className="order-recipient"><div><dt>Người nhận</dt><dd>{order.recipient.fullName}</dd></div><div><dt>Điện thoại</dt><dd>{order.recipient.phoneDisplay}</dd></div><div><dt>Địa chỉ</dt><dd>{order.recipient.addressDisplay}</dd></div></dl> : <div className="orders-state"><p>Order response không chứa snapshot địa chỉ.</p></div>}</section>
    </div><aside>
      <section className="order-total-card"><span className="eyebrow">Số tiền canonical</span><dl><div><dt>Tạm tính</dt><dd>{VND.format(order.pricing.subtotalVnd)}</dd></div><div><dt>Phí vận chuyển</dt><dd>{VND.format(order.pricing.shippingFeeVnd)}</dd></div><div className="grand"><dt>Tổng thanh toán</dt><dd>{VND.format(order.pricing.totalVnd)}</dd></div></dl></section>
      <section className="order-policy-card"><h2>Trạng thái hủy</h2><p>{order.cancellation.message}</p><small>Cancellation: {order.cancellation.status} · Compensation: {order.cancellation.compensationStatus}</small><Link href="/support/request">Cần hỗ trợ Order?</Link>{canManageOrder && ["DELIVERED", "COMPLETED"].includes(order.status) ? <Link className="order-after-sales-link" href={`/orders/${encodeURIComponent(order.orderId)}/after-sales/request`}>Tạo yêu cầu hậu mãi / đổi trả</Link> : null}</section>
    </aside></div>

    {dialogOpen ? <div className="order-dialog-backdrop" role="presentation" onMouseDown={() => !cancelling && setDialogOpen(false)}><section className="order-dialog" role="dialog" aria-modal="true" aria-labelledby="cancel-order-title" onMouseDown={(event) => event.stopPropagation()}><span className="order-dialog-icon">!</span><h2 id="cancel-order-title">Gửi lệnh hủy Order</h2><p>{order.cancellation.message}</p><form onSubmit={cancelOrder}><label><span>Lý do hủy <b>*</b></span><select value={reasonCode} onChange={(event) => { setReasonCode(event.target.value as CancelReasonCode); setCancelError(""); }}><option value="">Chọn lý do</option>{order.cancellation.allowedReasons.map((reason) => <option key={reason.code} value={reason.code}>{reason.label}</option>)}</select></label><label><span>Ghi chú</span><textarea value={note} maxLength={220} onChange={(event) => setNote(event.target.value)} placeholder="Thông tin bổ sung cho yêu cầu hủy…" /></label><div className="order-cancel-warning"><strong>Response `PENDING` chưa có nghĩa Order đã hủy.</strong><p>UI sẽ đọc lại cancellation/compensation status từ Order service.</p></div>{cancelError ? <div className="status-notice error" role="alert">{cancelError}</div> : null}<footer><button type="button" className="secondary-button" disabled={cancelling} onClick={() => setDialogOpen(false)}>Giữ Order</button><button className="order-danger-button" disabled={cancelling}>{cancelling ? "Đang gửi…" : "Gửi lệnh hủy"}</button></footer></form></section></div> : null}
  </main>;
}
