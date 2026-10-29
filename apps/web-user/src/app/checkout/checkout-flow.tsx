"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { ProductImage } from "@/components/product/product-image";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { createIdempotencyKey, PublicApiError } from "@/lib/api-contract";
import type {
  CheckoutRecipient,
  MvpCheckoutOperationView,
  MvpCheckoutPaymentMethod,
  MvpCheckoutPreparation,
  MvpCheckoutQuote,
  SavedCheckoutAddress,
  ShippingAddress
} from "@/lib/checkout/types";
import { getCheckoutFormErrors, validateShippingAddress } from "@/lib/checkout/validation";
import { checkoutService } from "@/services/checkout-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const EMPTY_RECIPIENT: CheckoutRecipient = { fullName: "", phone: "", email: "" };
const EMPTY_ADDRESS: ShippingAddress = { provinceCode: "", provinceName: "", wardCode: "", wardName: "", addressLine: "" };
const VOUCHER_PATTERN = /^[A-Z0-9_-]{3,50}$/u;

function CheckoutSteps({ complete = false }: { complete?: boolean }) {
  return (
    <ol className="checkout-steps" aria-label="Tiến trình mua hàng">
      <li className="done"><span aria-hidden="true">✓</span><div><small>Bước 1</small><strong>Giỏ hàng</strong></div></li>
      <li className={complete ? "done" : "active"} aria-current={complete ? undefined : "step"}><span aria-hidden="true">{complete ? "✓" : "2"}</span><div><small>Bước 2</small><strong>Quote &amp; xác nhận</strong></div></li>
      <li className={complete ? "active" : ""} aria-current={complete ? "step" : undefined}><span aria-hidden="true">3</span><div><small>Bước 3</small><strong>Order được tạo</strong></div></li>
    </ol>
  );
}

function CheckoutLoading() {
  return <div className="checkout-loading" aria-busy="true" aria-label="Đang kiểm tra giỏ hàng"><div /><div /></div>;
}

function optionLabel(address: SavedCheckoutAddress) {
  return `${address.label} · ${address.address.addressLine}, ${address.address.wardName}`;
}

function detailsErrors(error: PublicApiError): Record<string, string> {
  const source = error.details?.errors;
  if (!Array.isArray(source)) return {};
  const fieldNames: Record<string, string> = {
    recipient_name: "fullName",
    phone: "phone",
    street: "addressLine",
    province_code: "provinceCode",
    ward_code: "wardCode",
    voucher_code: "voucherCode"
  };
  return Object.fromEntries(source.flatMap((entry) => {
    if (!entry || typeof entry !== "object") return [];
    const item = entry as Record<string, unknown>;
    return typeof item.field === "string" && typeof item.message === "string"
      ? [[fieldNames[item.field] ?? item.field, item.message]]
      : [];
  }));
}

