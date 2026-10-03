"use client";

import Image from "next/image";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ProductImage } from "@/components/product/product-image";
import { useEffect, useMemo, useState } from "react";
import type {
  CheckoutConfirmation,
  CheckoutPaymentMethod,
  CheckoutPreparation,
  CheckoutRecipient,
  SavedCheckoutAddress,
  ShippingAddress,
  ShippingQuote
} from "@/lib/checkout/types";
import { CheckoutApiError } from "@/lib/checkout/types";
import { getCheckoutFormErrors, validateShippingAddress } from "@/lib/checkout/validation";
import { checkoutService } from "@/services/checkout-service";
import { customerCoreService } from "@/services/customer-core-service";
import { CustomerCoreApiError, type Voucher, type VoucherValidation } from "@/lib/customer-core/types";
import { useCartSummary } from "@/components/cart/cart-summary-provider";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const EMPTY_RECIPIENT: CheckoutRecipient = { fullName: "", phone: "", email: "" };
const EMPTY_ADDRESS: ShippingAddress = { provinceCode: "", provinceName: "", wardCode: "", wardName: "", addressLine: "" };

function CheckoutSteps({ complete = false }: { complete?: boolean }) {
  return (
    <ol className="checkout-steps" aria-label="Tiến trình mua hàng">
      <li className="done"><span aria-hidden="true">✓</span><div><small>Bước 1</small><strong>Giỏ hàng</strong></div></li>
      <li className={complete ? "done" : "active"} aria-current={complete ? undefined : "step"}><span aria-hidden="true">{complete ? "✓" : "2"}</span><div><small>Bước 2</small><strong>Thông tin &amp; xác nhận</strong></div></li>
      <li className={complete ? "active" : ""} aria-current={complete ? "step" : undefined}><span aria-hidden="true">3</span><div><small>Bước 3</small><strong>Chờ thanh toán</strong></div></li>
    </ol>
  );
}

function CheckoutLoading() {
  return <div className="checkout-loading" aria-busy="true" aria-label="Đang kiểm tra giỏ hàng"><div /><div /></div>;
}

function optionLabel(address: SavedCheckoutAddress) {
  return `${address.label} · ${address.address.addressLine}, ${address.address.wardName}`;
}

