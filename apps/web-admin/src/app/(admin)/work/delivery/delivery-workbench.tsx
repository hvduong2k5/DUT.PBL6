"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import type { ShipmentDetail, ShipmentList, ShipmentSummary } from "@/lib/shipping/types";
import { AdminShippingApiError } from "@/lib/shipping/types";
import { adminShippingService } from "@/services/admin-shipping-service";

const DATE_TIME = new Intl.DateTimeFormat("vi-VN", { dateStyle: "short", timeStyle: "short", timeZone: "Asia/Ho_Chi_Minh" });
const MONEY = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

function Status({ code, label, prefix = "shipment" }: { code: string; label: string; prefix?: string }) {
  return <span className={`admin-status ${prefix}-${code.toLocaleLowerCase()}`}>{label}</span>;
}

function messageOf(error: unknown) {
  return error instanceof AdminShippingApiError ? error.message : "Không thể tải công việc giao hàng.";
}

export function DeliveryWorkbench() {
  const [data, setData] = useState<ShipmentList | null>(null);
  const [selectedId, setSelectedId] = useState("");
  const [detail, setDetail] = useState<ShipmentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState("");
  const [detailError, setDetailError] = useState("");

  const openShipment = useCallback(async (item: ShipmentSummary) => {
    setSelectedId(item.shipmentId); setDetailLoading(true); setDetailError("");
    try { setDetail(await adminShippingService.assignedShipment(item.shipmentId)); }
    catch (cause) { setDetail(null); setDetailError(messageOf(cause)); }
    finally { setDetailLoading(false); }
  }, []);

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const result = await adminShippingService.assignedShipments();
      setData(result);
      const first = result.items.find((item) => item.status === "OUT_FOR_DELIVERY") ?? result.items.find((item) => !["DELIVERED", "RETURNED", "CANCELLED"].includes(item.status)) ?? result.items[0];
      if (first) await openShipment(first); else { setSelectedId(""); setDetail(null); }
    } catch (cause) { setData(null); setError(messageOf(cause)); }
    finally { setLoading(false); }
  }, [openShipment]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  const metrics = useMemo(() => {
    const items = data?.items ?? [];
    return {
      active: items.filter((item) => !["DELIVERED", "RETURNED", "CANCELLED"].includes(item.status)).length,
      delivering: items.filter((item) => item.status === "OUT_FOR_DELIVERY").length,
      cod: items.filter((item) => item.codAmountVnd > 0 && item.status !== "DELIVERED").reduce((sum, item) => sum + item.codAmountVnd, 0),
      attention: items.filter((item) => item.exceptionState !== "NONE").length
    };
  }, [data]);

  if (loading) return <div className="admin-page"><section className="admin-page-skeleton"><div /><div /></section></div>;
  if (error) return <div className="admin-page"><section className="admin-error-state"><span>!</span><h1>Không thể mở công việc giao hàng</h1><p>{error}</p><button className="admin-primary-button" onClick={() => void load()}>Thử lại</button></section></div>;

  return <div className="admin-page delivery-workbench-page">
    <header className="admin-page-header"><div><nav>Admin <span>›</span> Công việc của tôi <span>›</span> Giao hàng</nav><h1>Công việc giao hàng</h1><p>Chỉ hiển thị Shipment đang được phân công cho nhân viên giao hàng hiện tại theo US-SHIP-03.</p></div><div className="admin-page-actions"><button className="admin-secondary-button" type="button" onClick={() => void load()}>↻ Làm mới</button></div></header>
    <section className="admin-summary-grid"><article><span>Việc đang hoạt động</span><strong>{metrics.active}</strong><small>Không gồm đã giao, hoàn hoặc hủy</small></article><article><span>Đang giao</span><strong>{metrics.delivering}</strong><small>Shipment ở chặng giao cuối</small></article><article><span>COD cần thu</span><strong>{MONEY.format(metrics.cod)}</strong><small>Chỉ là nghĩa vụ thu, chưa phải Payment đã xác nhận</small></article><article><span>Cần chú ý</span><strong>{metrics.attention}</strong><small>Shipment có cảnh báo hoặc đang bị chặn</small></article></section>
    <section className="packing-workbench-layout delivery-workbench-layout">
      <aside className="packing-work-queue delivery-work-queue"><header><div><h2>Chuyến được giao</h2><p>Dữ liệu được giới hạn theo phân công</p></div><b>{data?.items.length ?? 0}</b></header><div>{data?.items.map((item) => <button type="button" className={selectedId === item.shipmentId ? "active" : ""} onClick={() => void openShipment(item)} key={item.shipmentId}><span><strong>{item.shipmentNumber}</strong><small>{item.orderNumber} · {item.packageReference}</small></span><span><Status code={item.status} label={item.statusLabel} /><small>{item.estimatedDeliveryAt ? `Hạn ${DATE_TIME.format(new Date(item.estimatedDeliveryAt))}` : "Chưa có hạn"}</small></span>{item.exceptionState !== "NONE" ? <Status code={item.exceptionState} label={item.exceptionLabel} prefix="shipping-exception" /> : <small>{item.providerLabel}</small>}</button>)}</div>{!data?.items.length ? <p className="packing-work-empty">Bạn chưa có Shipment được phân công.</p> : null}</aside>
      <main className="packing-work-detail delivery-work-detail">
        {detailLoading ? <div className="admin-modal-loading">Đang tải thông tin chuyến giao…</div> : detailError ? <div className="admin-inline-error">{detailError}</div> : detail ? <>
          <header className="packing-work-detail-header"><div><small>{detail.orderNumber}</small><h2>{detail.shipmentNumber}</h2><p>{detail.providerLabel} · {detail.serviceLabel} · Revision {detail.revision}</p></div><div><Status code={detail.status} label={detail.statusLabel} /><Status code={detail.exceptionState} label={detail.exceptionLabel} prefix="shipping-exception" /></div></header>
          {detail.exceptionReason ? <section className="admin-shipping-exception"><strong>{detail.exceptionLabel}</strong><p>{detail.exceptionReason}</p></section> : null}
          <div className="delivery-work-columns"><section><header className="admin-section-heading"><div><h3>Người nhận</h3><p>Thông tin nhạy cảm chỉ trả về cho người đang được phân công.</p></div></header><dl><div><dt>Họ tên</dt><dd>{detail.recipient.displayName}</dd></div><div><dt>Điện thoại</dt><dd>{detail.recipient.phoneDisplay}</dd></div><div><dt>Địa chỉ</dt><dd>{detail.recipient.addressDisplay}</dd></div><div><dt>Dự kiến giao</dt><dd>{detail.estimatedDeliveryAt ? DATE_TIME.format(new Date(detail.estimatedDeliveryAt)) : "Chưa có"}</dd></div></dl></section><section><header className="admin-section-heading"><div><h3>Kiện hàng và COD</h3><p>Shipment, Order và nghĩa vụ COD là các vòng đời riêng.</p></div></header><dl><div><dt>Kiện</dt><dd>{detail.packageReference}</dd></div><div><dt>Khối lượng</dt><dd>{detail.package.weightGrams.toLocaleString("vi-VN")} g</dd></div><div><dt>Kích thước</dt><dd>{detail.package.lengthCm && detail.package.widthCm && detail.package.heightCm ? `${detail.package.lengthCm} × ${detail.package.widthCm} × ${detail.package.heightCm} cm` : "Chưa có"}</dd></div><div><dt>COD</dt><dd><strong>{MONEY.format(detail.codAmountVnd)}</strong> · {detail.codStatusLabel}</dd></div></dl></section></div>
          <section><header className="admin-section-heading"><div><h3>Hành trình Shipment</h3><p>Hiển thị trạng thái công khai và nguồn cập nhật; không cho sửa dữ liệu provider.</p></div></header><div className="delivery-work-timeline">{detail.trackingEvents.filter((event) => event.mappingResult === "APPLIED").map((event) => <article key={event.eventId}><span /><div><strong>{event.publicDescription}</strong><small>{DATE_TIME.format(new Date(event.eventAt))} · {event.sourceLabel}</small></div><Status code={event.mappedStatus ?? "UNMAPPED"} label={event.mappedStatusLabel} /></article>)}</div></section>
          <section className="delivery-work-policy-note"><div><small>Trạng thái thao tác</small><h3>Chưa mở cập nhật kết quả giao</h3><p>Epic 11 đã xác nhận nhu cầu cập nhật giao thành công/thất bại, nhưng danh mục lý do, bằng chứng giao, nguồn xác nhận COD và ma trận chuyển trạng thái vẫn chưa chốt. Workbench hiện chỉ cung cấp đúng dữ liệu được phân công, không tự suy diễn mutation.</p></div></section>
        </> : <div className="admin-empty"><h3>Chọn một Shipment</h3><p>Thông tin công việc sẽ hiển thị tại đây.</p></div>}
      </main>
    </section>
  </div>;
}
