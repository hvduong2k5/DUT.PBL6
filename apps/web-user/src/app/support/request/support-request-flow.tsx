"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { ChangeEvent, FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import type { SupportAttachmentMetadata, SupportContext, SupportPriority, SupportSubjectCode } from "@/lib/support/types";
import { SupportApiError } from "@/lib/support/types";
import { supportService } from "@/services/support-service";

const ICONS: Record<SupportPriority, string> = { NORMAL: "◷", IMPORTANT: "⌛", URGENT: "◇" };

export function SupportRequestFlow() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [context, setContext] = useState<SupportContext | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [subject, setSubject] = useState<SupportSubjectCode | "">("");
  const [title, setTitle] = useState("");
  const [message, setMessage] = useState("");
  const [priority, setPriority] = useState<SupportPriority>("IMPORTANT");
  const [linkedOrderId, setLinkedOrderId] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [attachments, setAttachments] = useState<SupportAttachmentMetadata[]>([]);
  const [fileError, setFileError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [existingTicketId, setExistingTicketId] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true); setLoadError("");
    try {
      const next = await supportService.context(scenario);
      setContext(next);
      if (next.subjects.length) setSubject(next.subjects[0].code);
    } catch (cause) { setLoadError(cause instanceof SupportApiError ? cause.message : "Không thể tải thông tin hỗ trợ."); }
    finally { setLoading(false); }
  }, [scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);
  const selectedOrder = useMemo(() => context?.linkableOrders.find((order) => order.orderId === linkedOrderId), [context, linkedOrderId]);

  function chooseFiles(event: ChangeEvent<HTMLInputElement>) {
    if (!context) return;
    const incoming = [...(event.target.files ?? [])];
    const next: SupportAttachmentMetadata[] = [];
    const messages: string[] = [];
    for (const file of incoming) {
      if (!context.attachmentPolicy.allowedMediaTypes.includes(file.type as SupportAttachmentMetadata["mediaType"])) { messages.push(`${file.name}: loại tệp không hỗ trợ.`); continue; }
      if (file.size > context.attachmentPolicy.maxBytesPerFile) { messages.push(`${file.name}: vượt 10 MiB.`); continue; }
      next.push({ clientReference: `attachment-${globalThis.crypto.randomUUID()}`, fileName: file.name, mediaType: file.type as SupportAttachmentMetadata["mediaType"], sizeBytes: file.size });
    }
    setAttachments((current) => [...current, ...next].slice(0, context.attachmentPolicy.maxFiles));
    if (attachments.length + next.length > context.attachmentPolicy.maxFiles) messages.push(`Chỉ giữ tối đa ${context.attachmentPolicy.maxFiles} tệp.`);
    setFileError(messages.join(" "));
    event.target.value = "";
  }

  async function submit(event: FormEvent) {
    event.preventDefault(); setSubmitError(""); setExistingTicketId("");
    if (!context || !subject) { setSubmitError("Hãy chọn chủ đề cần hỗ trợ."); return; }
    if (title.trim().length < 5) { setSubmitError("Tiêu đề cần ít nhất 5 ký tự."); return; }
    if (message.trim().length < 10) { setSubmitError("Nội dung cần ít nhất 10 ký tự."); return; }
    if (context.customerMode === "GUEST" && (displayName.trim().length < 2 || !email.includes("@"))) { setSubmitError("Guest cần nhập họ tên và email hợp lệ."); return; }
    setSubmitting(true);
    try {
      const result = await supportService.create({ subjectCode: subject, title: title.trim(), message: message.trim(), requestedPriority: priority, linkedOrderId: linkedOrderId || undefined, contact: context.customerMode === "GUEST" ? { displayName: displayName.trim(), email: email.trim(), phone: phone.trim() || undefined } : undefined, attachments, idempotencyKey: `support-ticket-${globalThis.crypto.randomUUID()}` }, scenario);
      router.push(result.ticketPath);
    } catch (cause) {
      if (cause instanceof SupportApiError) { setSubmitError(cause.message); setExistingTicketId(cause.existingTicketId ?? ""); }
      else setSubmitError("Không thể gửi yêu cầu hỗ trợ lúc này.");
    } finally { setSubmitting(false); }
  }

  if (loading) return <main className="support-page"><div className="support-loading" aria-busy="true"><div /><div /></div></main>;
  if (!context) return <main className="support-page"><section className="return-state error"><span>!</span><h1>Chưa thể mở trung tâm hỗ trợ</h1><p>{loadError}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href="/">Về trang chủ</Link></div></section></main>;

  return <main className="support-page">
    <nav className="detail-breadcrumbs"><Link href="/">Trang chủ</Link><span>›</span><strong>Gửi yêu cầu hỗ trợ</strong></nav>
    <section className="support-hero"><span className="eyebrow">Trung tâm phục vụ Tri Kỷ &amp; thưởng trà Ô Mạ</span><h1>Gửi Yêu Cầu Hỗ Trợ &amp; Trao Đổi Cùng Nghệ Nhân</h1><p>Mọi thắc mắc về nghệ thuật thưởng thức bánh mứt, phối ngự trà, tư vấn lễ vật biếu tặng hay tiến trình giao nhận, quý khách vui lòng gửi thông tin để Ban Chăm Sóc Tri Kỷ phản hồi chu toàn nhất.</p><div><b>⌛ Cam kết phản hồi trong vòng 2 giờ làm việc</b><span>● Nghệ nhân đang trực tiếp nhận</span></div></section>

    <div className="support-layout"><form className="support-form" onSubmit={submit}>
      <section className="support-card"><header><b>1</b><h2>Thông Tin Tri Kỷ Gửi Yêu Cầu</h2></header>{context.customerMode === "REGISTERED" && context.customer ? <div className="support-customer"><span>{context.customer.displayName.split(" ").slice(-2).map((part) => part[0]).join("")}</span><div><strong>{context.customer.displayName}</strong><small>{context.customer.phoneDisplay} · {context.customer.emailDisplay}</small></div>{context.customer.membershipLabel ? <em>{context.customer.membershipLabel}</em> : null}</div> : <div className="support-guest"><label><span>Họ và tên *</span><input value={displayName} onChange={(event) => setDisplayName(event.target.value)} autoComplete="name" /></label><label><span>Email *</span><input type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" /></label><label><span>Điện thoại</span><input value={phone} onChange={(event) => setPhone(event.target.value)} autoComplete="tel" /></label></div>}</section>

      <section className="support-card"><header><b>2</b><h2>Chủ Đề Cần Trao Đổi Cùng Nghệ Nhân</h2></header><label className="support-field"><span>Chọn phạm vi chủ đề hỗ trợ *</span><select value={subject} onChange={(event) => setSubject(event.target.value as SupportSubjectCode)}>{context.subjects.map((item) => <option value={item.code} key={item.code}>{item.label}</option>)}</select><small>{context.subjects.find((item) => item.code === subject)?.description}</small></label></section>

      <section className="support-card"><header><b>3</b><h2>Liên Kết Đơn Hàng / Thức Quà Liên Quan</h2></header>{context.linkableOrders.length ? <><div className="support-order-toggle"><button className={!linkedOrderId ? "active" : ""} type="button" onClick={() => setLinkedOrderId("")}>Không liên kết</button>{context.linkableOrders.map((order) => <button className={linkedOrderId === order.orderId ? "active" : ""} type="button" onClick={() => setLinkedOrderId(order.orderId)} key={order.orderId}>#{order.orderNumber}</button>)}</div>{selectedOrder ? <div className="support-order"><span>▣</span><div><strong>Đơn hàng #{selectedOrder.orderNumber}</strong><small>{selectedOrder.summary}</small></div><em>{selectedOrder.statusLabel}</em></div> : <p className="support-hint">Ticket không liên kết vẫn được tiếp nhận. Chỉ chọn Order thực sự liên quan.</p>}</> : <p className="support-hint">Không có Order đã được xác minh để liên kết. Guest không được tự liên kết chỉ bằng mã dễ đoán.</p>}</section>

      <section className="support-card"><header><b>4</b><h2>Mức Độ Ưu Tiên Mong Muốn</h2></header><div className="support-priorities">{context.priorities.map((item) => <label className={priority === item.code ? "selected" : ""} key={item.code}><input type="radio" name="priority" checked={priority === item.code} onChange={() => setPriority(item.code)} /><span>{ICONS[item.code]}</span><strong>{item.label}</strong><small>{item.description}</small><em>{item.responsePromiseLabel}</em></label>)}</div><p className="support-hint">Đây là mức độ khách đề nghị; hệ thống và CSKH xác định priority/SLA chính thức.</p></section>

      <section className="support-card"><header><b>5</b><h2>Nội Dung Yêu Cầu &amp; Trao Đổi Chi Tiết</h2></header><label className="support-field"><span>Tóm tắt tiêu đề yêu cầu *</span><input value={title} maxLength={160} onChange={(event) => setTitle(event.target.value)} placeholder="Ví dụ: Tư vấn cách bảo quản bánh và thưởng trà" /><small>{title.length}/160 ký tự</small></label><label className="support-field"><span>Chi tiết câu hỏi hoặc việc cần hỗ trợ *</span><textarea value={message} maxLength={3000} onChange={(event) => setMessage(event.target.value)} placeholder="Mô tả rõ bối cảnh và nội dung cần hỗ trợ…" /><small>{message.length}/3000 ký tự</small></label></section>

      <section className="support-card"><header><b>6</b><h2>Hình Ảnh &amp; Tài Liệu Minh Họa</h2></header><label className="support-upload"><span>⇧</span><strong>Nhấp để tải lên hoặc kéo thả hình ảnh, tài liệu</strong><small>JPEG, PNG, WebP, PDF · tối đa 5 tệp · 10 MiB/tệp</small><input type="file" multiple accept="image/jpeg,image/png,image/webp,application/pdf" onChange={chooseFiles} /></label>{fileError ? <div className="status-notice error" role="alert">{fileError}</div> : null}<div className="support-files">{attachments.map((file) => <article key={file.clientReference}><span>{file.mediaType === "application/pdf" ? "PDF" : "▧"}</span><div><strong>{file.fileName}</strong><small>{(file.sizeBytes / 1024 / 1024).toFixed(1)} MiB · metadata local</small></div><button type="button" onClick={() => setAttachments((current) => current.filter((item) => item.clientReference !== file.clientReference))}>×</button></article>)}</div></section>

      {submitError ? <div className="status-notice error" role="alert">{submitError}{existingTicketId ? <> <Link href={`/support/tickets/${encodeURIComponent(existingTicketId)}`}>Mở Ticket hiện có</Link></> : null}</div> : null}
      <div className="support-actions"><button className="primary-button" disabled={submitting}>{submitting ? <><span className="spinner" /> Đang gửi…</> : "Gửi Yêu Cầu Đến Ban Tri Kỷ"}</button><Link className="secondary-link" href="/">Hủy bỏ &amp; quay lại</Link></div>
    </form>

      <aside className="support-aside"><section className="support-hotline"><span className="eyebrow">Kênh trực tiếp khẩn cấp</span><h2>1900 68 Hue</h2><p>Dành cho việc tiệc gấp trong ngày. Hotline không thay thế Ticket và không tự thay đổi trạng thái nghiệp vụ.</p><a href="tel:190068483">Gọi ngay</a></section><section><span className="eyebrow">Cam kết phụng sự</span><h2>Tri Kỷ Cố Đô</h2><ul><li>Phản hồi theo service promise công khai.</li><li>Bảo mật nội dung và tệp theo quyền.</li><li>Mỗi Ticket có mã theo dõi duy nhất.</li></ul></section><section><span className="eyebrow">Hỏi nhanh đáp gọn</span><h2>Trước khi gửi</h2><details open><summary>Có cần liên kết Order?</summary><p>Không bắt buộc. Chỉ liên kết khi Order thực sự liên quan và thuộc quyền của bạn.</p></details><details><summary>Ảnh đã được tải lên thật chưa?</summary><p>Local mock chỉ mô phỏng metadata; production cần upload và scan riêng.</p></details></section></aside>
    </div>
  </main>;
}
