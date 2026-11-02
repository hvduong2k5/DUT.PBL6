"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import type { ReturnEligibility, ReturnReason } from "@/lib/returns/types";
import { ReturnApiError } from "@/lib/returns/types";
import { returnService } from "@/services/return-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export function AfterSalesRequestFlow({ orderId }: { orderId: string }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [eligibility, setEligibility] = useState<ReturnEligibility | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [selected, setSelected] = useState<Record<string, number>>({});
  const [reason, setReason] = useState<ReturnReason | "">("");
  const [details, setDetails] = useState("");
  const [evidenceUrls, setEvidenceUrls] = useState("");
  const [bankCode, setBankCode] = useState("");
  const [accountNumber, setAccountNumber] = useState("");
  const [accountHolder, setAccountHolder] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true); setLoadError("");
    try { setEligibility(await returnService.eligibility(orderId, scenario)); }
    catch (cause) { setLoadError(cause instanceof ReturnApiError ? cause.message : "Không thể tải thông tin đơn hàng."); }
    finally { setLoading(false); }
  }, [orderId, scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  const chosenLine = useMemo(() => eligibility?.lines.find((line) => (selected[line.lineId] ?? 0) > 0) ?? null, [eligibility, selected]);
  const evidenceMediaUrls = useMemo(() => evidenceUrls.split(/\r?\n/u).map((value) => value.trim()).filter(Boolean), [evidenceUrls]);

  function toggleLine(lineId: string, max: number) {
    setSelected((current) => current[lineId] ? {} : { [lineId]: Math.min(1, max) });
    setSubmitError("");
  }

  function changeQuantity(lineId: string, max: number, delta: number) {
    setSelected((current) => ({ [lineId]: Math.max(1, Math.min(max, (current[lineId] ?? 1) + delta)) }));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitError("");
    if (!chosenLine) { setSubmitError("Hãy chọn một sản phẩm cần hỗ trợ."); return; }
    if (!reason) { setSubmitError("Hãy chọn lý do yêu cầu."); return; }
    if (details.trim().length < 10) { setSubmitError("Hãy mô tả tình trạng ít nhất 10 ký tự."); return; }
    if (evidenceMediaUrls.length < 1 || evidenceMediaUrls.length > 5) { setSubmitError("Hãy cung cấp từ 1 đến 5 URL ảnh hoặc video bằng chứng."); return; }
    if (!bankCode.trim() || !/^\d{6,30}$/u.test(accountNumber.trim()) || accountHolder.trim().length < 2) { setSubmitError("Hãy nhập đầy đủ thông tin tài khoản nhận hoàn tiền."); return; }
    setSubmitting(true);
    try {
      const result = await returnService.create({
        orderId,
        lines: [{ lineId: chosenLine.lineId, quantity: selected[chosenLine.lineId] }],
        reasonCode: reason,
        details: details.trim(),
        evidenceMediaUrls,
        refundBankCode: bankCode.trim(),
        refundAccountNumber: accountNumber.trim(),
        refundAccountHolder: accountHolder.trim(),
        idempotencyKey: `return-case-${globalThis.crypto.randomUUID()}`
      }, scenario);
      router.push(result.casePath);
    } catch (cause) {
      setSubmitError(cause instanceof ReturnApiError ? cause.message : "Không thể tạo yêu cầu đổi trả lúc này.");
    } finally { setSubmitting(false); }
  }

  if (loading) return <main className="return-page"><div className="return-loading" aria-busy="true"><div /><div /></div></main>;
  if (!eligibility) return <main className="return-page"><section className="return-state error"><span>!</span><h1>Chưa thể mở yêu cầu đổi trả</h1><p>{loadError}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href={`/orders/${encodeURIComponent(orderId)}`}>Về chi tiết đơn hàng</Link></div></section></main>;

  return <main className="return-page">
    <nav className="detail-breadcrumbs" aria-label="Đường dẫn"><Link href="/">Trang chủ</Link><span>›</span><Link href={`/orders/${encodeURIComponent(orderId)}`}>Đơn #{eligibility.orderNumber}</Link><span>›</span><strong>Tạo yêu cầu đổi trả</strong></nav>
    <section className="return-hero"><div><span className="eyebrow">Hỗ trợ sau mua hàng</span><h1>Yêu cầu đổi trả &amp; hoàn tiền</h1><p>Mỗi ticket áp dụng cho một SKU. Điều kiện đổi trả và số tiền hoàn được hệ thống xác nhận sau khi gửi.</p></div><dl><div><dt>Đơn hàng</dt><dd>#{eligibility.orderNumber}</dd></div><div><dt>Điều kiện</dt><dd>Xác nhận khi gửi</dd></div></dl></section>
    <div className="return-layout"><form className="return-form" onSubmit={submit}>
      <section className="return-card"><header><b>1</b><div><h2>Chọn một sản phẩm</h2><p>{eligibility.policyMessage}</p></div></header><div className="return-lines">{eligibility.lines.map((line) => <article className={selected[line.lineId] ? "selected" : ""} key={line.lineId}><input type="radio" name="return-line" aria-label={`Chọn ${line.productName}`} checked={Boolean(selected[line.lineId])} onChange={() => toggleLine(line.lineId, line.maxReturnQty)} /><span className="return-product-symbol">◇</span><div><strong>{line.productName}</strong><small>{line.skuLabel} · Đã mua {line.purchasedQty}</small><em>Có thể yêu cầu tối đa {line.maxReturnQty}</em></div><aside><b>{VND.format(line.unitPriceVnd)}</b>{selected[line.lineId] ? <div><button type="button" onClick={() => changeQuantity(line.lineId, line.maxReturnQty, -1)}>−</button><span>{selected[line.lineId]}</span><button type="button" onClick={() => changeQuantity(line.lineId, line.maxReturnQty, 1)}>+</button></div> : null}</aside></article>)}</div></section>

      <section className="return-card"><header><b>2</b><div><h2>Lý do &amp; mô tả tình trạng</h2><p>Thông tin này được gửi trực tiếp cùng ticket đổi trả.</p></div></header><div className="return-fields"><label><span>Phân loại sự cố <b>*</b></span><select value={reason} onChange={(event) => setReason(event.target.value as ReturnReason)}><option value="">Chọn lý do</option>{eligibility.reasonOptions.map((option) => <option key={option.code} value={option.code}>{option.label}</option>)}</select></label><label><span>Mô tả chi tiết <b>*</b></span><textarea value={details} maxLength={1000} onChange={(event) => setDetails(event.target.value)} placeholder="Nêu thời điểm nhận hàng và mức độ ảnh hưởng…" /><small>{details.length}/1000 ký tự</small></label></div></section>

      <section className="return-card"><header><b>3</b><div><h2>URL ảnh hoặc video bằng chứng</h2><p>API customer yêu cầu ít nhất một URL media đã được tải lên; nhập tối đa 5 URL, mỗi dòng một URL.</p></div></header><div className="return-fields"><label><span>URL bằng chứng <b>*</b></span><textarea value={evidenceUrls} onChange={(event) => setEvidenceUrls(event.target.value)} placeholder={"https://cdn.example.vn/returns/anh-01.jpg\nhttps://cdn.example.vn/returns/video-01.mp4"} /><small>{evidenceMediaUrls.length}/5 URL</small></label></div></section>

      <section className="return-card"><header><b>4</b><div><h2>Tài khoản nhận hoàn tiền</h2><p>Thông tin bắt buộc theo API; việc nhập tài khoản không có nghĩa ticket đã được duyệt hoàn tiền.</p></div></header><div className="return-fields"><label><span>Mã ngân hàng <b>*</b></span><input value={bankCode} maxLength={30} onChange={(event) => setBankCode(event.target.value.toUpperCase())} placeholder="Ví dụ: MBBANK" /></label><label><span>Số tài khoản <b>*</b></span><input inputMode="numeric" value={accountNumber} maxLength={30} onChange={(event) => setAccountNumber(event.target.value.replace(/\D/gu, ""))} placeholder="Nhập số tài khoản" /></label><label><span>Tên chủ tài khoản <b>*</b></span><input value={accountHolder} maxLength={100} onChange={(event) => setAccountHolder(event.target.value.toUpperCase())} placeholder="NGUYEN VAN AN" /></label></div></section>

      {submitError ? <div className="status-notice error" role="alert">{submitError}</div> : null}
      <div className="return-actions"><Link className="secondary-link" href={`/orders/${encodeURIComponent(orderId)}`}>Hủy &amp; về đơn hàng</Link><button className="primary-button" disabled={submitting}>{submitting ? <><span className="spinner" /> Đang gửi…</> : "Gửi yêu cầu đổi trả"}</button></div>
    </form>

      <aside className="return-summary"><section><span className="eyebrow">Tóm tắt</span><h2>Phạm vi yêu cầu</h2><dl><div><dt>Sản phẩm</dt><dd>{chosenLine ? 1 : 0}</dd></div><div><dt>Số lượng</dt><dd>{chosenLine ? selected[chosenLine.lineId] : 0}</dd></div><div><dt>Giá trị tham chiếu</dt><dd>{VND.format(chosenLine ? chosenLine.unitPriceVnd * selected[chosenLine.lineId] : 0)}</dd></div><div><dt>Bằng chứng</dt><dd>{evidenceMediaUrls.length}/5</dd></div></dl><p>Giá trị tham chiếu không phải số tiền hoàn được duyệt.</p></section><section><span className="eyebrow">Thông tin giao hàng</span><h2>{eligibility.pickupSnapshot.recipientName}</h2><p>{eligibility.pickupSnapshot.phoneDisplay}</p><p>{eligibility.pickupSnapshot.addressDisplay}</p></section><section className="return-help"><h2>Cần hỗ trợ?</h2><p>Hotline Tri Kỷ 24/7</p><a href="tel:190068483">1900 68 Hue</a></section></aside>
    </div>
  </main>;
}
