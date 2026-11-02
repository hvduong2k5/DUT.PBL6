"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { SupportMessage, SupportTicketDetail } from "@/lib/support/types";
import { SupportApiError } from "@/lib/support/types";
import { supportService } from "@/services/support-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export function SupportTicketFlow({ ticketId }: { ticketId: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [data, setData] = useState<SupportTicketDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try { setData(await supportService.detail(ticketId, scenario)); }
    catch (cause) { setError(cause instanceof SupportApiError ? cause.message : "Không thể tải Ticket hỗ trợ."); }
    finally { setLoading(false); }
  }, [ticketId, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  async function sendMessage(event: FormEvent) {
    event.preventDefault(); const value = message.trim();
    if (value.length < 10) { setSendError("Nội dung phản hồi cần ít nhất 10 ký tự."); return; }
    setSending(true); setSendError("");
    try {
      const result = await supportService.message(ticketId, value, [], `support-message-${globalThis.crypto.randomUUID()}`, scenario);
      setData((current) => current ? { ...current, messages: [...current.messages, { ...result, message: value } as SupportMessage], updatedAt: result.occurredAt } : current);
      setMessage("");
    } catch (cause) { setSendError(cause instanceof SupportApiError ? cause.message : "Không thể gửi phản hồi."); }
    finally { setSending(false); }
  }

  if (loading) return <main className="support-page"><div className="ticket-loading" aria-busy="true"><div /><div /><div /></div></main>;
  if (!data) return <main className="support-page"><section className="return-state error"><span>!</span><h1>Chưa thể mở Ticket</h1><p>{error || "Ticket không tồn tại hoặc bạn không có quyền xem."}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href="/support/request">Gửi yêu cầu mới</Link></div></section></main>;

  const resolved = data.status === "RESOLVED" || data.status === "CLOSED";
  return <main className="support-page">
    <nav className="detail-breadcrumbs"><Link href="/">Trang chủ</Link><span>›</span><Link href="/support/request">Chăm sóc khách hàng</Link><span>›</span><strong>#{data.ticketNumber}</strong></nav>
    <section className={`ticket-hero ${resolved ? "resolved" : ""}`}><div><span className="ticket-code">#{data.ticketNumber}</span><span className="ticket-status">{data.statusLabel}</span><h1>{data.title}</h1><p>{data.subjectLabel} · Ưu tiên mong muốn: {data.requestedPriorityLabel}{data.linkedOrder ? <> · <Link href={`/orders/${encodeURIComponent(data.linkedOrder.orderId)}`}>Order #{data.linkedOrder.orderNumber}</Link></> : null}</p></div><div><Link className="secondary-link" href="/support/request">Gửi yêu cầu khác</Link></div></section>

    <div className="ticket-layout"><div className="ticket-main"><section className="ticket-conversation">{data.messages.map((item, index) => <article className={item.senderRole.toLowerCase()} key={item.messageId}><header><span>{item.senderRole === "SUPPORT" ? "ÔM" : item.senderRole === "SYSTEM" ? "i" : "TK"}</span><div><strong>{item.senderDisplayName}</strong><small>{new Date(item.occurredAt).toLocaleString("vi-VN", { dateStyle: "medium", timeStyle: "short" })}</small></div></header><p>{item.message}</p>{item.attachments.length ? <div className="ticket-attachments">{item.attachments.map((file) => <span key={file.attachmentId}>▧ {file.label} · {file.state}</span>)}</div> : null}{index < data.messages.length - 1 ? <div className="ticket-separator">{Math.round((new Date(data.messages[index + 1].occurredAt).getTime() - new Date(item.occurredAt).getTime()) / 60000)} phút sau đó</div> : null}</article>)}</section>

      <section className="ticket-reply"><header><h2>Phản hồi thêm cho Ban Chăm Sóc</h2><small>{data.publicAssignee ? `Đang kết nối: ${data.publicAssignee.displayName}` : "Đội ngũ Tri Kỷ sẽ tiếp nhận"}</small></header>{data.canReply ? <form onSubmit={sendMessage}><label className="sr-only" htmlFor="support-reply">Nội dung phản hồi</label><textarea id="support-reply" value={message} maxLength={3000} onChange={(event) => setMessage(event.target.value)} placeholder="Nhập lời nhắn gửi thêm…" />{sendError ? <div className="status-notice error" role="alert">{sendError}</div> : null}<button className="primary-button" disabled={sending}>{sending ? "Đang gửi…" : "Gửi phản hồi"}</button></form> : <div className="status-notice">Ticket đã đóng trao đổi. Hãy tạo yêu cầu mới nếu cần hỗ trợ thêm.</div>}</section></div>

      <aside className="ticket-aside"><section><span className="eyebrow">Thông tin hồ sơ Ticket</span><dl><div><dt>Cam kết phản hồi</dt><dd>{data.servicePromise.label}</dd></div><div><dt>Tình trạng</dt><dd>{data.statusLabel}</dd></div><div><dt>Người tiếp nhận</dt><dd>{data.publicAssignee?.displayName ?? "Đang phân công"}</dd></div><div><dt>Kênh tiếp nhận</dt><dd>Cổng Web Tri Kỷ</dd></div></dl><p>{data.statusDescription}</p></section>{data.linkedOrder ? <section><span className="eyebrow">Thức quà liên quan</span><div className="ticket-products">{data.linkedOrder.products.map((product) => <article key={`${product.name}-${product.variant}`}><span>◇</span><div><strong>{product.name}</strong><small>{product.variant} · SL {product.quantity}</small></div><b>{VND.format(product.priceVnd * product.quantity)}</b></article>)}</div><Link className="secondary-link full-width" href={`/orders/${encodeURIComponent(data.linkedOrder.orderId)}`}>Xem chi tiết Order</Link></section> : null}<section className="ticket-rating"><span className="eyebrow">Đánh giá của Tri Kỷ</span><h2>{resolved ? "Trải nghiệm hỗ trợ thế nào?" : "Sau khi Ticket hoàn tất"}</h2><div aria-label="Năm sao">☆ ☆ ☆ ☆ ☆</div><p>CSAT chỉ là phần trình bày UI trong MVP, chưa lưu dữ liệu.</p></section><section className="support-hotline"><h2>Điện Đàm Hoàng Cung</h2><p>Cần hỗ trợ trực tiếp bàn tiệc?</p><a href="tel:190068483">1900 68 Hue</a></section></aside>
    </div>
  </main>;
}
