"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { ReturnCaseDetail, ReturnCaseMessage, ReturnResolution } from "@/lib/returns/types";
import { ReturnApiError } from "@/lib/returns/types";
import { returnService } from "@/services/return-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const RESOLUTION_LABELS: Record<ReturnResolution, string> = { REPLACEMENT: "Đổi sản phẩm", ORIGINAL_PAYMENT_REFUND: "Hoàn phương thức gốc", LOYALTY_CREDIT: "Điểm Di Sản" };

export function AfterSalesCaseFlow({ caseId }: { caseId: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [data, setData] = useState<ReturnCaseDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try { setData(await returnService.detail(caseId, scenario)); }
    catch (cause) { setError(cause instanceof ReturnApiError ? cause.message : "Không thể tải hồ sơ hậu mãi."); }
    finally { setLoading(false); }
  }, [caseId, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  async function sendSupplement(event: FormEvent) {
    event.preventDefault();
    const value = message.trim();
    if (value.length < 2) { setSendError("Hãy nhập nội dung cần bổ sung."); return; }
    setSending(true); setSendError("");
    try {
      const result = await returnService.supplement(caseId, value, `return-supplement-${globalThis.crypto.randomUUID()}`, scenario);
      setData((current) => current ? { ...current, communications: [...current.communications, { ...result, message: value } as ReturnCaseMessage] } : current);
      setMessage("");
    } catch (cause) { setSendError(cause instanceof ReturnApiError ? cause.message : "Không thể gửi nội dung bổ sung."); }
    finally { setSending(false); }
  }

  if (loading) return <main className="case-page"><div className="case-loading" aria-busy="true"><div /><div /><div /></div></main>;
  if (!data) return <main className="case-page"><section className="return-state error"><span>!</span><h1>Chưa thể mở hồ sơ hậu mãi</h1><p>{error || "Hồ sơ không tồn tại hoặc bạn không có quyền xem."}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href="/account/orders">Đơn hàng của tôi</Link></div></section></main>;

  const success = ["APPROVED", "REFUNDED", "CLOSED"].includes(data.status);
  const danger = data.status === "REJECTED";
  return <main className="case-page">
    <nav className="detail-breadcrumbs"><Link href="/">Trang chủ</Link><span>›</span><Link href={`/orders/${encodeURIComponent(data.orderId)}`}>Order #{data.orderNumber}</Link><span>›</span><strong>#{data.caseNumber}</strong></nav>
    <section className={`case-hero ${success ? "success" : ""} ${danger ? "danger" : ""}`}><div><span className="eyebrow">Trung tâm hậu mãi • Customer-safe projection</span><h1>Hồ sơ hậu mãi #{data.caseNumber}</h1><p>Tạo lúc {new Date(data.createdAt).toLocaleString("vi-VN", { dateStyle: "medium", timeStyle: "short" })} · Order #{data.orderNumber}</p></div><div><span>{success ? "✓" : danger ? "×" : "◷"}</span><strong>{data.statusLabel}</strong><small>{data.statusDescription}</small></div></section>

    <section className="case-steps"><header><span className="eyebrow">Tiến trình xử lý</span><p>Quyết định, pickup và Refund là các mốc riêng biệt.</p></header><ol>{data.steps.map((step, index) => <li className={step.state.toLocaleLowerCase()} key={step.code}><span>{step.state === "COMPLETED" ? "✓" : index + 1}</span><div><small>Bước {index + 1}</small><strong>{step.label}</strong><time>{step.occurredAt ? new Date(step.occurredAt).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" }) : "Chưa diễn ra"}</time></div></li>)}</ol></section>

    <div className="case-layout"><div className="case-main">
      {data.decision ? <section className="case-card decision-card"><header><span>ÔM</span><div><h2>Quyết định hậu mãi</h2><p>{new Date(data.decision.decidedAt).toLocaleString("vi-VN", { dateStyle: "long", timeStyle: "short" })}</p></div></header><blockquote>“{data.decision.summary}”</blockquote><strong>{data.decision.label}</strong></section> : <section className="case-card decision-card pending"><header><span>◷</span><div><h2>Đang chờ quyết định</h2><p>Hồ sơ chưa có quyết định cuối cùng.</p></div></header><p>Ban Hậu mãi sẽ cập nhật projection sau khi actor có quyền hoàn tất thẩm định.</p></section>}

      <section className="case-card"><header><span>▧</span><div><h2>Thức quà &amp; phạm vi xử lý</h2><p>Phần yêu cầu và phần được duyệt được hiển thị riêng.</p></div></header><div className="case-items">{data.items.map((item) => <article key={item.lineId}><span>◇</span><div><strong>{item.productName}</strong><small>{item.skuLabel} · Số lượng {item.quantity}</small><em>Yêu cầu: {RESOLUTION_LABELS[item.requestedResolution]}</em></div><aside><b>{VND.format(item.unitPriceVnd * item.quantity)}</b><small>{item.approvedResolution ? `Đã duyệt: ${RESOLUTION_LABELS[item.approvedResolution]}` : "Chưa duyệt / không duyệt"}</small></aside></article>)}</div></section>

      <section className="case-card"><header><span>▣</span><div><h2>Bằng chứng đã xác nhận</h2><p>Chỉ projection an toàn; không public storage key hay Packing Video gốc.</p></div></header><div className="case-evidence">{data.evidence.map((item) => <article className={item.previewKind.toLocaleLowerCase()} key={item.evidenceId}><span>{item.previewKind === "VIDEO" ? "▶" : item.previewKind === "WORKSHOP" ? "⌂" : "▧"}</span><div><strong>{item.label}</strong><small>{item.mediaType} · {item.state}</small></div></article>)}</div></section>

      <section className="case-card case-conversation"><header><span>☵</span><div><h2>Trao đổi &amp; bổ sung hồ sơ</h2><p>Projection hội thoại từ Customer Service boundary.</p></div></header><div className="case-messages">{data.communications.map((item) => <article className={item.senderRole.toLocaleLowerCase()} key={item.messageId}><div><strong>{item.senderDisplayName}</strong><time>{new Date(item.occurredAt).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" })}</time></div><p>{item.message}</p></article>)}</div><form onSubmit={sendSupplement}><label className="sr-only" htmlFor="case-supplement">Bổ sung hồ sơ</label><textarea id="case-supplement" value={message} maxLength={1000} onChange={(event) => setMessage(event.target.value)} placeholder="Bổ sung thông tin liên quan đến hồ sơ…" />{sendError ? <div className="status-notice error" role="alert">{sendError}</div> : null}<button className="primary-button" disabled={sending}>{sending ? "Đang gửi…" : "Gửi bổ sung"}</button></form></section>
    </div>

      <aside className="case-aside">{data.pickup ? <section><span className="eyebrow">Điều phối thu hồi / giao bù</span><h2>{data.pickup.carrierLabel}</h2><dl><div><dt>Trạng thái</dt><dd>{data.pickup.statusLabel}</dd></div><div><dt>Khung giờ</dt><dd>{data.pickup.windowLabel}</dd></div><div><dt>Mã vận đơn</dt><dd>{data.pickup.trackingCode}</dd></div><div><dt>Địa chỉ</dt><dd>{data.pickup.addressDisplay}</dd></div></dl><Link className="primary-link full-width" href={`/orders/${encodeURIComponent(data.orderId)}`}>Theo dõi trong Order</Link></section> : <section><span className="eyebrow">Điều phối</span><h2>Chưa tạo pickup</h2><p>Việc phê duyệt Case không tự đồng nghĩa hàng đã được nhận hoặc Shipment đã hoàn tất.</p></section>}
      <section><span className="eyebrow">Refund boundary</span><h2>{data.refund.required ? "Có nghĩa vụ hoàn tiền" : "Không có Refund"}</h2><dl><div><dt>Phương thức</dt><dd>{data.refund.methodLabel ?? "—"}</dd></div><div><dt>Số tiền</dt><dd>{VND.format(data.refund.amountVnd)}</dd></div><div><dt>Trạng thái</dt><dd>{data.refund.status ?? "—"}</dd></div></dl>{data.refund.status === "PENDING" ? <p className="case-warning">Đang chờ không có nghĩa là đã hoàn tiền.</p> : null}</section>
      <section className="case-support"><h2>Cần hỗ trợ?</h2><p>Hotline Tri Kỷ 24/7</p><a href="tel:190068483">1900 68 Hue</a><Link href="/account/orders">Trở về đơn hàng</Link></section></aside>
    </div>
  </main>;
}
