"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { AccountShell } from "@/components/account/account-shell";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { PublicApiError } from "@/lib/api-contract";
import type { OrderListResponse, OrderStatus } from "@/lib/orders/types";
import { orderService } from "@/services/order-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const PAGE_LIMIT = 20;
type StatusFilter = OrderStatus | "ALL";
const FILTERS: Array<{ value: StatusFilter; label: string }> = [
  { value: "ALL", label: "Tất cả" },
  { value: "PENDING_PAYMENT", label: "Chờ thanh toán" },
  { value: "PAYMENT_FINALIZING", label: "Đang xác minh" },
  { value: "PAID", label: "Đã thanh toán" },
  { value: "CONFIRMED_COD", label: "Đã xác nhận COD" },
  { value: "PROCESSING", label: "Đang chuẩn bị" },
  { value: "PACKED", label: "Đã đóng gói" },
  { value: "SHIPPED", label: "Đang giao" },
  { value: "DELIVERED", label: "Đã giao" },
  { value: "COMPLETED", label: "Hoàn tất" },
  { value: "DELIVERY_FAILED", label: "Giao thất bại" },
  { value: "CANCELLED_BY_USER", label: "Khách đã hủy" },
  { value: "CANCELLED_TIMEOUT", label: "Hủy do hết hạn" },
  { value: "CANCELLED_BY_ADMIN", label: "Hệ thống đã hủy" }
];

function statusTone(status: OrderStatus) {
  if (["DELIVERED", "COMPLETED"].includes(status)) return "success";
  if (status.startsWith("CANCELLED_") || status === "DELIVERY_FAILED") return "muted";
  if (status === "PENDING_PAYMENT") return "warning";
  return "active";
}

function appendUnique(current: OrderListResponse, next: OrderListResponse): OrderListResponse {
  const items = [...current.items];
  const known = new Set(items.map((item) => item.orderId));
  for (const item of next.items) {
    if (!known.has(item.orderId)) {
      known.add(item.orderId);
      items.push(item);
    }
  }
  return { items, nextCursor: next.nextCursor, resultCount: items.length };
}

export function OrdersFlow() {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [data, setData] = useState<OrderListResponse | null>(null);
  const [status, setStatus] = useState<StatusFilter>("ALL");
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let active = true;
    void Promise.resolve().then(async () => {
      if (!active) return;
      setLoading(true);
      setData(null);
      setError("");
      try {
        const payload = await orderService.list({ limit: PAGE_LIMIT, status, mockScenario: scenario });
        if (active) setData(payload);
      } catch (cause) {
        if (active) setError(cause instanceof PublicApiError ? cause.message : "Không thể tải lịch sử Order.");
      } finally {
        if (active) setLoading(false);
      }
    });
    return () => { active = false; };
  }, [refreshKey, scenario, status]);

  async function loadMore() {
    if (!data?.nextCursor || loadingMore) return;
    setLoadingMore(true);
    setError("");
    try {
      const next = await orderService.list({ cursor: data.nextCursor, limit: PAGE_LIMIT, status, mockScenario: scenario });
      setData((current) => current ? appendUnique(current, next) : next);
    } catch (cause) {
      setError(cause instanceof PublicApiError ? cause.message : "Không thể tải trang Order tiếp theo.");
    } finally {
      setLoadingMore(false);
    }
  }

  return <><SiteHeader /><AccountShell active="orders">
    <section className="orders-hero"><div><span className="eyebrow">Nhật ký thưởng thức</span><h1>Đơn hàng của tôi</h1><p>Danh sách được đọc theo cursor, mới nhất trước; trạng thái là trạng thái Order canonical.</p></div><dl><div><dt>Đã tải</dt><dd>{data?.items.length ?? "—"}</dd></div><div><dt>Đang xử lý</dt><dd>{data?.items.filter((item) => ["PENDING_PAYMENT", "PAYMENT_FINALIZING", "PAID", "CONFIRMED_COD", "PROCESSING", "PACKED", "SHIPPED"].includes(item.status)).length ?? "—"}</dd></div></dl></section>

    <section className="orders-toolbar">
      <div className="orders-tabs" role="tablist" aria-label="Lọc chính xác theo trạng thái Order">{FILTERS.map((item) => <button key={item.value} type="button" role="tab" aria-selected={status === item.value} className={status === item.value ? "active" : ""} onClick={() => setStatus(item.value)}>{item.label}</button>)}</div>
      <button type="button" className="secondary-button" disabled={loading || loadingMore} onClick={() => setRefreshKey((value) => value + 1)}>Làm mới từ đầu</button>
    </section>

    {loading ? <section className="orders-loading" aria-busy="true"><div /><div /><div /></section> : error && !data ? <section className="orders-state error" role="alert"><span>!</span><h2>Chưa thể mở lịch sử Order</h2><p>{error}</p><button className="secondary-button" onClick={() => setRefreshKey((value) => value + 1)}>Thử lại</button></section> : !data?.items.length ? <section className="orders-state"><span>▤</span><h2>Chưa có Order phù hợp</h2><p>Bộ lọc trạng thái hiện tại chưa có Order. Tìm kiếm theo mã hoặc sản phẩm chưa thuộc public contract.</p><Link className="primary-link" href="/products">Khám phá sản phẩm</Link></section> : <section className="orders-list" aria-live="polite">
      {data.items.map((order) => <article className="order-card" key={order.orderId}>
        <header><div><strong>#{order.orderNumber}</strong><span>{new Date(order.placedAt).toLocaleString("vi-VN", { dateStyle: "medium", timeStyle: "short" })}</span></div><b className={`order-status ${statusTone(order.status)}`}>{order.statusLabel}</b></header>
        <div className="order-card-body"><div className="order-preview-stack">{order.previewItems.length ? order.previewItems.map((item, index) => <div className="order-preview" key={`${order.orderId}-${item.name}`}><span aria-hidden="true">{index === 0 ? "◈" : "◇"}</span><div><strong>{item.name}</strong><small>Số lượng: {item.quantity}</small></div></div>) : <div className="order-preview"><span aria-hidden="true">◇</span><div><strong>Snapshot sản phẩm chưa được trả về</strong><small>Thông tin Order tổng quan vẫn là dữ liệu canonical.</small></div></div>}</div><dl><div><dt>Số món</dt><dd>{order.itemCount ?? "—"}</dd></div><div><dt>Payment</dt><dd>{order.paymentStatusLabel}</dd></div><div><dt>Tổng thanh toán</dt><dd>{VND.format(order.totalVnd)}</dd></div></dl></div>
        <footer><p>{order.cancelPolicyMessage}</p><Link className="primary-link" href={`/orders/${encodeURIComponent(order.orderId)}`}>Xem chi tiết &amp; tracking</Link></footer>
      </article>)}
      {error ? <div className="status-notice error" role="alert">{error}</div> : null}
      {data.nextCursor ? <button type="button" className="secondary-button" disabled={loadingMore} onClick={() => void loadMore()}>{loadingMore ? "Đang tải…" : "Tải thêm Order"}</button> : <p className="cart-checkout-note" role="status">Đã tải hết danh sách cho bộ lọc hiện tại.</p>}
    </section>}
  </AccountShell><SiteFooter /></>;
}
