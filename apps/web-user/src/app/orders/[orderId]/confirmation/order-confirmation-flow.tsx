"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { OrderDetail } from "@/lib/orders/types";
import { OrderApiError } from "@/lib/orders/types";
import { orderService } from "@/services/order-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export function OrderConfirmationFlow({ orderId }: { orderId: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [order, setOrder] = useState<OrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try { setOrder(await orderService.detail(orderId, scenario)); }
    catch (cause) { setError(cause instanceof OrderApiError ? cause.message : "Không thể mở xác nhận Order."); }
    finally { setLoading(false); }
  }, [orderId, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  if (loading) return <main className="order-confirmation-page"><div className="order-confirmation-loading" aria-busy="true" /></main>;
  if (!order) return <main className="order-confirmation-page"><section className="orders-state error"><span>!</span><h1>Chưa thể mở xác nhận</h1><p>{error}</p><button className="secondary-button" onClick={() => void load()}>Thử lại</button></section></main>;

  return <main className="order-confirmation-page">
    <ol className="payment-steps" aria-label="Tiến trình mua hàng"><li className="done"><span>✓</span><div><strong>Giỏ hàng</strong><small>Hoàn tất</small></div></li><li className="done"><span>✓</span><div><strong>Checkout &amp; Payment</strong><small>Đã tạo Order</small></div></li><li className="active"><span>3</span><div><strong>Xác nhận &amp; tracking</strong><small>Đang hoạt động</small></div></li></ol>
    <section className="order-confirmation-hero"><span className="order-confirmation-seal">✓</span><p className="eyebrow">Order đã được tiếp nhận</p><h1>{order.status === "PENDING_PAYMENT" ? "Order đang chờ thanh toán" : "Đặt hàng thành công — cảm ơn bạn"}</h1><p>{order.statusDescription} Bạn có thể theo dõi các mốc công khai mà không cần dựa vào dữ liệu Catalog hiện tại.</p><div><small>Mã Order</small><strong>#{order.orderNumber}</strong><button type="button" onClick={() => void navigator.clipboard.writeText(order.orderNumber)}>Sao chép mã</button></div></section>
    <section className="order-status-grid confirmation"><article><span>▤</span><div><small>Order</small><strong>{order.statusLabel}</strong><p>{order.statusDescription}</p></div></article><article><span>◎</span><div><small>Payment</small><strong>{order.payment.label}</strong><p>{order.payment.method === "COD" ? "Thu tiền theo xác nhận giao hàng hợp lệ." : "Chỉ nguồn tin cậy mới xác minh Payment."}</p></div></article><article><span>▱</span><div><small>Vận chuyển</small><strong>{order.shipping.label}</strong><p>{order.shipping.estimatedDelivery ?? "Chưa có dự kiến."}</p></div></article></section>
    <div className="order-confirmation-layout"><section className="order-confirmation-card"><header><div><p className="eyebrow">Snapshot đã chốt</p><h2>Thức quà trong Order</h2></div><b>{order.lines.reduce((total, line) => total + line.quantity, 0)} sản phẩm</b></header><div>{order.lines.map((line) => <article key={line.lineId}><span>◇</span><div><strong>{line.productName}</strong><small>{line.skuLabel} • Số lượng {line.quantity}</small></div><b>{VND.format(line.lineSubtotalVnd)}</b></article>)}</div><dl><div><dt>Tạm tính</dt><dd>{VND.format(order.pricing.subtotalVnd)}</dd></div><div><dt>Phí vận chuyển</dt><dd>{VND.format(order.pricing.shippingFeeVnd)}</dd></div><div><dt>Tổng thanh toán</dt><dd>{VND.format(order.pricing.totalVnd)}</dd></div></dl></section><aside className="order-confirmation-card recipient"><p className="eyebrow">Thông tin nhận hàng</p><h2>{order.recipient.fullName}</h2><dl><div><dt>Liên hệ</dt><dd>{order.recipient.phoneDisplay}</dd></div><div><dt>Địa chỉ</dt><dd>{order.recipient.addressDisplay}</dd></div><div><dt>Phương thức</dt><dd>{order.shipping.methodName}</dd></div></dl><div className="order-confirmation-note">Thông tin là snapshot của Order và không sửa trực tiếp từ trang này.</div></aside></div>
    <section className="order-confirmation-actions"><Link className="primary-link" href={`/orders/${encodeURIComponent(order.orderId)}`}>Theo dõi chi tiết Order</Link><Link className="secondary-link" href="/account/orders">Mở lịch sử Order</Link><Link href="/products">Tiếp tục khám phá</Link></section>
  </main>;
}
