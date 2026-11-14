"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { createIdempotencyKey, PublicApiError, type OrderTrackingResponse } from "@/lib/api-contract";
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
  const [tracking, setTracking] = useState<OrderTrackingResponse | null>(null);
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
      setTracking(null);
      if (detail.cancellation.canViewTracking) {
        setTrackingState("loading");
        try {
          setTracking(await orderService.tracking(orderId, scenario));
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
          ? "Đơn đã thay đổi hoặc không còn đủ điều kiện hủy. Dữ liệu đang được tải lại."
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
    <section className="order-detail-hero"><div><span className="eyebrow">Chi tiết đơn hàng</span><h1>Đơn #{order.orderNumber}</h1><p>Đặt hàng · {new Date(order.placedAt).toLocaleString("vi-VN", { dateStyle: "long", timeStyle: "short" })}</p></div><div className="order-detail-actions"><button type="button" className="secondary-button" onClick={() => window.print()}>In thông tin</button>{canManageOrder && order.cancellation.canCancel ? <button type="button" className="order-danger-button" onClick={() => { setDialogOpen(true); setCancelError(""); }}>Hủy đơn</button> : null}</div></section>
    {notice ? <div className="order-notice" role="status">{notice}</div> : null}
    {order.operationalHold || order.hasActiveCase ? <div className="checkout-inline-error" role="status"><strong>Đơn đang cần lưu ý</strong><p>{order.statusDescription}</p></div> : null}

    <section className="order-status-grid">
      <article><span>▤</span><div><small>Trạng thái đơn</small><strong>{order.statusLabel}</strong><p>{order.statusDescription}</p></div></article>
      <article><span>◎</span><div><small>Trạng thái thanh toán</small><strong>{order.payment.label}</strong><p>{order.payment.method}{order.payment.expiresAt ? ` · hết hạn ${new Date(order.payment.expiresAt).toLocaleString("vi-VN")}` : ""}</p></div></article>
      <article><span>▱</span><div><small>Trạng thái kho</small><strong>{order.stock.label}</strong></div></article>
    </section>

    <section className="order-journey"><header><div><span className="eyebrow">Hành trình giao hàng</span><h2>Theo dõi đơn hàng</h2></div>{tracking?.shipment_id ? <b>Mã vận đơn: {tracking.shipment_id}</b> : null}</header>
      {!order.cancellation.canViewTracking ? <p className="cart-checkout-note" role="status">Đơn này chưa có thông tin theo dõi hành trình.</p> : trackingState === "loading" ? <p className="cart-checkout-note" role="status">Đang đọc tracking…</p> : trackingState === "unavailable" ? <p className="cart-checkout-note" role="alert">Theo dõi hành trình hiện chưa khả dụng; chi tiết đơn vẫn hiển thị bình thường.</p> : tracking ? <div className="order-tracking-minimal"><dl><div><dt>Trạng thái đơn</dt><dd>{order.statusLabel}</dd></div><div><dt>Mã vận đơn</dt><dd>{tracking.shipment_id ?? "Chưa có"}</dd></div></dl><p>{tracking.shipment_id ? "Đơn đã có mã vận đơn; thông tin hãng vận chuyển và hành trình sẽ hiển thị khi có." : "Chưa có dữ liệu hành trình cho đơn này."}</p></div> : null}
    </section>

    <div className="order-detail-layout"><div className="order-detail-main">
      <section className="order-detail-card"><header><span>◈</span><div><h2>Danh mục thức quà</h2><p>{order.snapshotAvailable ? "Danh sách sản phẩm tại thời điểm đặt hàng." : "Đơn này chưa có danh sách sản phẩm chi tiết."}</p></div></header>{order.lines.length ? <div className="order-lines">{order.lines.map((line) => <article key={line.lineId}><span aria-hidden="true">◇</span><div><strong>{line.productName}</strong><small>{line.skuLabel}</small><em>{lineAmount(line.unitPriceVnd)} × {line.quantity}</em></div><b>{lineAmount(line.lineSubtotalVnd)}</b></article>)}</div> : <div className="orders-state"><p>Đơn này chưa có dòng sản phẩm.</p></div>}</section>
      <section className="order-detail-card"><header><span>⌖</span><div><h2>Thông tin nhận hàng</h2><p>Thông tin người nhận tại thời điểm đặt hàng.</p></div></header>{order.recipient ? <dl className="order-recipient"><div><dt>Người nhận</dt><dd>{order.recipient.fullName}</dd></div><div><dt>Điện thoại</dt><dd>{order.recipient.phoneDisplay}</dd></div><div><dt>Địa chỉ</dt><dd>{order.recipient.addressDisplay}</dd></div></dl> : <div className="orders-state"><p>Đơn này chưa có thông tin địa chỉ giao hàng.</p></div>}</section>
    </div><aside>
      <section className="order-total-card"><span className="eyebrow">Chi tiết số tiền</span><dl><div><dt>Tạm tính</dt><dd>{VND.format(order.pricing.subtotalVnd)}</dd></div><div><dt>Phí vận chuyển</dt><dd>{VND.format(order.pricing.shippingFeeVnd)}</dd></div><div className="grand"><dt>Tổng thanh toán</dt><dd>{VND.format(order.pricing.totalVnd)}</dd></div></dl></section>
      <section className="order-policy-card"><h2>Trạng thái hủy</h2><p>{order.cancellation.message}</p>{order.cancellation.status !== "NONE" ? <small>Trạng thái hủy: {order.cancellation.statusLabel} · Bồi hoàn: {order.cancellation.compensationLabel}</small> : null}<Link href="/support/request">Cần hỗ trợ về đơn này?</Link>{canManageOrder && ["DELIVERED", "COMPLETED"].includes(order.status) ? <Link className="order-after-sales-link" href={`/orders/${encodeURIComponent(order.orderId)}/after-sales/request`}>Tạo yêu cầu hậu mãi / đổi trả</Link> : null}</section>
    </aside></div>

    {dialogOpen ? <div className="order-dialog-backdrop" role="presentation" onMouseDown={() => !cancelling && setDialogOpen(false)}><section className="order-dialog" role="dialog" aria-modal="true" aria-labelledby="cancel-order-title" onMouseDown={(event) => event.stopPropagation()}><span className="order-dialog-icon">!</span><h2 id="cancel-order-title">Gửi yêu cầu hủy đơn</h2><p>{order.cancellation.message}</p><form onSubmit={cancelOrder}><label><span>Lý do hủy <b>*</b></span><select value={reasonCode} onChange={(event) => { setReasonCode(event.target.value as CancelReasonCode); setCancelError(""); }}><option value="">Chọn lý do</option>{order.cancellation.allowedReasons.map((reason) => <option key={reason.code} value={reason.code}>{reason.label}</option>)}</select></label><label><span>Ghi chú</span><textarea value={note} maxLength={220} onChange={(event) => setNote(event.target.value)} placeholder="Thông tin bổ sung cho yêu cầu hủy…" /></label><div className="order-cancel-warning"><strong>Yêu cầu đang chờ xử lý chưa có nghĩa đơn đã hủy.</strong><p>Trang sẽ tự tải lại trạng thái hủy và hoàn tiền sau khi gửi.</p></div>{cancelError ? <div className="status-notice error" role="alert">{cancelError}</div> : null}<footer><button type="button" className="secondary-button" disabled={cancelling} onClick={() => setDialogOpen(false)}>Giữ đơn</button><button className="order-danger-button" disabled={cancelling}>{cancelling ? "Đang gửi…" : "Gửi lệnh hủy"}</button></footer></form></section></div> : null}
  </main>;
}
