"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { AccountShell } from "@/components/account/account-shell";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import type { OrderListResponse, OrderStatus } from "@/lib/orders/types";
import { OrderApiError } from "@/lib/orders/types";
import { orderService } from "@/services/order-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const FILTERS: Array<{ value: OrderStatus | "ALL"; label: string }> = [
  { value: "ALL", label: "Tất cả" },
  { value: "PENDING_PAYMENT", label: "Chờ thanh toán" },
  { value: "PROCESSING", label: "Đang chuẩn bị" },
  { value: "SHIPPED", label: "Đang giao" },
  { value: "DELIVERED", label: "Đã giao" },
  { value: "CANCELLED", label: "Đã hủy" }
];

function statusTone(status: OrderStatus) {
  if (["DELIVERED", "COMPLETED"].includes(status)) return "success";
  if (["CANCELLED", "EXPIRED", "DELIVERY_FAILED"].includes(status)) return "muted";
  if (status === "PENDING_PAYMENT") return "warning";
  return "active";
}

export function OrdersFlow() {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [data, setData] = useState<OrderListResponse | null>(null);
  const [status, setStatus] = useState<OrderStatus | "ALL">("ALL");
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [year, setYear] = useState(new Date().getFullYear());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try { setData(await orderService.list({ q: query, status, year, pageSize: 10, mockScenario: scenario })); }
    catch (cause) { setError(cause instanceof OrderApiError ? cause.message : "Không thể tải lịch sử Order."); }
    finally { setLoading(false); }
  }, [query, status, year, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  function submitSearch(event: FormEvent) {
    event.preventDefault();
    setQuery(queryInput.trim());
  }

  return <><SiteHeader /><AccountShell active="orders" customerName={data?.customer.displayName} customerEmail={data?.customer.email}>
    <section className="orders-hero"><div><span className="eyebrow">Nhật ký thưởng thức</span><h1>Đơn hàng của tôi</h1><p>Theo dõi từng Order và hành trình giao nhận bằng dữ liệu snapshot tại thời điểm đặt.</p></div><dl><div><dt>Tổng kết quả</dt><dd>{data?.totalItems ?? "—"}</dd></div><div><dt>Đang xử lý</dt><dd>{data?.items.filter((item) => ["PENDING_PAYMENT", "PAID", "CONFIRMED", "PROCESSING", "PACKED", "SHIPPED"].includes(item.status)).length ?? "—"}</dd></div></dl></section>

    <section className="orders-toolbar">
      <form onSubmit={submitSearch}><label className="sr-only" htmlFor="order-search">Tìm Order</label><input id="order-search" value={queryInput} onChange={(event) => setQueryInput(event.target.value)} placeholder="Tìm theo mã Order hoặc tên sản phẩm…" /><button type="submit">Tìm kiếm</button></form>
      <label><span>Năm</span><select value={year} onChange={(event) => setYear(Number(event.target.value))}>{[2026, 2025, 2024].map((value) => <option key={value}>{value}</option>)}</select></label>
      <div className="orders-tabs" role="tablist" aria-label="Lọc trạng thái">{FILTERS.map((item) => <button key={item.value} type="button" role="tab" aria-selected={status === item.value} className={status === item.value ? "active" : ""} onClick={() => setStatus(item.value)}>{item.label}</button>)}</div>
    </section>

    {loading ? <section className="orders-loading" aria-busy="true"><div /><div /><div /></section> : error ? <section className="orders-state error" role="alert"><span>!</span><h2>Chưa thể mở lịch sử Order</h2><p>{error}</p><button className="secondary-button" onClick={() => void load()}>Thử lại</button></section> : !data?.items.length ? <section className="orders-state"><span>▤</span><h2>Chưa có Order phù hợp</h2><p>Thử thay đổi bộ lọc hoặc khám phá sản phẩm Cố Đô cho lần đặt quà đầu tiên.</p><Link className="primary-link" href="/products">Khám phá sản phẩm</Link></section> : <section className="orders-list" aria-live="polite">
      {data.items.map((order) => <article className="order-card" key={order.orderId}>
        <header><div><strong>#{order.orderNumber}</strong><span>{new Date(order.placedAt).toLocaleString("vi-VN", { dateStyle: "medium", timeStyle: "short" })}</span></div><b className={`order-status ${statusTone(order.status)}`}>{order.statusLabel}</b></header>
        <div className="order-card-body"><div className="order-preview-stack">{order.previewItems.map((item, index) => <div className="order-preview" key={`${order.orderId}-${item.name}`}><span aria-hidden="true">{index === 0 ? "◈" : "◇"}</span><div><strong>{item.name}</strong><small>Số lượng: {item.quantity}</small></div></div>)}</div><dl><div><dt>Số món</dt><dd>{order.itemCount}</dd></div><div><dt>Payment</dt><dd>{order.paymentStatus}</dd></div><div><dt>Tổng thanh toán</dt><dd>{VND.format(order.totalVnd)}</dd></div></dl></div>
        <footer><p>{order.cancelPolicyMessage}</p><Link className="primary-link" href={`/orders/${encodeURIComponent(order.orderId)}`}>Xem chi tiết &amp; tracking</Link></footer>
      </article>)}
    </section>}
  </AccountShell><SiteFooter /></>;
}
