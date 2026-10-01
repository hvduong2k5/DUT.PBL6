"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { ReturnCaseDetail } from "@/lib/returns/types";
import { ReturnApiError } from "@/lib/returns/types";
import { returnService } from "@/services/return-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export function AfterSalesCaseFlow({ caseId }: { caseId: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [data, setData] = useState<ReturnCaseDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try { setData(await returnService.detail(caseId, scenario)); }
    catch (cause) { setError(cause instanceof ReturnApiError ? cause.message : "Không thể tải ticket đổi trả."); }
    finally { setLoading(false); }
  }, [caseId, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  if (loading) return <main className="case-page"><div className="case-loading" aria-busy="true"><div /><div /><div /></div></main>;
  if (!data) return <main className="case-page"><section className="return-state error"><span>!</span><h1>Chưa thể mở ticket đổi trả</h1><p>{error || "Ticket không tồn tại hoặc bạn không có quyền xem."}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href="/account/orders">Đơn hàng của tôi</Link></div></section></main>;

  const success = ["APPROVED", "REFUNDED", "CLOSED"].includes(data.status);
  const danger = data.status === "REJECTED";
  return <main className="case-page">
    <nav className="detail-breadcrumbs"><Link href="/">Trang chủ</Link><span>›</span><Link href={`/orders/${encodeURIComponent(data.orderId)}`}>Đơn #{data.orderNumber}</Link><span>›</span><strong>#{data.caseNumber}</strong></nav>
    <section className={`case-hero ${success ? "success" : ""} ${danger ? "danger" : ""}`}><div><span className="eyebrow">Trung tâm hậu mãi</span><h1>Ticket đổi trả #{data.caseNumber}</h1><p>Tạo lúc {new Date(data.createdAt).toLocaleString("vi-VN", { dateStyle: "medium", timeStyle: "short" })} · Đơn #{data.orderNumber}</p></div><div><span>{success ? "✓" : danger ? "×" : "◷"}</span><strong>{data.statusLabel}</strong><small>{data.statusDescription}</small></div></section>

    <section className="case-steps"><header><span className="eyebrow">Tiến trình xử lý</span><p>Trạng thái mới nhất được đồng bộ từ API customer.</p></header><ol>{data.steps.map((step, index) => <li className={step.state.toLocaleLowerCase()} key={step.code}><span>{step.state === "COMPLETED" ? "✓" : index + 1}</span><div><small>Bước {index + 1}</small><strong>{step.label}</strong><time>{step.occurredAt ? new Date(step.occurredAt).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" }) : "Chưa diễn ra"}</time></div></li>)}</ol></section>

    <div className="case-layout"><div className="case-main">
      <section className="case-card decision-card"><header><span>☵</span><div><h2>Lý do yêu cầu</h2><p>Nội dung ghi nhận trên ticket</p></div></header><blockquote>“{data.reason}”</blockquote></section>
      <section className="case-card decision-card pending"><header><span>i</span><div><h2>Thông tin ticket hiện có</h2><p>API customer chưa trả lại SKU, URL bằng chứng, lịch sử trao đổi hoặc thông tin điều phối trong màn tra cứu.</p></div></header><p>Các dữ liệu chưa được API cung cấp sẽ không được giao diện tự suy đoán hoặc hiển thị giả.</p></section>
    </div>

      <aside className="case-aside"><section><span className="eyebrow">Hoàn tiền</span><h2>{data.refund.required ? "Khoản hoàn dự kiến" : "Chưa có khoản hoàn"}</h2><dl><div><dt>Phương thức</dt><dd>{data.refund.methodLabel ?? "—"}</dd></div><div><dt>Số tiền</dt><dd>{data.refund.required ? VND.format(data.refund.amountVnd) : "—"}</dd></div><div><dt>Trạng thái</dt><dd>{data.refund.status ?? "—"}</dd></div></dl>{data.refund.status === "PENDING" ? <p className="case-warning">Đang chờ không có nghĩa là đã hoàn tiền.</p> : null}</section>
      <section className="case-support"><h2>Cần bổ sung thông tin?</h2><p>API customer hiện chưa có thao tác bổ sung ticket. Vui lòng liên hệ hỗ trợ.</p><a href="tel:190068483">1900 68 Hue</a><Link href="/account/orders">Trở về đơn hàng</Link></section></aside>
    </div>
  </main>;
}