export function CheckoutFlow() {
  const { reload: reloadCartSummary } = useCartSummary();
  const searchParams = useSearchParams();
  const itemIds = useMemo(() => (searchParams.get("items") ?? "").split(",").filter(Boolean), [searchParams]);
  const mockScenario = searchParams.get("mockScenario") ?? undefined;
  const cartScenario = searchParams.get("cartScenario") ?? undefined;
  const requestOptions = useMemo(() => ({ mockScenario, cartScenario }), [mockScenario, cartScenario]);
  const commandRef = useRef<{ quoteId: string; key: string } | undefined>(undefined);

  const [preparation, setPreparation] = useState<MvpCheckoutPreparation | null>(null);
  const [recipient, setRecipient] = useState<CheckoutRecipient>(EMPTY_RECIPIENT);
  const [address, setAddress] = useState<ShippingAddress>(EMPTY_ADDRESS);
  const [selectedAddressId, setSelectedAddressId] = useState("");
  const [paymentMethod, setPaymentMethod] = useState<MvpCheckoutPaymentMethod>("VIETQR");
  const [voucherCode, setVoucherCode] = useState("");
  const [quote, setQuote] = useState<MvpCheckoutQuote | null>(null);
  const [operation, setOperation] = useState<MvpCheckoutOperationView | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [pageError, setPageError] = useState("");
  const [quoteError, setQuoteError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [loading, setLoading] = useState(itemIds.length > 0);
  const [quoting, setQuoting] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [retryKey, setRetryKey] = useState(0);

  useEffect(() => {
    if (!itemIds.length) return;
    let active = true;
    setLoading(true);
    setPageError("");
    checkoutService.prepare(itemIds, requestOptions).then((payload) => {
      if (!active) return;
      setPreparation(payload);
      const preferred = payload.savedAddresses.find((saved) => saved.isDefault) ?? payload.savedAddresses[0];
      if (preferred) {
        setSelectedAddressId(preferred.addressId);
        setRecipient(preferred.recipient);
        setAddress(preferred.address);
      } else if (payload.defaultRecipient) {
        setRecipient(payload.defaultRecipient);
      }
    }).catch((cause: unknown) => {
      if (active) setPageError(cause instanceof Error ? cause.message : "Không thể chuẩn bị Checkout lúc này.");
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [itemIds, requestOptions, retryKey]);

  useEffect(() => {
    if (!preparation || validateShippingAddress(address).length > 0 || !recipient.fullName.trim() || !recipient.phone.trim()) return;
    const normalizedVoucher = voucherCode.trim().toUpperCase();
    if (normalizedVoucher && !VOUCHER_PATTERN.test(normalizedVoucher)) {
      setQuote(null);
      setQuoteError("Mã ưu đãi cần từ 3 đến 50 ký tự hợp lệ.");
      return;
    }
    let active = true;
    const timer = globalThis.setTimeout(() => {
      setQuoting(true);
      setQuoteError("");
      checkoutService.createQuote({
        cartRevision: preparation.cartRevision,
        paymentMethod,
        selectedAddressId: selectedAddressId || undefined,
        recipient,
        address,
        voucherCode: normalizedVoucher || undefined
      }, requestOptions).then((result) => {
        if (!active) return;
        setQuote(result);
        commandRef.current = undefined;
      }).catch((cause: unknown) => {
        if (!active) return;
        setQuote(null);
        setQuoteError(cause instanceof PublicApiError ? cause.message : "Chưa thể tạo Checkout quote.");
      }).finally(() => {
        if (active) setQuoting(false);
      });
    }, 450);
    return () => {
      active = false;
      globalThis.clearTimeout(timer);
    };
  }, [address, paymentMethod, preparation, recipient, requestOptions, selectedAddressId, voucherCode]);

  function applySavedAddress(addressId: string) {
    setSelectedAddressId(addressId);
    const saved = preparation?.savedAddresses.find((entry) => entry.addressId === addressId);
    if (!saved) return;
    setRecipient(saved.recipient);
    setAddress(saved.address);
    setFieldErrors({});
    setQuote(null);
  }

  function updateRecipient<K extends keyof CheckoutRecipient>(key: K, value: CheckoutRecipient[K]) {
    setSelectedAddressId("");
    setRecipient((current) => ({ ...current, [key]: value }));
    setFieldErrors((current) => ({ ...current, [key]: "" }));
    setQuote(null);
  }

  function updateProvince(provinceCode: string) {
    const province = preparation?.locations.find((entry) => entry.provinceCode === provinceCode);
    setSelectedAddressId("");
    setAddress((current) => ({ ...current, provinceCode, provinceName: province?.provinceName ?? "", wardCode: "", wardName: "" }));
    setQuote(null);
  }

  function updateWard(wardCode: string) {
    const province = preparation?.locations.find((entry) => entry.provinceCode === address.provinceCode);
    const ward = province?.wards.find((entry) => entry.wardCode === wardCode);
    setSelectedAddressId("");
    setAddress((current) => ({ ...current, wardCode, wardName: ward?.wardName ?? "" }));
    setQuote(null);
  }

  async function continuePolling(current: MvpCheckoutOperationView, retryAfterSeconds = 1) {
    setSubmitting(true);
    setSubmitError("");
    try {
      const result = await checkoutService.waitForOperation(current, retryAfterSeconds, requestOptions, setOperation);
      setOperation(result);
      if (result.succeeded) {
        void reloadCartSummary();
      } else if (result.terminal) {
        setSubmitError(result.status === "MANUAL_REVIEW"
          ? "Checkout cần được nhân viên kiểm tra thủ công. Không gửi lại lệnh với khóa mới."
          : `Checkout thất bại${result.errorCode ? `: ${result.errorCode}` : "."}`);
      } else {
        setSubmitError("Checkout vẫn đang được xử lý. Bạn có thể kiểm tra lại operation này.");
      }
    } catch (cause) {
      setSubmitError(cause instanceof PublicApiError ? cause.message : "Chưa thể đọc trạng thái Checkout operation.");
    } finally {
      setSubmitting(false);
    }
  }

  async function confirmCheckout() {
    if (!preparation || !quote) return;
    const errors = getCheckoutFormErrors(recipient, address);
    if (voucherCode.trim() && !VOUCHER_PATTERN.test(voucherCode.trim().toUpperCase())) errors.voucherCode = "Mã ưu đãi không hợp lệ.";
    setFieldErrors(errors);
    setSubmitError("");
    if (Object.keys(errors).length) return;
    if (Date.parse(quote.expiresAt) <= Date.now() || Date.parse(quote.shippingExpiresAt) <= Date.now()) {
      setQuote(null);
      setQuoteError("Quote đã hết hạn. Hệ thống đang tạo quote mới.");
      return;
    }

    setSubmitting(true);
    try {
      if (!commandRef.current || commandRef.current.quoteId !== quote.quoteId) {
        commandRef.current = { quoteId: quote.quoteId, key: createIdempotencyKey("checkout") };
      }
      const accepted = await checkoutService.accept(quote, commandRef.current.key, requestOptions);
      setOperation(accepted.operation);
      await continuePolling(accepted.operation, accepted.retryAfterSeconds);
    } catch (cause) {
      if (cause instanceof PublicApiError) {
        setSubmitError(cause.status === 409
          ? "Cart hoặc quote đã thay đổi. Hãy tạo lại quote trước khi thử lại."
          : cause.message);
        setFieldErrors((current) => ({ ...current, ...detailsErrors(cause) }));
        if ([409, 410, 422].includes(cause.status)) setQuote(null);
      } else {
        setSubmitError("Không thể accept Checkout lúc này. Nếu kết quả chưa rõ, hãy dùng lại cùng operation/khóa hiện tại.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (!itemIds.length) {
    return <main className="checkout-page"><CheckoutSteps /><section className="checkout-state-card"><span aria-hidden="true">◇</span><p className="eyebrow">Chưa có lựa chọn Checkout</p><h1>Hãy chọn sản phẩm từ giỏ hàng</h1><p>Checkout chỉ xử lý những dòng bạn chủ động chọn trong giỏ.</p><Link className="primary-link" href="/cart">Quay lại giỏ hàng</Link></section></main>;
  }
  if (loading) return <main className="checkout-page"><CheckoutSteps /><CheckoutLoading /></main>;
  if (pageError || !preparation) {
    return <main className="checkout-page"><CheckoutSteps /><section className="checkout-state-card error" role="alert"><span aria-hidden="true">!</span><p className="eyebrow">Checkout cần được làm mới</p><h1>Chưa thể kiểm tra giỏ hàng</h1><p>{pageError || "Giỏ hàng đã thay đổi."}</p><div><button className="primary-button" type="button" onClick={() => setRetryKey((value) => value + 1)}>Thử tải lại</button><Link className="secondary-link" href="/cart">Về giỏ hàng</Link></div></section></main>;
  }

  if (operation?.succeeded && operation.orderId) {
    const paymentPath = paymentMethod === "VIETQR" ? `/payment/${encodeURIComponent(operation.orderId)}` : `/orders/${encodeURIComponent(operation.orderId)}/confirmation`;
    return <main className="checkout-page"><CheckoutSteps complete /><section className="checkout-confirmed"><span className="checkout-confirmed-icon" aria-hidden="true">✓</span><p className="eyebrow">Checkout operation thành công</p><h1>Order đã được tạo an toàn</h1><p>Operation <code>{operation.operationId}</code> đã hoàn tất và trả về Order <code>{operation.orderId}</code>.</p><dl><div><dt>Trạng thái operation</dt><dd>{operation.status}</dd></div><div><dt>Trạng thái Order</dt><dd>{operation.orderStatus ?? "Đang đồng bộ"}</dd></div><div><dt>Cập nhật lúc</dt><dd>{new Date(operation.updatedAt).toLocaleString("vi-VN")}</dd></div></dl><div className="checkout-confirmed-actions"><Link className="primary-link" href={paymentPath}>{paymentMethod === "VIETQR" ? "Mở trang thanh toán" : "Xem đơn hàng"}</Link><Link className="secondary-link" href="/products">Tiếp tục mua sắm</Link></div></section></main>;
  }

  const selectedProvince = preparation.locations.find((entry) => entry.provinceCode === address.provinceCode);
  const quoteExpired = quote ? Date.parse(quote.expiresAt) <= Date.now() || Date.parse(quote.shippingExpiresAt) <= Date.now() : false;

  return (
    <main className="checkout-page">
      <CheckoutSteps />
      <div className="checkout-guest-banner"><span aria-hidden="true">♢</span><div><strong>{preparation.customerMode === "REGISTERED" ? "Checkout dùng danh tính tài khoản đã xác minh." : "Guest Checkout chưa được bật trong contract target."}</strong><p>Giá, tồn kho và phí vận chuyển chỉ được chốt bởi quote từ server.</p></div></div>
      {preparation.accountDataWarning ? <div className="checkout-inline-error" role="status"><strong>Thông tin tài khoản chưa tải đầy đủ</strong><p>{preparation.accountDataWarning}</p></div> : null}

      <div className="checkout-layout">
        <div className="checkout-main-column">
          <section className="checkout-panel" aria-labelledby="recipient-title">
            <header><span>1</span><h1 id="recipient-title">Thông tin người nhận &amp; giao hàng</h1><small>Revision {preparation.cartRevision}</small></header>
            <div className="checkout-panel-body">
              {preparation.savedAddresses.length ? <label className="checkout-field full"><span>Địa chỉ đã lưu</span><select value={selectedAddressId} onChange={(event) => applySavedAddress(event.currentTarget.value)}><option value="">Nhập thông tin mới</option>{preparation.savedAddresses.map((saved) => <option value={saved.addressId} key={saved.addressId}>{optionLabel(saved)}</option>)}</select></label> : null}
              <div className="checkout-form-grid">
                <label className="checkout-field"><span>Họ và tên người nhận <b>*</b></span><input value={recipient.fullName} onChange={(event) => updateRecipient("fullName", event.currentTarget.value)} />{fieldErrors.fullName ? <small role="alert">{fieldErrors.fullName}</small> : null}</label>
                <label className="checkout-field"><span>Số điện thoại <b>*</b></span><input inputMode="tel" value={recipient.phone} onChange={(event) => updateRecipient("phone", event.currentTarget.value)} />{fieldErrors.phone ? <small role="alert">{fieldErrors.phone}</small> : null}</label>
                <label className="checkout-field full"><span>Email nhận thông báo</span><input type="email" value={recipient.email ?? ""} onChange={(event) => updateRecipient("email", event.currentTarget.value)} />{fieldErrors.email ? <small role="alert">{fieldErrors.email}</small> : null}</label>
                <label className="checkout-field"><span>Tỉnh/Thành phố <b>*</b></span><select value={address.provinceCode} onChange={(event) => updateProvince(event.currentTarget.value)}><option value="">Chọn Tỉnh/Thành phố</option>{preparation.locations.map((entry) => <option value={entry.provinceCode} key={entry.provinceCode}>{entry.provinceName}</option>)}</select>{fieldErrors.provinceCode ? <small role="alert">{fieldErrors.provinceCode}</small> : null}</label>
                <label className="checkout-field"><span>Phường/Xã <b>*</b></span><select value={address.wardCode} disabled={!selectedProvince} onChange={(event) => updateWard(event.currentTarget.value)}><option value="">Chọn Phường/Xã</option>{selectedProvince?.wards.map((entry) => <option value={entry.wardCode} key={entry.wardCode}>{entry.wardName}</option>)}</select>{fieldErrors.wardCode ? <small role="alert">{fieldErrors.wardCode}</small> : null}</label>
                <label className="checkout-field full"><span>Địa chỉ cụ thể <b>*</b></span><input value={address.addressLine} onChange={(event) => { setSelectedAddressId(""); setAddress((current) => ({ ...current, addressLine: event.currentTarget.value })); setQuote(null); }} />{fieldErrors.addressLine ? <small role="alert">{fieldErrors.addressLine}</small> : null}</label>
              </div>
            </div>
          </section>

          <section className="checkout-panel" aria-labelledby="payment-title">
            <header><span>2</span><h2 id="payment-title">Phương thức thanh toán</h2><small>Quote phụ thuộc phương thức</small></header>
            <div className="checkout-options">
              <label className={`checkout-option payment ${paymentMethod === "VIETQR" ? "selected" : ""}`}><input type="radio" name="payment" checked={paymentMethod === "VIETQR"} onChange={() => { setPaymentMethod("VIETQR"); setQuote(null); }} /><span><strong>Chuyển khoản VietQR</strong><small>Payment instructions được đọc sau khi Order tạo thành công.</small></span></label>
              <label className={`checkout-option payment ${paymentMethod === "COD" ? "selected" : ""}`}><input type="radio" name="payment" checked={paymentMethod === "COD"} onChange={() => { setPaymentMethod("COD"); setQuote(null); }} /><span><strong>Thanh toán khi nhận hàng (COD)</strong><small>Khả dụng thực tế vẫn do backend quyết định.</small></span></label>
            </div>
          </section>
        </div>

        <aside className="checkout-summary" aria-labelledby="checkout-summary-title">
          <div className="checkout-summary-heading"><div><p className="eyebrow">Quote server</p><h2 id="checkout-summary-title">Xác nhận thương mại</h2></div><span>{preparation.itemCount} sản phẩm</span></div>
          <div className="checkout-summary-lines">{preparation.items.map((item, index) => <article key={item.itemId}><div className="checkout-summary-image"><ProductImage src={item.imageUrl} alt={item.imageAlt} fill sizes="64px" loading={index < 2 ? "eager" : "lazy"} /></div><div><strong>{item.productName}</strong><small>{item.skuLabel} · ×{item.quantity}</small></div><b>{preparation.pricingComplete ? VND.format(item.lineSubtotalVnd) : "Chờ quote"}</b></article>)}</div>
          <section className="checkout-voucher"><div><strong>Mã ưu đãi</strong><small>Được backend xét khi tạo quote</small></div><div className="checkout-voucher-entry"><input value={voucherCode} onChange={(event) => { setVoucherCode(event.currentTarget.value.toUpperCase()); setQuote(null); setFieldErrors((current) => ({ ...current, voucherCode: "" })); }} placeholder="Nhập mã voucher" /></div>{fieldErrors.voucherCode ? <p className="checkout-field-error" role="alert">{fieldErrors.voucherCode}</p> : null}</section>
          <dl className="checkout-totals"><div><dt>Tạm tính</dt><dd>{quote ? VND.format(quote.subtotalVnd) : preparation.pricingComplete ? VND.format(preparation.subtotalVnd) : "Chờ quote"}</dd></div><div><dt>Phí vận chuyển</dt><dd>{quote ? VND.format(quote.shippingFeeVnd) : "Chưa có quote"}</dd></div><div className="grand-total"><dt><strong>Tổng thanh toán</strong><small>{quote ? `Quote hết hạn ${new Date(quote.expiresAt).toLocaleString("vi-VN")}` : "Được server chốt"}</small></dt><dd>{quote ? VND.format(quote.totalVnd) : "—"}</dd></div></dl>
          {quoting ? <p className="cart-checkout-note" role="status">Đang tạo quote từ Cart revision {preparation.cartRevision}…</p> : null}
          {quoteError ? <div className="checkout-submit-error" role="alert">{quoteError}</div> : null}
          {operation ? <div className="checkout-payment-boundary" role="status"><strong>Operation: {operation.status}</strong><p>{operation.operationId} · stage {operation.stage}</p>{!operation.terminal && !submitting ? <button type="button" className="secondary-button" onClick={() => void continuePolling(operation)}>Kiểm tra lại</button> : null}</div> : null}
          {submitError ? <div className="checkout-submit-error" role="alert">{submitError}</div> : null}
          <button className="checkout-confirm-button" type="button" disabled={submitting || quoting || !quote || quoteExpired} onClick={() => void confirmCheckout()}>{submitting ? <><i />Đang xử lý operation…</> : quoteExpired ? "Quote đã hết hạn" : quote ? "Accept Checkout →" : "Chờ Checkout quote"}</button>
          <p className="checkout-legal">Accept chỉ gửi quote ID, cart revision và payment method. Mutation dùng cùng Idempotency-Key khi kết quả chưa rõ.</p>
        </aside>
      </div>
    </main>
  );
}
