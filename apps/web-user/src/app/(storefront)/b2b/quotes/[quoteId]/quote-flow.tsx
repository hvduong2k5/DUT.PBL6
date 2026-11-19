"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import type { AcceptB2BQuoteResult, B2BQuoteDetail } from "@/lib/b2b/types";
import { B2BApiError } from "@/lib/b2b/types";
import { b2bService } from "@/services/b2b-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const DATE = new Intl.DateTimeFormat("vi-VN", { dateStyle: "long" });

export function B2BQuoteFlow({ quoteId }: { quoteId: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [quote, setQuote] = useState<B2BQuoteDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [confirmed, setConfirmed] = useState(false);
  const [accepting, setAccepting] = useState(false);
  const [accepted, setAccepted] = useState<AcceptB2BQuoteResult | null>(null);

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try { setQuote(await b2bService.quote(quoteId, scenario)); }
    catch (cause) { setError(cause instanceof B2BApiError ? cause.message : "Không thể tải báo giá."); }
    finally { setLoading(false); }
  }, [quoteId, scenario]);
  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  async function accept() {
    if (!confirmed) { setError("Hãy xác nhận đã đọc và đồng ý toàn bộ điều khoản."); return; }
    setAccepting(true); setError("");
    try { setAccepted(await b2bService.acceptQuote(quoteId, `b2b-accept-${globalThis.crypto.randomUUID()}`, scenario)); }
    catch (cause) { setError(cause instanceof B2BApiError ? cause.message : "Không thể chấp thuận báo giá lúc này."); }
    finally { setAccepting(false); }
  }

  if (loading) return <main className="b2b-page"><div className="b2b-loading" aria-busy="true"><div /><div /></div></main>;
  if (!quote) return <main className="b2b-page"><section className="return-state error"><span>!</span><h1>Không thể mở báo giá</h1><p>{error}</p><div><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href="/b2b">Về B2B</Link></div></section></main>;

  return <main className="b2b-page b2b-quote-page"><nav className="detail-breadcrumbs"><Link href="/">Trang chủ</Link><span>›</span><Link href="/b2b">Báo giá B2B</Link><span>›</span><strong>{quote.quoteNumber}</strong></nav>
    <section className="b2b-quote-hero"><div><span className="eyebrow">Dự thảo hợp đồng kinh tế · phiên bản {quote.version}</span><h1>Bảng Báo Giá Quà Biếu &amp;<br />Dự Thảo Hợp Đồng Kinh Tế</h1><p>Dành riêng cho {quote.organization.legalName}</p></div><div className="b2b-quote-meta"><span><small>Mã báo giá</small><strong>{quote.quoteNumber}</strong></span><span><small>Ngày phát hành</small><strong>{DATE.format(new Date(quote.issuedAt))}</strong></span><span><small>Hiệu lực đến</small><strong>{DATE.format(new Date(quote.expiresAt))}</strong></span></div><footer><strong>Trạng thái: {accepted ? "Đã chấp thuận" : quote.statusLabel}</strong><span>{quote.canAccept && !accepted ? "Đang chờ doanh nghiệp duyệt và ký kết điện tử" : "Báo giá không còn hành động chấp thuận"}</span></footer></section>
    {accepted ? <section className="b2b-created" role="status"><span>✓</span><div><h2>Đã chấp thuận báo giá</h2><p>{accepted.message} Order tham chiếu: {accepted.orderNumber}.</p></div><Link href="/account/orders">Theo dõi Order</Link></section> : null}

    <div className="b2b-quote-layout"><div className="b2b-quote-main"><section className="b2b-card"><header><b>1</b><h2>Danh mục ngự phẩm &amp; lễ vật kèm theo</h2></header><div className="b2b-quote-items">{quote.items.map((item) => <article key={item.skuId}><div><strong>{item.name}</strong><small>{item.variant}</small></div><span>{item.quantity} set</span><span>{VND.format(item.unitPriceVnd)}/set</span><b>{VND.format(item.lineTotalVnd)}</b></article>)}</div>{quote.customizations.length ? <div className="b2b-customizations"><strong>Tùy biến đã bao gồm</strong>{quote.customizations.map((item) => <span key={item}>✓ {item}</span>)}</div> : null}</section>
      <section className="b2b-card"><header><b>2</b><h2>Tóm tắt điều khoản hợp đồng</h2></header><div className="b2b-terms">{quote.terms.map((term, index) => <article key={term.title}><b>{index + 1}</b><div><strong>{term.title}</strong><p>{term.description}</p></div></article>)}</div></section></div>
      <aside className="b2b-quote-aside"><section className="b2b-totals"><span className="eyebrow">Quyết toán tài chính B2B</span><div><span>Tổng giá trị gốc</span><b>{VND.format(quote.totals.merchandiseVnd)}</b></div><div><span>Chiết khấu</span><b>−{VND.format(quote.totals.discountVnd)}</b></div><div><span>Tùy biến</span><b>{VND.format(quote.totals.customizationVnd)}</b></div><div><span>VAT</span><b>{VND.format(quote.totals.vatVnd)}</b></div><strong className="b2b-grand-total"><small>Tổng giá trị hợp đồng</small>{VND.format(quote.totals.grandTotalVnd)}</strong><div className="deposit"><span>Đặt cọc {quote.totals.depositPercent}%</span><b>{VND.format(quote.totals.depositVnd)}</b></div><div><span>Còn lại</span><b>{VND.format(quote.totals.remainingVnd)}</b></div></section>
      <section className="b2b-accept"><span className="eyebrow">Ký duyệt hợp đồng điện tử</span><p>Người chấp thuận phải là đại diện có thẩm quyền của {quote.organization.legalName}.</p><label><input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)} /> Tôi đã đọc và đồng ý toàn bộ báo giá cùng điều khoản phiên bản {quote.version}.</label>{error ? <div className="status-notice error">{error}</div> : null}<button className="primary-button" type="button" disabled={!quote.canAccept || accepting || Boolean(accepted)} onClick={() => void accept()}>{accepting ? "Đang ghi nhận…" : accepted ? "Đã chấp thuận" : "Chấp Thuận & Đặt Cọc"}</button><small>Thao tác được ghi nhận theo đúng Quote Version và không đồng nghĩa Order đã thanh toán.</small></section>
      <section><span className="eyebrow">Thông tin thanh toán</span><h2>{quote.payment.bankName}</h2><p>{quote.payment.accountName}<br />{quote.payment.accountNumberMasked}<br />Nội dung: <strong>{quote.payment.transferContent}</strong></p></section><section><span className="eyebrow">Chuyên viên phụ trách</span><h2>{quote.contact.displayName}</h2><p>{quote.contact.title}<br />{quote.contact.phone}<br />{quote.contact.email}</p></section></aside></div>
  </main>;
}