export function CheckoutFlow() {
  const { reload: reloadCartSummary } = useCartSummary();
  const searchParams = useSearchParams();
  const itemIds = useMemo(() => (searchParams.get("items") ?? "").split(",").filter(Boolean), [searchParams]);
  const mockScenario = searchParams.get("mockScenario") ?? undefined;
  const cartScenario = searchParams.get("cartScenario") ?? undefined;
  const requestOptions = useMemo(() => ({ mockScenario, cartScenario }), [mockScenario, cartScenario]);
  const [preparation, setPreparation] = useState<CheckoutPreparation | null>(null);
  const [recipient, setRecipient] = useState<CheckoutRecipient>(EMPTY_RECIPIENT);
  const [address, setAddress] = useState<ShippingAddress>(EMPTY_ADDRESS);
  const [selectedAddressId, setSelectedAddressId] = useState("");
  const [quote, setQuote] = useState<ShippingQuote | null>(null);
  const [shippingOptionId, setShippingOptionId] = useState("");
  const [paymentMethod, setPaymentMethod] = useState<CheckoutPaymentMethod>("BANK_TRANSFER");
  const [availableVouchers, setAvailableVouchers] = useState<Voucher[]>([]);
  const [voucherCode, setVoucherCode] = useState("");
  const [voucherResult, setVoucherResult] = useState<VoucherValidation>();
  const [voucherError, setVoucherError] = useState("");
  const [validatingVoucher, setValidatingVoucher] = useState(false);
  const [priceAcknowledged, setPriceAcknowledged] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [pageError, setPageError] = useState("");
  const [quoteError, setQuoteError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [loading, setLoading] = useState(itemIds.length > 0);
  const [quoting, setQuoting] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [retryKey, setRetryKey] = useState(0);
  const [confirmation, setConfirmation] = useState<CheckoutConfirmation | null>(null);
  const [idempotencyKey] = useState(() => `checkout-${globalThis.crypto.randomUUID()}`);

  useEffect(() => {
    let active = true;
    customerCoreService.getVouchers(mockScenario)
      .then((payload) => { if (active) setAvailableVouchers(payload.vouchers); })
      .catch(() => { if (active) setAvailableVouchers([]); });
    return () => { active = false; };
  }, [mockScenario]);

  useEffect(() => {
    if (!itemIds.length) return;
    let active = true;
    checkoutService.prepare({ itemIds }, requestOptions).then((payload) => {
      if (!active) return;
      setPreparation(payload);
      const preferred = payload.savedAddresses.find((saved) => saved.isDefault) ?? payload.savedAddresses[0];
      if (preferred) {
        setSelectedAddressId(preferred.addressId);
        setRecipient(preferred.recipient);
        setAddress(preferred.address);
      } else if (payload.defaultRecipient) setRecipient(payload.defaultRecipient);
    }).catch((cause: unknown) => {
      if (!active) return;
      setPageError(cause instanceof CheckoutApiError ? cause.message : "Không thể chuẩn bị Checkout lúc này.");
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [itemIds, requestOptions, retryKey]);

  useEffect(() => {
    if (!preparation || validateShippingAddress(address).length > 0) return;
    let active = true;
    const timer = window.setTimeout(() => {
      setQuoting(true);
      setQuoteError("");
      checkoutService.quoteShipping({ checkoutSessionId: preparation.checkoutSessionId, itemIds, address }, requestOptions).then((payload) => {
        if (!active) return;
        setQuote(payload);
        setShippingOptionId((current) => payload.options.some((option) => option.shippingOptionId === current) ? current : payload.options[0]?.shippingOptionId ?? "");
      }).catch((cause: unknown) => {
        if (!active) return;
        setQuote(null);
        setShippingOptionId("");
        setQuoteError(cause instanceof CheckoutApiError ? cause.message : "Chưa thể lấy phí vận chuyển.");
      }).finally(() => { if (active) setQuoting(false); });
    }, 450);
    return () => { active = false; window.clearTimeout(timer); };
  }, [address, itemIds, preparation, requestOptions, retryKey]);

  function invalidateShippingQuote() {
    setQuote(null);
    setShippingOptionId("");
    setQuoteError("");
  }

  function applySavedAddress(addressId: string) {
    setSelectedAddressId(addressId);
    const saved = preparation?.savedAddresses.find((entry) => entry.addressId === addressId);
    if (!saved) return;
    invalidateShippingQuote();
    setRecipient(saved.recipient);
    setAddress(saved.address);
    setFieldErrors({});
  }

  function updateProvince(provinceCode: string) {
    const province = preparation?.locations.find((entry) => entry.provinceCode === provinceCode);
    const savedProvince = preparation?.savedAddresses.find((entry) => entry.address.provinceCode === provinceCode)?.address;
    setSelectedAddressId("");
    invalidateShippingQuote();
    setAddress((current) => ({ ...current, provinceCode, provinceName: province?.provinceName ?? savedProvince?.provinceName ?? "", wardCode: "", wardName: "" }));
  }

  function updateWard(wardCode: string) {
    const province = preparation?.locations.find((entry) => entry.provinceCode === address.provinceCode);
    const ward = province?.wards.find((entry) => entry.wardCode === wardCode);
    const savedWard = preparation?.savedAddresses.find((entry) => entry.address.provinceCode === address.provinceCode && entry.address.wardCode === wardCode)?.address;
    setSelectedAddressId("");
    invalidateShippingQuote();
    setAddress((current) => ({ ...current, wardCode, wardName: ward?.wardName ?? savedWard?.wardName ?? "" }));
  }

  async function applyVoucher() {
    if (!preparation) return;
    const normalizedCode = voucherCode.trim().toUpperCase();
    if (!normalizedCode) {
      setVoucherResult(undefined);
      setVoucherError("Hãy nhập hoặc chọn mã ưu đãi.");
      return;
    }
    setValidatingVoucher(true);
    setVoucherResult(undefined);
    setVoucherError("");
    try {
      const result = await customerCoreService.validateVoucher({ voucher_code: normalizedCode, subtotal_amount: preparation.subtotalVnd }, mockScenario);
      setVoucherCode(normalizedCode);
      setVoucherResult(result);
      if (!result.is_valid) setVoucherError(result.message || "Mã ưu đãi không áp dụng được cho giỏ hàng này.");
    } catch (cause) {
      setVoucherError(cause instanceof CustomerCoreApiError ? cause.message : "Chưa thể kiểm tra mã ưu đãi.");
    } finally {
      setValidatingVoucher(false);
    }
  }

  async function confirmCheckout() {
    if (!preparation) return;
    const errors = getCheckoutFormErrors(recipient, address);
    if (!shippingOptionId) errors.shippingOptionId = "Vui lòng chọn phương thức vận chuyển khả dụng.";
    if (voucherCode.trim() && !voucherResult?.is_valid) errors.voucherCode = "Hãy kiểm tra hợp lệ mã ưu đãi trước khi đặt hàng.";
    if (preparation.requiresPriceAcknowledgement && !priceAcknowledged) errors.priceAcknowledgement = "Vui lòng xác nhận giá mới trước khi đặt hàng.";
    setFieldErrors(errors);
    setSubmitError("");
    if (Object.keys(errors).length) {
      document.getElementById("checkout-form-start")?.focus();
      return;
    }
    setSubmitting(true);
    try {
      const result = await checkoutService.confirm({
        checkoutSessionId: preparation.checkoutSessionId,
        itemIds,
        recipient,
        address,
        shippingOptionId,
        paymentMethod,
        idempotencyKey,
        priceRevalidatedAt: preparation.priceRevalidatedAt,
        priceChangesAcknowledged: priceAcknowledged,
        voucherCode: voucherResult?.is_valid ? voucherCode.trim().toUpperCase() : undefined
      }, requestOptions);
      setConfirmation(result);
      void reloadCartSummary();
    } catch (cause) {
      if (cause instanceof CheckoutApiError) {
        setSubmitError(cause.itemIssues.length ? `${cause.message} ${cause.itemIssues.map((issue) => issue.message).join(" ")}` : cause.message);
        const mapped = Object.fromEntries(cause.errors.map((error) => [error.field, error.message]));
        setFieldErrors((current) => ({ ...current, ...mapped }));
        if (["CHECKOUT_ITEMS_UNAVAILABLE", "CART_CHANGED", "PRICE_CHANGED"].includes(cause.code)) setPreparation(null);
      } else setSubmitError("Không thể xác nhận đơn lúc này.");
    } finally {
      setSubmitting(false);
    }
  }

  if (!itemIds.length) {
    return (
      <main className="checkout-page">
        <CheckoutSteps />
        <section className="checkout-state-card"><span aria-hidden="true">◇</span><p className="eyebrow">Chưa có lựa chọn Checkout</p><h1>Hãy chọn sản phẩm từ giỏ hàng</h1><p>Checkout chỉ xử lý những dòng khả dụng mà bạn chủ động chọn trong giỏ.</p><Link className="primary-link" href="/cart">Quay lại giỏ hàng</Link></section>
      </main>
    );
  }

  if (loading) return <main className="checkout-page"><CheckoutSteps /><CheckoutLoading /></main>;

  if (pageError || (!preparation && !confirmation)) {
    return (
      <main className="checkout-page">
        <CheckoutSteps />
        <section className="checkout-state-card error" role="alert"><span aria-hidden="true">!</span><p className="eyebrow">Checkout cần được làm mới</p><h1>Chưa thể kiểm tra giỏ hàng</h1><p>{pageError || submitError || "Dữ liệu sản phẩm đã thay đổi. Vui lòng quay lại giỏ để kiểm tra."}</p><div><button className="primary-button" type="button" onClick={() => { setLoading(true); setPageError(""); setRetryKey((value) => value + 1); }}>Thử tải lại</button><Link className="secondary-link" href={cartScenario ? `/cart?mockScenario=${cartScenario}` : "/cart"}>Về giỏ hàng</Link></div></section>
      </main>
    );
  }

  if (confirmation) {
    return (
      <main className="checkout-page">
        <CheckoutSteps complete />
        <section className="checkout-confirmed" aria-labelledby="checkout-confirmed-title">
          <span className="checkout-confirmed-icon" aria-hidden="true">✓</span>
          <p className="eyebrow">Đã tạo Order và giữ hàng</p>
          <h1 id="checkout-confirmed-title">Đơn {confirmation.orderNumber} đang chờ bước tiếp theo</h1>
          <p>{confirmation.message}</p>
          {confirmation.vietQrUrl && confirmation.paymentMethod === "BANK_TRANSFER" ? <div className="checkout-vietqr"><Image src={confirmation.vietQrUrl} alt={`Mã VietQR cho đơn ${confirmation.orderNumber}`} width={260} height={260} unoptimized /><small>Quét mã trước {new Date(confirmation.paymentExpiresAt ?? confirmation.reservationExpiresAt).toLocaleString("vi-VN")}</small></div> : null}
          <dl><div><dt>Trạng thái đơn</dt><dd>{confirmation.status === "PENDING_PAYMENT" ? "Chờ thanh toán" : "Đã tiếp nhận"}</dd></div><div><dt>Tổng thanh toán</dt><dd>{VND.format(confirmation.totalVnd)}</dd></div><div><dt>Giữ hàng đến</dt><dd>{new Date(confirmation.reservationExpiresAt).toLocaleString("vi-VN")}</dd></div></dl>
          <div className="checkout-payment-boundary" role="status"><strong>{confirmation.paymentMethod === "BANK_TRANSFER" ? "Đang chờ xác nhận VietQR" : "Đơn COD đã được tiếp nhận"}</strong><p>{confirmation.paymentMethod === "BANK_TRANSFER" ? <>Order vẫn là <code>UNPAID</code>. Trạng thái chỉ thay đổi sau khi hệ thống nhận xác minh thanh toán hợp lệ.</> : "Bạn có thể theo dõi trạng thái xử lý trong chi tiết đơn hàng."}</p></div>
          {confirmation.customerMode === "GUEST" ? <div className="checkout-payment-boundary" role="status"><strong>Quyền xem Guest Order đã được cấp cho đúng đơn này</strong><p>Hãy lưu mã <code>{confirmation.orderNumber}</code>. Khi quyền tạm thời hết hạn, bạn có thể xác minh lại bằng số điện thoại hoặc email đã đặt hàng.</p></div> : null}
          <div className="checkout-confirmed-actions"><Link className="primary-link" href={confirmation.paymentPath}>{confirmation.paymentMethod === "BANK_TRANSFER" ? "Mở trang thanh toán" : "Xem đơn hàng"}</Link>{confirmation.customerMode === "GUEST" ? <Link className="secondary-link" href="/track-order">Tra cứu Guest Order</Link> : null}<Link className="secondary-link" href="/products">Tiếp tục mua sắm</Link></div>
        </section>
      </main>
    );
  }

  if (!preparation) return null;

  const selectedShipping = quote?.options.find((option) => option.shippingOptionId === shippingOptionId);
  const voucherDiscountVnd = voucherResult?.is_valid ? voucherResult.discount_amount.units : 0;
  const totalVnd = Math.max(0, preparation.subtotalVnd + (selectedShipping?.feeVnd ?? 0) - voucherDiscountVnd);
  const provinceOptions = preparation.locations.some((entry) => entry.provinceCode === address.provinceCode) || !address.provinceCode
    ? preparation.locations
    : [...preparation.locations, { provinceCode: address.provinceCode, provinceName: address.provinceName, wards: [] }];
  const knownWards = preparation.locations.find((entry) => entry.provinceCode === address.provinceCode)?.wards ?? [];
  const wards = knownWards.some((entry) => entry.wardCode === address.wardCode) || !address.wardCode
    ? knownWards
    : [...knownWards, { wardCode: address.wardCode, wardName: address.wardName }];

  return (
    <main className="checkout-page">
      <CheckoutSteps />
      <div className="checkout-guest-banner"><span aria-hidden="true">♢</span><div><strong>{preparation.customerMode === "GUEST" ? "Bạn đang đặt hàng với tư cách khách." : "Địa chỉ tài khoản chỉ được sao chép cho đơn hiện tại."}</strong><p>{preparation.customerMode === "GUEST" ? "Không cần đăng nhập hoặc tạo tài khoản để hoàn tất Checkout." : "Chỉnh sửa tại đây không tự thay đổi sổ địa chỉ đã lưu."}</p></div></div>
      {preparation.accountDataWarning ? <div className="checkout-inline-error" role="status"><strong>Thông tin tài khoản chưa tải đầy đủ</strong><p>{preparation.accountDataWarning}</p></div> : null}

      <div className="checkout-layout">
        <div className="checkout-main-column">
          <section className="checkout-panel" aria-labelledby="recipient-title">
            <header><span>1</span><h1 id="recipient-title" tabIndex={-1}>Thông tin người nhận &amp; giao hàng</h1><small>{preparation.customerMode === "GUEST" ? "Khách vãng lai" : "Tài khoản Tri Kỷ"}</small></header>
            <div className="checkout-panel-body" id="checkout-form-start">
              {preparation.savedAddresses.length ? <label className="checkout-field full"><span>Địa chỉ đã lưu</span><select value={selectedAddressId} onChange={(event) => applySavedAddress(event.currentTarget.value)}><option value="">Nhập thông tin mới</option>{preparation.savedAddresses.map((saved) => <option value={saved.addressId} key={saved.addressId}>{optionLabel(saved)}</option>)}</select></label> : null}
              <div className="checkout-form-grid">
                <label className="checkout-field"><span>Họ và tên người nhận <b>*</b></span><input value={recipient.fullName} aria-invalid={Boolean(fieldErrors.fullName)} onChange={(event) => { const value = event.currentTarget.value; setRecipient((current) => ({ ...current, fullName: value })); setSelectedAddressId(""); }} />{fieldErrors.fullName ? <small role="alert">{fieldErrors.fullName}</small> : null}</label>
                <label className="checkout-field"><span>Số điện thoại liên hệ <b>*</b></span><input inputMode="tel" value={recipient.phone} aria-invalid={Boolean(fieldErrors.phone)} onChange={(event) => { const value = event.currentTarget.value; setRecipient((current) => ({ ...current, phone: value })); setSelectedAddressId(""); }} />{fieldErrors.phone ? <small role="alert">{fieldErrors.phone}</small> : null}</label>
                <label className="checkout-field full"><span>Email nhận hóa đơn (không bắt buộc)</span><input type="email" value={recipient.email ?? ""} aria-invalid={Boolean(fieldErrors.email)} onChange={(event) => { const value = event.currentTarget.value; setRecipient((current) => ({ ...current, email: value })); setSelectedAddressId(""); }} />{fieldErrors.email ? <small role="alert">{fieldErrors.email}</small> : null}</label>
                <label className="checkout-field"><span>Tỉnh/Thành phố <b>*</b></span><select value={address.provinceCode} aria-invalid={Boolean(fieldErrors.provinceCode)} onChange={(event) => updateProvince(event.currentTarget.value)}><option value="">Chọn Tỉnh/Thành phố</option>{provinceOptions.map((entry) => <option value={entry.provinceCode} key={entry.provinceCode}>{entry.provinceName}</option>)}</select>{fieldErrors.provinceCode ? <small role="alert">{fieldErrors.provinceCode}</small> : null}</label>
                <label className="checkout-field"><span>Phường/Xã <b>*</b></span><select value={address.wardCode} disabled={!address.provinceCode} aria-invalid={Boolean(fieldErrors.wardCode)} onChange={(event) => updateWard(event.currentTarget.value)}><option value="">Chọn Phường/Xã</option>{wards.map((entry) => <option value={entry.wardCode} key={entry.wardCode}>{entry.wardName}</option>)}</select>{fieldErrors.wardCode ? <small role="alert">{fieldErrors.wardCode}</small> : null}</label>
                <label className="checkout-field full"><span>Địa chỉ cụ thể <b>*</b></span><input value={address.addressLine} aria-invalid={Boolean(fieldErrors.addressLine)} placeholder="Số nhà, ngõ, tên đường" onChange={(event) => { const value = event.currentTarget.value; invalidateShippingQuote(); setAddress((current) => ({ ...current, addressLine: value })); setSelectedAddressId(""); }} />{fieldErrors.addressLine ? <small role="alert">{fieldErrors.addressLine}</small> : null}</label>
              </div>
              <p className="checkout-data-note">Thông tin này chỉ dùng để tạo và thực hiện đơn hiện tại.</p>
            </div>
          </section>

          <section className="checkout-panel" aria-labelledby="shipping-title">
            <header><span>2</span><h2 id="shipping-title">Phương thức vận chuyển</h2><small>Phí theo địa chỉ &amp; kiện hàng</small></header>
            <div className="checkout-panel-body checkout-options">
              {quoting ? <div className="checkout-inline-loading" role="status"><i />Đang tính lại phí vận chuyển…</div> : null}
              {!quoting && quoteError ? <div className="checkout-inline-error" role="alert"><strong>Chưa thể lấy phí vận chuyển</strong><p>{quoteError}</p><button type="button" onClick={() => setAddress((current) => ({ ...current }))}>Thử lại</button></div> : null}
              {!quoting && quote && quote.options.length === 0 ? <div className="checkout-inline-error" role="status"><strong>Chưa có phương thức phù hợp</strong><p>Hiện chưa thể giao kiện hàng đến địa chỉ đã chọn. Vui lòng đổi địa chỉ nhận hàng.</p></div> : null}
              {!quoting && !quote && !quoteError ? <p className="checkout-option-placeholder">Hoàn thiện địa chỉ để xem phí và lựa chọn giao hàng.</p> : null}
              {quote?.options.map((option) => <label className={`checkout-option ${shippingOptionId === option.shippingOptionId ? "selected" : ""}`} key={option.shippingOptionId}><input type="radio" name="shipping" value={option.shippingOptionId} checked={shippingOptionId === option.shippingOptionId} onChange={() => setShippingOptionId(option.shippingOptionId)} /><span><strong>{option.name}</strong><small>{option.description} · {option.estimatedDelivery}</small></span><b>{option.calculatedAtCheckout ? "Tính khi đặt" : option.feeVnd === 0 ? "Miễn phí" : VND.format(option.feeVnd)}</b></label>)}
              {fieldErrors.shippingOptionId ? <p className="checkout-field-error" role="alert">{fieldErrors.shippingOptionId}</p> : null}
            </div>
          </section>

          <section className="checkout-panel" aria-labelledby="payment-title">
            <header><span>3</span><h2 id="payment-title">Phương thức thanh toán</h2><small>VietQR hoặc COD</small></header>
            <div className="checkout-panel-body checkout-options">
              <label className={`checkout-option payment ${paymentMethod === "BANK_TRANSFER" ? "selected" : ""}`}><input type="radio" name="payment" checked={paymentMethod === "BANK_TRANSFER"} onChange={() => setPaymentMethod("BANK_TRANSFER")} /><span><strong>Chuyển khoản VietQR</strong><small>Mã QR và thời hạn thanh toán được cấp ngay sau khi tạo đơn.</small></span><b aria-hidden="true">▦</b></label>
              <label className={`checkout-option payment ${paymentMethod === "COD" ? "selected" : ""}`}><input type="radio" name="payment" checked={paymentMethod === "COD"} onChange={() => setPaymentMethod("COD")} /><span><strong>Thanh toán khi nhận hàng (COD)</strong><small>Thanh toán trực tiếp cho đơn vị vận chuyển khi nhận hàng.</small></span><b aria-hidden="true">▣</b></label>
            </div>
          </section>
        </div>

        <aside className="checkout-summary" aria-labelledby="checkout-summary-title">
          <div className="checkout-summary-heading"><div><p className="eyebrow">Kiểm tra lần cuối</p><h2 id="checkout-summary-title">Giỏ hàng Ngự phẩm</h2></div><span>{preparation.itemCount} sản phẩm</span></div>
          <div className="checkout-summary-lines">{preparation.items.map((item, index) => <article key={item.itemId}><div className="checkout-summary-image"><ProductImage src={item.imageUrl} alt={item.imageAlt} fill sizes="64px" loading={index < 2 ? "eager" : "lazy"} /></div><div><strong>{item.productName}</strong><small>{item.skuLabel} · ×{item.quantity}</small>{item.priceChanged ? <em>Giá đã được cập nhật</em> : null}</div><b>{VND.format(item.lineSubtotalVnd)}</b></article>)}</div>
          {preparation.notices.length ? <div className="checkout-price-notice" role="status"><strong>Giá đã thay đổi khi kiểm tra lại</strong>{preparation.notices.map((notice) => <p key={notice.itemId}>{notice.message}</p>)}<label><input type="checkbox" checked={priceAcknowledged} onChange={(event) => setPriceAcknowledged(event.currentTarget.checked)} />Tôi đồng ý tiếp tục với giá hiện tại.</label>{fieldErrors.priceAcknowledgement ? <small role="alert">{fieldErrors.priceAcknowledgement}</small> : null}</div> : null}
          <section className="checkout-voucher" aria-labelledby="checkout-voucher-title"><div><strong id="checkout-voucher-title">Mã ưu đãi</strong><small>Được kiểm tra theo giá trị giỏ hiện tại</small></div><div className="checkout-voucher-entry"><input list="checkout-voucher-options" value={voucherCode} onChange={(event) => { setVoucherCode(event.currentTarget.value.toUpperCase()); setVoucherResult(undefined); setVoucherError(""); setFieldErrors((current) => ({ ...current, voucherCode: "" })); }} placeholder="Nhập mã voucher" /><button type="button" onClick={() => void applyVoucher()} disabled={validatingVoucher}>{validatingVoucher ? "Đang kiểm tra…" : "Áp dụng"}</button><datalist id="checkout-voucher-options">{availableVouchers.map((voucher) => <option key={voucher.voucher_code} value={voucher.voucher_code}>{voucher.title}</option>)}</datalist></div>{voucherResult?.is_valid ? <p className="checkout-voucher-success" role="status">{voucherResult.message} Giảm dự kiến {VND.format(voucherDiscountVnd)}.</p> : null}{voucherError || fieldErrors.voucherCode ? <p className="checkout-field-error" role="alert">{voucherError || fieldErrors.voucherCode}</p> : null}</section>
          <dl className="checkout-totals"><div><dt>Tạm tính</dt><dd>{VND.format(preparation.subtotalVnd)}</dd></div><div><dt>Phí vận chuyển</dt><dd>{selectedShipping?.calculatedAtCheckout ? "Tính khi đặt hàng" : selectedShipping ? (selectedShipping.feeVnd ? VND.format(selectedShipping.feeVnd) : "Miễn phí") : "Chưa xác định"}</dd></div><div><dt>Ưu đãi</dt><dd>{voucherResult?.is_valid ? `−${VND.format(voucherDiscountVnd)}` : "Chưa áp dụng"}</dd></div><div className="grand-total"><dt><strong>{selectedShipping?.calculatedAtCheckout ? "Tạm tính trước phí" : "Tổng thanh toán"}</strong><small>Được chốt lại khi xác nhận</small></dt><dd>{selectedShipping ? VND.format(totalVnd) : "—"}</dd></div></dl>
          {submitError ? <div className="checkout-submit-error" role="alert">{submitError}</div> : null}
          <button className="checkout-confirm-button" type="button" disabled={submitting || quoting || !shippingOptionId} onClick={confirmCheckout}>{submitting ? <><i />Đang kiểm tra &amp; giữ hàng…</> : "Xác nhận đặt hàng →"}</button>
          <p className="checkout-legal">Giá, khả dụng và phí sẽ được server kiểm tra lại. Thao tác lặp dùng cùng khóa để không tạo Order trùng.</p>
          <div className="checkout-trust"><span>✺ Không public tồn kho, kho hàng hoặc Batch/Lot</span><span>⌁ Snapshot đơn không tự đổi theo catalog</span><span>♙ Guest không bắt buộc tạo tài khoản</span></div>
        </aside>
      </div>
    </main>
  );
}
