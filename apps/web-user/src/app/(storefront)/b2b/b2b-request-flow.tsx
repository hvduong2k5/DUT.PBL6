"use client";

import Link from "next/link";
import { ChangeEvent, FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { B2BAttachmentMetadata, B2BContext, B2BPurpose, CreateB2BQuoteRequestResult } from "@/lib/b2b/types";
import { B2BApiError } from "@/lib/b2b/types";
import { b2bService } from "@/services/b2b-service";
import { resolveProductImageSource } from "@/components/product/product-image";

const PURPOSES: Array<{ code: B2BPurpose; label: string; description: string }> = [
  { code: "VIP_CUSTOMER_GIFT", label: "Tri ân khách hàng VIP & đối tác", description: "Quà trang trọng, tinh tế và có nhận diện riêng." },
  { code: "EMPLOYEE_GIFT", label: "Quà Tết nhân viên", description: "Đồng bộ số lượng lớn theo ngân sách doanh nghiệp." },
  { code: "EVENT_GIFT", label: "Hội nghị & sự kiện", description: "Quà tặng theo thời điểm và thông điệp chương trình." },
  { code: "PARTNER_GIFT", label: "Quà ngoại giao", description: "Thiết kế cao cấp dành cho đối tác chiến lược." }
];
const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export function B2BRequestFlow() {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [context, setContext] = useState<B2BContext | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [purpose, setPurpose] = useState<B2BPurpose>("VIP_CUSTOMER_GIFT");
  const [deliveryDate, setDeliveryDate] = useState("");
  const [provinceCode, setProvinceCode] = useState("");
  const [quantities, setQuantities] = useState<Record<string, number>>({});
  const [branding, setBranding] = useState({ engraveLogo: true, customSleeve: true, greetingCard: true, ribbon: false });
  const [attachments, setAttachments] = useState<B2BAttachmentMetadata[]>([]);
  const [notes, setNotes] = useState("");
  const [invoiceRequested, setInvoiceRequested] = useState(true);
  const [sampleRequested, setSampleRequested] = useState(true);
  const [fileError, setFileError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [created, setCreated] = useState<CreateB2BQuoteRequestResult | null>(null);

  const load = useCallback(async () => {
    setLoading(true); setLoadError("");
    try {
      const next = await b2bService.context(scenario);
      setContext(next);
      setProvinceCode(next.deliveryProvinces[0]?.provinceCode ?? "");
      setQuantities(Object.fromEntries(next.catalog.map((item, index) => [item.skuId, index === 0 ? item.minimumQuantity : 0])));
    } catch (cause) { setLoadError(cause instanceof B2BApiError ? cause.message : "Không thể tải không gian B2B."); }
    finally { setLoading(false); }
  }, [scenario]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);
  const selectedItems = useMemo(() => context?.catalog.flatMap((item) => quantities[item.skuId] > 0 ? [{ ...item, quantity: quantities[item.skuId] }] : []) ?? [], [context, quantities]);
  const totalQuantity = selectedItems.reduce((sum, item) => sum + item.quantity, 0);
  const merchandise = selectedItems.reduce((sum, item) => sum + item.quantity * item.unitPriceVnd, 0);
  const tier = context?.discountTiers.filter((item) => totalQuantity >= item.minimumQuantity && (!item.maximumQuantity || totalQuantity <= item.maximumQuantity)).at(-1);
  const estimate = merchandise * (1 - (tier?.discountPercent ?? 0) / 100);
  const minimumDate = useMemo(() => { const date = new Date(); date.setDate(date.getDate() + 7); return date.toISOString().slice(0, 10); }, []);

  function chooseFiles(event: ChangeEvent<HTMLInputElement>) {
    if (!context) return;
    const messages: string[] = [];
    const next = [...(event.target.files ?? [])].flatMap((file) => {
      if (!context.attachmentPolicy.allowedMediaTypes.includes(file.type as B2BAttachmentMetadata["mediaType"])) { messages.push(`${file.name}: định dạng không được hỗ trợ.`); return []; }
      if (file.size > context.attachmentPolicy.maxBytesPerFile) { messages.push(`${file.name}: vượt 25 MiB.`); return []; }
      return [{ clientReference: `b2b-file-${globalThis.crypto.randomUUID()}`, fileName: file.name, mediaType: file.type as B2BAttachmentMetadata["mediaType"], sizeBytes: file.size }];
    });
    setAttachments((current) => [...current, ...next].slice(0, context.attachmentPolicy.maxFiles));
    if (attachments.length + next.length > context.attachmentPolicy.maxFiles) messages.push(`Chỉ giữ tối đa ${context.attachmentPolicy.maxFiles} tệp.`);
    setFileError(messages.join(" "));
    event.target.value = "";
  }

  async function submit(event: FormEvent) {
    event.preventDefault(); setSubmitError(""); setCreated(null);
    if (!context) return;
    if (!deliveryDate || deliveryDate < minimumDate) { setSubmitError("Ngày giao dự kiến cần cách hiện tại ít nhất 7 ngày."); return; }
    if (!provinceCode) { setSubmitError("Hãy chọn Tỉnh/Thành phố giao hàng."); return; }
    if (!selectedItems.length || selectedItems.some((item) => item.quantity < item.minimumQuantity)) { setSubmitError("Mỗi sản phẩm được chọn phải đạt số lượng tối thiểu."); return; }
    const province = context.deliveryProvinces.find((item) => item.provinceCode === provinceCode);
    if (!province) return;
    setSubmitting(true);
    try {
      const result = await b2bService.createQuoteRequest({ organizationId: context.organization.organizationId, purpose, requestedDeliveryDate: deliveryDate, deliveryProvinceCode: province.provinceCode, deliveryProvinceName: province.provinceName, items: selectedItems.map((item) => ({ skuId: item.skuId, quantity: item.quantity })), branding: { ...branding, attachments }, notes: notes.trim() || undefined, invoiceRequested, sampleRequested, idempotencyKey: `b2b-request-${globalThis.crypto.randomUUID()}` }, scenario);
      setCreated(result);
      globalThis.scrollTo({ top: 0, behavior: "smooth" });
    } catch (cause) { setSubmitError(cause instanceof B2BApiError ? cause.message : "Không thể gửi yêu cầu báo giá lúc này."); }
    finally { setSubmitting(false); }
  }

  if (loading) return <main className="b2b-page"><div className="b2b-loading" aria-busy="true"><div /><div /></div></main>;
  if (!context) return <main className="b2b-page"><section className="return-state error"><span>!</span><h1>Chưa thể mở không gian B2B</h1><p>{loadError}</p><button className="primary-button" onClick={() => void load()}>Thử lại</button></section></main>;

  return <main className="b2b-page">
    <nav className="detail-breadcrumbs"><Link href="/">Trang chủ</Link><span>›</span><strong>Báo giá sỉ B2B</strong></nav>
    <section className="b2b-hero"><div><span className="eyebrow">Giải pháp quà biếu doanh nghiệp &amp; hội nghị ngoại giao</span><h1>Đặc Quyền Quà Biếu Cố Đô<br />Dành Cho Doanh Nghiệp</h1><p>Tinh hoa bánh mứt Cung Đình trong giải pháp quà tặng có định mức số lượng, nhận diện thương hiệu và điều khoản thương mại minh bạch.</p><div className="b2b-benefits"><span>▦ Tập đoàn &amp; đối tác</span><span>٪ Chiết khấu theo số lượng</span><span>◉ Sản phẩm OCOP</span></div></div><aside><b>Hộp quà thiết kế riêng</b><small>Hỗ trợ khắc logo và thiệp doanh nghiệp</small></aside></section>
    {created ? <section className="b2b-created" role="status"><span>✓</span><div><h2>Đã tiếp nhận yêu cầu {created.requestNumber}</h2><p>{created.message} Chuyên viên sẽ phản hồi trên tài khoản doanh nghiệp.</p></div><button type="button" onClick={() => setCreated(null)}>Tạo yêu cầu khác</button></section> : null}

    <div className="b2b-layout"><form className="b2b-form" onSubmit={submit}>
      <section className="b2b-card"><header><b>1</b><h2>Thông tin doanh nghiệp &amp; tổ chức</h2><em>{context.organization.verificationStatus === "VERIFIED" ? "Đã xác minh" : "Đang xác minh"}</em></header><div className="b2b-company-grid"><label><span>Tên doanh nghiệp</span><input readOnly value={context.organization.legalName} /></label><label><span>Mã số thuế</span><input readOnly value={context.organization.taxCode} /></label><label><span>Người đại diện</span><input readOnly value={`${context.organization.representativeName} · ${context.organization.representativeTitle}`} /></label><label><span>Email liên hệ</span><input readOnly value={context.organization.email} /></label><label className="wide"><span>Địa chỉ xuất hóa đơn</span><input readOnly value={context.organization.invoiceAddress} /></label></div></section>

      <section className="b2b-card"><header><b>2</b><h2>Mục đích tặng quà &amp; thời hạn dự kiến</h2></header><div className="b2b-purpose-grid">{PURPOSES.map((item) => <label className={purpose === item.code ? "selected" : ""} key={item.code}><input type="radio" name="purpose" checked={purpose === item.code} onChange={() => setPurpose(item.code)} /><strong>{item.label}</strong><small>{item.description}</small></label>)}</div><div className="b2b-company-grid"><label><span>Ngày dự kiến cần nhận hàng *</span><input type="date" min={minimumDate} value={deliveryDate} onChange={(event) => setDeliveryDate(event.target.value)} /></label><label><span>Tỉnh/Thành phố nhận hàng *</span><select value={provinceCode} onChange={(event) => setProvinceCode(event.target.value)}>{context.deliveryProvinces.map((item) => <option key={item.provinceCode} value={item.provinceCode}>{item.provinceName}</option>)}</select></label></div></section>

      <section className="b2b-card"><header><b>3</b><h2>Thức quà Cung đình &amp; số lượng dự kiến</h2><em>MOQ theo từng SKU</em></header><div className="b2b-products">{context.catalog.map((item) => <article className={quantities[item.skuId] > 0 ? "selected" : ""} key={item.skuId}>{item.imageUrl ? <div className="b2b-product-image" aria-hidden="true" style={{ backgroundImage: `url(${resolveProductImageSource(item.imageUrl)})` }} /> : <div className="b2b-product-placeholder">Ô MẠ</div>}<h3>{item.name}</h3><p>{item.variant}</p><strong>{VND.format(item.unitPriceVnd)}<small>/set</small></strong><label><span>Số lượng · tối thiểu {item.minimumQuantity}</span><input type="number" min="0" step={item.minimumQuantity} value={quantities[item.skuId] ?? 0} onChange={(event) => setQuantities((current) => ({ ...current, [item.skuId]: Math.max(0, Number(event.target.value) || 0) }))} /></label></article>)}</div><div className="b2b-estimate"><div><span>Tổng số lượng dự kiến</span><strong>{totalQuantity} set</strong></div><div><span>Ưu đãi tham khảo</span><strong>{tier ? `${tier.discountPercent}% · ${tier.label}` : "Chờ đạt MOQ"}</strong></div><div><span>Tạm tính tham khảo</span><strong>{VND.format(estimate)}</strong></div><small>Giá chính thức, thuế, vận chuyển và tùy biến sẽ được chốt trên Quote Version do Sales Manager phát hành.</small></div></section>

      <section className="b2b-card"><header><b>4</b><h2>Tùy biến dấu ấn doanh nghiệp</h2></header><div className="b2b-options">{([['engraveLogo','Khắc logo laser / ép nhũ kim'],['customSleeve','Thiết kế sleeve theo nhận diện'],['greetingCard','Thiệp chúc mừng riêng'],['ribbon','Ruy băng theo màu thương hiệu']] as const).map(([key, label]) => <label key={key}><input type="checkbox" checked={branding[key]} onChange={(event) => setBranding((current) => ({ ...current, [key]: event.target.checked }))} /><span>{label}</span></label>)}</div><label className="support-upload b2b-upload"><span>⇧</span><strong>Tải logo hoặc Brand Guidelines</strong><small>PNG, JPEG, WebP, SVG, PDF · tối đa 3 tệp · 25 MiB/tệp</small><input type="file" multiple accept="image/png,image/jpeg,image/webp,image/svg+xml,application/pdf" onChange={chooseFiles} /></label>{fileError ? <div className="status-notice error">{fileError}</div> : null}<div className="support-files">{attachments.map((file) => <article key={file.clientReference}><span>{file.mediaType === "application/pdf" ? "PDF" : "LOGO"}</span><div><strong>{file.fileName}</strong><small>{(file.sizeBytes / 1024 / 1024).toFixed(1)} MiB · metadata mock</small></div><button type="button" onClick={() => setAttachments((current) => current.filter((item) => item.clientReference !== file.clientReference))}>×</button></article>)}</div></section>

      <section className="b2b-card"><header><b>5</b><h2>Ghi chú quy cách &amp; xác nhận</h2></header><label className="support-field"><span>Yêu cầu chi tiết</span><textarea maxLength={2000} value={notes} onChange={(event) => setNotes(event.target.value)} placeholder="Ngân sách mục tiêu, quy cách đóng gói, địa điểm giao, thông điệp thiệp…" /><small>{notes.length}/2000 ký tự</small></label><div className="b2b-confirmations"><label><input type="checkbox" checked={invoiceRequested} onChange={(event) => setInvoiceRequested(event.target.checked)} /> Yêu cầu hóa đơn theo thông tin doanh nghiệp đã xác minh</label><label><input type="checkbox" checked={sampleRequested} onChange={(event) => setSampleRequested(event.target.checked)} /> Đề nghị chuyên viên tư vấn hộp mẫu trước khi chốt</label></div></section>
      {submitError ? <div className="status-notice error" role="alert">{submitError}</div> : null}
      <div className="b2b-submit"><span>Yêu cầu chỉ được tạo khi đại diện doanh nghiệp có quyền.</span><button className="primary-button" disabled={submitting}>{submitting ? "Đang gửi yêu cầu…" : "Gửi Yêu Cầu Nhận Báo Giá B2B"}</button></div>
    </form>

      <aside className="b2b-aside"><section><span className="eyebrow">Chuyên viên phụ trách</span><h2>Lê Thị Đoan Trang</h2><p>Phòng Chăm Sóc Doanh Nghiệp<br />Hotline B2B: 0905 128 688<br />b2b@omabanhmuthue.vn</p></section><section><span className="eyebrow">Bảng chiết khấu</span><h2>Theo số lượng</h2>{context.discountTiers.map((item) => <div className="b2b-tier" key={item.minimumQuantity}><span>{item.minimumQuantity}{item.maximumQuantity ? `–${item.maximumQuantity}` : "+"} set</span><strong>{item.discountPercent}%</strong><small>{item.label}</small></div>)}</section><section><span className="eyebrow">Yêu cầu gần đây</span><h2>Hồ sơ doanh nghiệp</h2>{context.recentRequests.map((item) => <article className="b2b-request-row" key={item.requestId}><div><strong>{item.requestNumber}</strong><small>{item.purposeLabel} · {item.totalQuantity} set</small></div><em>{item.statusLabel}</em>{item.quoteId ? <Link href={`/b2b/quotes/${encodeURIComponent(item.quoteId)}`}>Xem báo giá →</Link> : null}</article>)}</section></aside>
    </div>
  </main>;
}
