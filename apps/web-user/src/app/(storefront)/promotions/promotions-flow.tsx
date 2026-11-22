"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { HeritageBanner } from "@/components/heritage/heritage-banner";
import { Motif } from "@/components/heritage/motif";
import type { Voucher, VoucherValidation } from "@/lib/customer-core/types";
import { CustomerCoreApiError } from "@/lib/customer-core/types";
import { customerCoreService } from "@/services/customer-core-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const DATE = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric" });

function voucherValue(voucher: Voucher) {
  if (voucher.discount_type === "PERCENTAGE") return `${voucher.discount_value}%`;
  if (voucher.discount_type === "FREESHIP") return "Freeship";
  return VND.format(voucher.discount_value);
}

export function PromotionsFlow() {
  const searchParams = useSearchParams();
  const { hasCapability } = useCustomerSession();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [vouchers, setVouchers] = useState<Voucher[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [retryKey, setRetryKey] = useState(0);
  const [code, setCode] = useState("");
  const [orderAmount, setOrderAmount] = useState(345000);
  const [validating, setValidating] = useState(false);
  const [result, setResult] = useState<VoucherValidation>();
  const [validationError, setValidationError] = useState("");

  useEffect(() => {
    let active = true;
    customerCoreService.getVouchers(scenario)
      .then((payload) => { if (active) setVouchers(payload.vouchers); })
      .catch((cause: unknown) => {
        if (active) setError(cause instanceof CustomerCoreApiError ? cause.message : "Chưa thể tải kho ưu đãi.");
      })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [retryKey, scenario]);

  function retry() {
    setLoading(true);
    setError("");
    setRetryKey((value) => value + 1);
  }

  function chooseVoucher(voucherCode: string) {
    setCode(voucherCode);
    setResult(undefined);
    setValidationError("");
    document.getElementById("voucher-checker")?.scrollIntoView({ behavior: "smooth", block: "center" });
  }

  async function validate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedCode = code.trim().toUpperCase();
    if (!normalizedCode || !Number.isFinite(orderAmount) || orderAmount <= 0) {
      setValidationError("Nhập mã ưu đãi và giá trị đơn hàng hợp lệ.");
      return;
    }
    setValidating(true);
    setResult(undefined);
    setValidationError("");
    try {
      setResult(await customerCoreService.validateVoucher({ voucher_code: normalizedCode, subtotal_amount: orderAmount }, scenario));
    } catch (cause) {
      setValidationError(cause instanceof CustomerCoreApiError ? cause.message : "Chưa thể kiểm tra mã ưu đãi.");
    } finally {
      setValidating(false);
    }
  }

  return (
    <main className="gh-page" id="main-content">
      <section className="gh-home-band" aria-labelledby="promotions-title">
        <Motif name="02-canh-hoa" className="gh-band__blossom" />
        <Motif name="06-mai-ngoi" className="gh-band__veranda" />
        <div className="gh-wrap">
          <HeritageBanner level={1} titleId="promotions-title" kicker="Đặc quyền Tri Kỷ" title="Ưu đãi dành cho những cuộc sum vầy"
            lead="Khám phá voucher đang phát hành và kiểm tra điều kiện trước khi dùng tại bước thanh toán."
            image={{ src: "/images/heritage/photos/sp-hop-qua-doan-vien.jpg", alt: "Hộp quà Đoàn Viên Gói Huế gồm mè xửng và trà cung đình" }}>
            <div className="gh-banner__actions">
              <a className="gh-btn gh-btn--fold" href="#voucher-checker">Kiểm tra mã ưu đãi <span className="gh-btn__arrow" aria-hidden="true">→</span></a>
              {hasCapability("LOYALTY_VIEW") ? <Link className="gh-btn gh-btn--ghost" href="/account/loyalty">Xem ví điểm của tôi</Link> : null}
            </div>
          </HeritageBanner>
        </div>
      </section>

      <section className="gh-page-section" aria-labelledby="voucher-title">
        <Motif name="05-may" className="gh-page-section__cloud" />
        <div className="gh-wrap">
          <div className="gh-section-header gh-section-header--center">
            <div className="gh-section-header__line" />
            <h2 id="voucher-title">Mã ưu đãi đang khả dụng</h2>
            <div className="gh-section-header__line" />
          </div>
          <p className="gh-page-section__note">Điều kiện cuối cùng được xác nhận khi đặt hàng.</p>

          {loading ? <div className="gh-skeleton-grid" aria-busy="true" aria-label="Đang tải ưu đãi">
            {[0, 1, 2].map((index) => <div key={index} className="gh-frame-card gh-skeleton-card" />)}
          </div> : null}
          {!loading && error ? <div className="gh-frame-functional gh-frame-functional--fold gh-state" role="alert">
            <h2>Chưa tải được kho ưu đãi</h2>
            <p>{error}</p>
            <div className="gh-state__actions"><button type="button" className="gh-btn gh-btn--brown" onClick={retry}>Thử lại</button></div>
          </div> : null}
          {!loading && !error && !vouchers.length ? <div className="gh-frame-functional gh-frame-functional--fold gh-state">
            <h2>Chưa có ưu đãi đang phát hành</h2>
            <p>Hãy quay lại vào dịp khác để nhận đặc quyền mới.</p>
          </div> : null}
          {!loading && !error && vouchers.length ? <div className="gh-coupons">{vouchers.map((voucher) => <article key={voucher.voucher_code} className="gh-coupon">
            <div className="gh-coupon__value gh-surface-imperial">
              <strong>{voucherValue(voucher)}</strong>
              <span>{voucher.discount_type === "FREESHIP" ? "phí vận chuyển" : "giá trị ưu đãi"}</span>
            </div>
            <div className="gh-coupon__body">
              <span className="gh-badge gh-badge--gold">{voucher.voucher_code}</span>
              <h3>{voucher.title}</h3>
              <dl>
                <div><dt>Đơn tối thiểu</dt><dd>{VND.format(voucher.min_order_amount)}</dd></div>
                <div><dt>Giảm tối đa</dt><dd>{VND.format(voucher.max_discount_amount)}</dd></div>
              </dl>
              <div className="gh-coupon__foot">
                <small>Hết hạn {DATE.format(new Date(voucher.expires_at))}</small>
                <button type="button" className="gh-btn gh-btn--paper" onClick={() => chooseVoucher(voucher.voucher_code)}>Kiểm tra mã</button>
              </div>
            </div>
          </article>)}</div> : null}
        </div>
      </section>

      <section className="gh-page-section" id="voucher-checker" aria-labelledby="voucher-checker-title">
        <Motif name="12-hoa-sen-cung-dinh-phai" className="gh-page-section__lotus" />
        <div className="gh-wrap">
          <div className="gh-frame-plaque gh-checker">
            <div>
              <span className="gh-kicker">Kiểm tra điều kiện</span>
              <h2 id="voucher-checker-title">Ước tính ưu đãi cho đơn hàng</h2>
              <p>Kết quả giúp bạn chuẩn bị trước; bước thanh toán vẫn là nơi xác nhận số tiền cuối cùng.</p>
            </div>
            <form className="gh-checker__form" onSubmit={validate}>
              <div className="gh-field-group">
                <label className="gh-field-label" htmlFor="voucher-code">Mã ưu đãi</label>
                <input id="voucher-code" className="gh-field" value={code} required onChange={(event) => { setCode(event.target.value); setResult(undefined); setValidationError(""); }} placeholder="Nhập mã voucher" />
              </div>
              <div className="gh-field-group">
                <label className="gh-field-label" htmlFor="voucher-order-amount">Giá trị đơn hàng (VNĐ)</label>
                <input id="voucher-order-amount" className="gh-field" type="number" min="1" step="1000" value={orderAmount} required onChange={(event) => { setOrderAmount(Number(event.target.value)); setResult(undefined); setValidationError(""); }} />
              </div>
              <button className="gh-btn gh-btn--brown gh-btn--fold" disabled={validating}>{validating ? "Đang kiểm tra…" : "Kiểm tra ưu đãi"}</button>
              {result ? <div className={`gh-result ${result.is_valid ? "gh-result--ok" : "gh-result--bad"}`} role="status"><strong>{result.is_valid ? `Tiết kiệm ${VND.format(result.discount_amount.units)}` : "Chưa thể áp dụng"}</strong><p>{result.message}</p></div> : null}
              {validationError ? <div className="gh-result gh-result--bad" role="alert"><strong>Mã chưa được áp dụng</strong><p>{validationError}</p></div> : null}
            </form>
          </div>
        </div>
      </section>
    </main>
  );
}
