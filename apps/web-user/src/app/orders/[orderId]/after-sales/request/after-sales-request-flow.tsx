"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { ChangeEvent, FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import type { EvidenceMetadata, ReturnEligibility, ReturnReason, ReturnResolution } from "@/lib/returns/types";
import { ReturnApiError } from "@/lib/returns/types";
import { returnService } from "@/services/return-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const ALLOWED_TYPES = new Set(["image/jpeg", "image/png", "image/webp", "video/mp4"]);
const MAX_FILE_SIZE = 15 * 1024 * 1024;

export function AfterSalesRequestFlow({ orderId }: { orderId: string }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [eligibility, setEligibility] = useState<ReturnEligibility | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [selected, setSelected] = useState<Record<string, number>>({});
  const [resolution, setResolution] = useState<ReturnResolution | "">("");
  const [reason, setReason] = useState<ReturnReason | "">("");
  const [details, setDetails] = useState("");
  const [pickupNote, setPickupNote] = useState("");
  const [evidence, setEvidence] = useState<EvidenceMetadata[]>([]);
  const [fileError, setFileError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [existingCaseId, setExistingCaseId] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true); setLoadError("");
    try { setEligibility(await returnService.eligibility(orderId, scenario)); }
    catch (cause) { setLoadError(cause instanceof ReturnApiError ? cause.message : "Không thể kiểm tra điều kiện hậu mãi."); }
    finally { setLoading(false); }
  }, [orderId, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  const chosenLines = useMemo(() => eligibility?.lines.filter((line) => (selected[line.lineId] ?? 0) > 0) ?? [], [eligibility, selected]);
  const estimatedValue = chosenLines.reduce((sum, line) => sum + line.unitPriceVnd * (selected[line.lineId] ?? 0), 0);

  function toggleLine(lineId: string, max: number) {
    setSelected((current) => current[lineId] ? Object.fromEntries(Object.entries(current).filter(([key]) => key !== lineId)) : { ...current, [lineId]: Math.min(1, max) });
    setSubmitError("");
  }

  function changeQuantity(lineId: string, max: number, delta: number) {
    setSelected((current) => ({ ...current, [lineId]: Math.max(1, Math.min(max, (current[lineId] ?? 1) + delta)) }));
  }

  function chooseFiles(event: ChangeEvent<HTMLInputElement>) {
    const incoming = [...(event.target.files ?? [])];
    const next: EvidenceMetadata[] = [];
    const messages: string[] = [];
    for (const file of incoming) {
      if (!ALLOWED_TYPES.has(file.type)) { messages.push(`${file.name}: loại tệp không hỗ trợ.`); continue; }
      if (file.size > MAX_FILE_SIZE) { messages.push(`${file.name}: vượt 15 MiB.`); continue; }
      next.push({ clientReference: `evidence-${globalThis.crypto.randomUUID()}`, fileName: file.name, mediaType: file.type as EvidenceMetadata["mediaType"], sizeBytes: file.size });
    }
    setEvidence((current) => [...current, ...next].slice(0, 5));
    if (evidence.length + next.length > 5) messages.push("Chỉ giữ tối đa 5 tệp.");
    setFileError(messages.join(" "));
    event.target.value = "";
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitError(""); setExistingCaseId("");
    if (!chosenLines.length) { setSubmitError("Hãy chọn ít nhất một sản phẩm cần hỗ trợ."); return; }
    if (!resolution) { setSubmitError("Hãy chọn phương án giải quyết mong muốn."); return; }
    if (!reason) { setSubmitError("Hãy chọn lý do yêu cầu."); return; }
    if (details.trim().length < 10) { setSubmitError("Hãy mô tả tình trạng ít nhất 10 ký tự."); return; }
    setSubmitting(true);
    try {
      const result = await returnService.create({ orderId, lines: chosenLines.map((line) => ({ lineId: line.lineId, quantity: selected[line.lineId] })), reasonCode: reason, preferredResolution: resolution, details: details.trim(), pickupNote: pickupNote.trim() || undefined, evidence, idempotencyKey: `return-case-${globalThis.crypto.randomUUID()}` }, scenario);
      router.push(result.casePath);
    } catch (cause) {
      if (cause instanceof ReturnApiError) { setSubmitError(cause.message); setExistingCaseId(cause.existingCaseId ?? ""); }
      else setSubmitError("Không thể tạo hồ sơ hậu mãi lúc này.");
    } finally { setSubmitting(false); }
  }

  if (loading) return <main className="return-page"><div className="return-loading" aria-busy="true"><div /><div /></div></main>;
  if (!eligibility) return <main className="return-page"><section className="return-state error"><span>!</span><h1>Chưa thể mở yêu cầu hậu mãi</h1><p>{loadError}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href={`/orders/${encodeURIComponent(orderId)}`}>Về chi tiết Order</Link></div></section></main>;
  if (!eligibility.eligible) return <main className="return-page"><nav className="detail-breadcrumbs"><Link href={`/orders/${encodeURIComponent(orderId)}`}>Order #{eligibility.orderNumber}</Link><span>›</span><strong>Hậu mãi</strong></nav><section className="return-state"><span>⌛</span><h1>Order chưa đủ điều kiện tự tạo hồ sơ</h1><p>{eligibility.policyMessage}</p><div><Link className="secondary-link" href={`/orders/${encodeURIComponent(orderId)}`}>Về Order</Link><a className="primary-link" href="tel:190068483">Liên hệ hỗ trợ</a></div></section></main>;

  return <main className="return-page">
    <nav className="detail-breadcrumbs" aria-label="Đường dẫn"><Link href="/">Trang chủ</Link><span>›</span><Link href={`/orders/${encodeURIComponent(orderId)}`}>Order #{eligibility.orderNumber}</Link><span>›</span><strong>Tạo yêu cầu hậu mãi</strong></nav>
    <section className="return-hero"><div><span className="eyebrow">Giữ trọn niềm tin sau mỗi thức quà</span><h1>Yêu cầu hậu mãi &amp; đổi trả</h1><p>Chọn đúng sản phẩm và mô tả tình trạng. Phương án bạn chọn là mong muốn ban đầu, chưa phải quyết định hoàn tiền/đổi mới.</p></div><dl><div><dt>Order</dt><dd>#{eligibility.orderNumber}</dd></div><div><dt>Hạn chính sách</dt><dd>{new Date(eligibility.policyWindowEndsAt).toLocaleDateString("vi-VN")}</dd></div></dl></section>
    <div className="return-layout"><form className="return-form" onSubmit={submit}>
      <section className="return-card"><header><b>1</b><div><h2>Chọn thức quà cần hỗ trợ</h2><p>{eligibility.policyMessage}</p></div></header><div className="return-lines">{eligibility.lines.map((line) => <article className={`${selected[line.lineId] ? "selected" : ""} ${!line.eligible ? "disabled" : ""}`} key={line.lineId}><input type="checkbox" aria-label={`Chọn ${line.productName}`} disabled={!line.eligible} checked={Boolean(selected[line.lineId])} onChange={() => toggleLine(line.lineId, line.maxReturnQty)} /><span className="return-product-symbol">◇</span><div><strong>{line.productName}</strong><small>{line.skuLabel} · Đã mua {line.purchasedQty}</small><em>{line.eligible ? `Có thể yêu cầu tối đa ${line.maxReturnQty}` : "Đã có hồ sơ hoặc không còn eligible"}</em></div><aside><b>{VND.format(line.unitPriceVnd)}</b>{selected[line.lineId] ? <div><button type="button" onClick={() => changeQuantity(line.lineId, line.maxReturnQty, -1)}>−</button><span>{selected[line.lineId]}</span><button type="button" onClick={() => changeQuantity(line.lineId, line.maxReturnQty, 1)}>+</button></div> : null}</aside></article>)}</div></section>

      <section className="return-card"><header><b>2</b><div><h2>Phương án giải quyết mong muốn</h2><p>Quyết định cuối cùng phụ thuộc chính sách và thẩm định có quyền.</p></div></header><div className="resolution-grid">{eligibility.resolutionOptions.map((option) => <label className={resolution === option.code ? "selected" : ""} key={option.code}><input type="radio" name="resolution" value={option.code} checked={resolution === option.code} onChange={() => setResolution(option.code)} /><span>{option.code === "REPLACEMENT" ? "↻" : option.code === "ORIGINAL_PAYMENT_REFUND" ? "₫" : "✦"}</span><strong>{option.label}</strong><small>{option.description}</small></label>)}</div></section>

      <section className="return-card"><header><b>3</b><div><h2>Lý do &amp; mô tả tình trạng</h2><p>Cung cấp thông tin có thể đối chiếu, không gửi dữ liệu tài chính nhạy cảm.</p></div></header><div className="return-fields"><label><span>Phân loại sự cố <b>*</b></span><select value={reason} onChange={(event) => setReason(event.target.value as ReturnReason)}><option value="">Chọn lý do</option>{eligibility.reasonOptions.map((option) => <option key={option.code} value={option.code}>{option.label}</option>)}</select></label><label><span>Mô tả chi tiết <b>*</b></span><textarea value={details} maxLength={1000} onChange={(event) => setDetails(event.target.value)} placeholder="Nêu thời điểm nhận/khui kiện và mức độ ảnh hưởng…" /><small>{details.length}/1000 ký tự</small></label></div></section>

      <section className="return-card"><header><b>4</b><div><h2>Bằng chứng ảnh &amp; video</h2><p>JPEG, PNG, WebP hoặc MP4 · tối đa 5 tệp · 15 MiB/tệp.</p></div></header><label className="evidence-picker"><span>⇧</span><strong>Chọn ảnh hoặc video từ thiết bị</strong><small>Local mock chỉ lưu metadata để duyệt UI; chưa upload binary.</small><input type="file" multiple accept="image/jpeg,image/png,image/webp,video/mp4" onChange={chooseFiles} /></label>{fileError ? <div className="status-notice error" role="alert">{fileError}</div> : null}<div className="evidence-files">{evidence.map((file) => <article key={file.clientReference}><span>{file.mediaType.startsWith("video/") ? "▶" : "▧"}</span><div><strong>{file.fileName}</strong><small>{(file.sizeBytes / 1024 / 1024).toFixed(1)} MiB · chờ upload production</small></div><button type="button" aria-label={`Bỏ ${file.fileName}`} onClick={() => setEvidence((current) => current.filter((item) => item.clientReference !== file.clientReference))}>×</button></article>)}</div></section>

      <section className="return-card"><header><b>5</b><div><h2>Địa điểm tiếp nhận thu hồi</h2><p>Dùng snapshot giao hàng; backend sẽ xác nhận lại trước khi điều phối.</p></div></header><dl className="return-pickup"><div><dt>Người nhận</dt><dd>{eligibility.pickupSnapshot.recipientName}</dd></div><div><dt>Điện thoại</dt><dd>{eligibility.pickupSnapshot.phoneDisplay}</dd></div><div className="wide"><dt>Địa chỉ</dt><dd>{eligibility.pickupSnapshot.addressDisplay}</dd></div></dl><label className="pickup-note"><span>Thời gian thuận tiện / ghi chú</span><input value={pickupNote} maxLength={300} onChange={(event) => setPickupNote(event.target.value)} placeholder="Ví dụ: thu hồi sau 14:00…" /></label></section>

      {submitError ? <div className="status-notice error" role="alert">{submitError}{existingCaseId ? <> <Link href={`/after-sales/${encodeURIComponent(existingCaseId)}`}>Mở hồ sơ hiện có</Link></> : null}</div> : null}
      <div className="return-actions"><Link className="secondary-link" href={`/orders/${encodeURIComponent(orderId)}`}>Hủy &amp; về Order</Link><button className="primary-button" disabled={submitting}>{submitting ? <><span className="spinner" /> Đang gửi hồ sơ…</> : "Gửi yêu cầu & nhận mã hồ sơ"}</button></div>
    </form>

      <aside className="return-summary"><section><span className="eyebrow">Tóm tắt dự kiến</span><h2>Phạm vi yêu cầu</h2><dl><div><dt>Sản phẩm đã chọn</dt><dd>{chosenLines.length}</dd></div><div><dt>Tổng số lượng</dt><dd>{Object.values(selected).reduce((sum, value) => sum + value, 0)}</dd></div><div><dt>Giá trị tham chiếu</dt><dd>{VND.format(estimatedValue)}</dd></div><div><dt>Bằng chứng</dt><dd>{evidence.length}/5</dd></div></dl><p>Giá trị tham chiếu không phải số tiền hoàn được duyệt.</p></section><section><span className="eyebrow">Bảo an Ô Mạ</span><h2>Quy trình minh bạch</h2><ul><li>Eligibility được kiểm tra lại khi gửi.</li><li>Case không tự tạo Refund.</li><li>Bằng chứng chỉ dùng khi đã lưu và scan hợp lệ.</li><li>Refund hoàn tất cần xác nhận tin cậy từ Payment.</li></ul></section><section className="return-help"><h2>Cần hỗ trợ gấp?</h2><p>Hotline Tri Kỷ 24/7</p><a href="tel:190068483">1900 68 Hue</a></section></aside>
    </div>
  </main>;
}
