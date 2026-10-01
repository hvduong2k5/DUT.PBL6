"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
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
    <main className="promotions-page">
      <section className="promotions-hero"><div><p className="eyebrow">Đặc quyền Tri Kỷ</p><h1>Ưu đãi dành cho những cuộc sum vầy</h1><p>Khám phá voucher đang phát hành và kiểm tra điều kiện trước khi dùng tại Checkout.</p></div><Link href="/account/loyalty">Xem ví điểm của tôi →</Link></section>

      <section className="voucher-section" aria-labelledby="voucher-title">
        <header><div><p className="eyebrow">Kho voucher</p><h2 id="voucher-title">Mã ưu đãi đang khả dụng</h2></div><small>Điều kiện cuối cùng được xác nhận khi đặt hàng.</small></header>
        {loading ? <div className="voucher-loading" aria-busy="true"><span /><span /><span /></div> : null}
        {!loading && error ? <div className="voucher-state" role="alert"><p>{error}</p><button type="button" onClick={retry}>Thử lại</button></div> : null}
        {!loading && !error && !vouchers.length ? <div className="voucher-state"><strong>Chưa có ưu đãi đang phát hành</strong><p>Hãy quay lại vào dịp khác để nhận đặc quyền mới.</p></div> : null}
        {!loading && !error && vouchers.length ? <div className="voucher-grid">{vouchers.map((voucher) => <article key={voucher.voucher_code}>
          <div className="voucher-value"><strong>{voucherValue(voucher)}</strong><span>{voucher.discount_type === "FREESHIP" ? "phí vận chuyển" : "giá trị ưu đãi"}</span></div>
          <div className="voucher-copy"><span className="voucher-code">{voucher.voucher_code}</span><h3>{voucher.title}</h3><dl><div><dt>Đơn tối thiểu</dt><dd>{VND.format(voucher.min_order_amount)}</dd></div><div><dt>Giảm tối đa</dt><dd>{VND.format(voucher.max_discount_amount)}</dd></div></dl><small>Hết hạn {DATE.format(new Date(voucher.expires_at))}</small></div>
          <button type="button" onClick={() => chooseVoucher(voucher.voucher_code)}>Kiểm tra mã</button>
        </article>)}</div> : null}
      </section>

      <section className="voucher-checker" id="voucher-checker">
        <div><p className="eyebrow">Kiểm tra điều kiện</p><h2>Ước tính ưu đãi cho đơn hàng</h2><p>Kết quả giúp bạn chuẩn bị trước; Checkout vẫn là nơi xác nhận số tiền cuối cùng.</p></div>
        <form onSubmit={validate}>
          <label htmlFor="voucher-code">Mã ưu đãi</label><input id="voucher-code" value={code} required onChange={(event) => { setCode(event.target.value); setResult(undefined); setValidationError(""); }} placeholder="Nhập mã voucher" />
          <label htmlFor="voucher-order-amount">Giá trị đơn hàng</label><input id="voucher-order-amount" type="number" min="1" step="1000" value={orderAmount} required onChange={(event) => { setOrderAmount(Number(event.target.value)); setResult(undefined); setValidationError(""); }} />
          <button className="primary-button" disabled={validating}>{validating ? "Đang kiểm tra…" : "Kiểm tra ưu đãi"}</button>
        </form>
        {result ? <div className={`voucher-result ${result.is_valid ? "valid" : "invalid"}`} role="status"><strong>{result.is_valid ? `Tiết kiệm ${VND.format(result.discount_amount.units)}` : "Chưa thể áp dụng"}</strong><p>{result.message}</p></div> : null}
        {validationError ? <div className="voucher-result invalid" role="alert"><strong>Mã chưa được áp dụng</strong><p>{validationError}</p></div> : null}
      </section>
    </main>
  );
}
