"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import type { PaymentStatus, PaymentStatusResult, PaymentSummary } from "@/lib/payment/types";
import { PaymentApiError } from "@/lib/payment/types";
import { paymentService } from "@/services/payment-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

const STATUS_COPY: Record<PaymentStatus, { title: string; tone: string }> = {
  PENDING: { title: "Đang chờ xác minh thanh toán", tone: "pending" },
  SUCCEEDED: { title: "Thanh toán đã được xác minh", tone: "success" },
  FAILED: { title: "Lần thử thanh toán không thành công", tone: "error" },
  EXPIRED: { title: "Thời hạn thanh toán đã kết thúc", tone: "error" },
  REQUIRES_RECONCILIATION: { title: "Giao dịch đang chờ đối soát", tone: "review" },
  COD_PENDING_COLLECTION: { title: "Đơn COD đã được tiếp nhận", tone: "cod" }
};

function formatCountdown(expiresAt: string, now: number) {
  const seconds = Math.max(0, Math.floor((Date.parse(expiresAt) - now) / 1000));
  return { expired: seconds === 0, text: `${String(Math.floor(seconds / 60)).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}` };
}

function PaymentSteps() {
  return <ol className="payment-steps" aria-label="Tiến trình mua hàng"><li className="done"><span>✓</span><div><strong>Giỏ hàng Tri Kỷ</strong><small>Đã kiểm tra</small></div></li><li className="done"><span>✓</span><div><strong>Thông tin &amp; Checkout</strong><small>Đã tạo Order</small></div></li><li className="active" aria-current="step"><span>3</span><div><strong>Xác nhận &amp; thanh toán</strong><small>Đang chờ đối soát</small></div></li></ol>;
}

function DemoQr() {
  return <div className="payment-demo-qr" role="img" aria-label="Minh họa QR demo, không thể dùng để chuyển tiền"><i className="finder one" /><i className="finder two" /><i className="finder three" /><span>Ô MẠ<br />DEMO</span></div>;
}

export function PaymentFlow({ orderId }: { orderId: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [summary, setSummary] = useState<PaymentSummary | null>(null);
  const [statusResult, setStatusResult] = useState<PaymentStatusResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [checking, setChecking] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [error, setError] = useState("");
  const [copyMessage, setCopyMessage] = useState("");
  const [now, setNow] = useState(() => Date.now());
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let active = true;
    Promise.resolve().then(() => {
      if (!active) return Promise.reject(new Error("cancelled"));
      setLoading(true);
      setError("");
      return paymentService.getSummary(orderId, scenario);
    }).then((payload) => { if (active) setSummary(payload); }).catch((cause: unknown) => {
      if (!active) return;
      setError(cause instanceof PaymentApiError ? cause.message : "Không thể tải thông tin thanh toán.");
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [orderId, scenario, reloadKey]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const activeStatus = statusResult?.status ?? summary?.status ?? "PENDING";
  const statusCopy = STATUS_COPY[activeStatus];
  const countdown = useMemo(() => summary ? formatCountdown(summary.expiresAt, now) : { expired: false, text: "--:--" }, [summary, now]);

  async function copyValue(value: string, label: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopyMessage(`Đã sao chép ${label}.`);
    } catch {
      setCopyMessage(`Không thể tự động sao chép ${label}.`);
    }
    window.setTimeout(() => setCopyMessage(""), 2500);
  }

  async function checkStatus() {
    setChecking(true);
    setError("");
    try {
      const result = await paymentService.getStatus(orderId, scenario);
      setStatusResult(result);
      setSummary((current) => current ? { ...current, status: result.status, orderPaymentState: result.orderPaymentState, checkedAt: new Date().toISOString() } : current);
    } catch (cause) {
      setError(cause instanceof PaymentApiError ? cause.message : "Chưa thể kiểm tra trạng thái Payment.");
    } finally { setChecking(false); }
  }

  async function retryPayment() {
    setRetrying(true);
    setError("");
    try {
      const result = await paymentService.retry(orderId, `payment-retry-${globalThis.crypto.randomUUID()}`, scenario);
      setSummary((current) => current ? { ...current, paymentId: result.paymentId, status: result.status, orderPaymentState: result.orderPaymentState, expiresAt: result.expiresAt } : current);
      setStatusResult({ paymentId: result.paymentId, status: result.status, orderPaymentState: result.orderPaymentState, verifiedAt: null, message: result.message, canRetry: false });
      setNow(Date.now());
    } catch (cause) {
      setError(cause instanceof PaymentApiError ? cause.message : "Không thể tạo lần thử Payment mới.");
    } finally { setRetrying(false); }
  }

  if (loading) return <main className="payment-page"><PaymentSteps /><div className="payment-loading" aria-busy="true"><span className="large-spinner" /><p>Đang mở thông tin thanh toán an toàn…</p></div></main>;

  if (!summary || error && !summary) return <main className="payment-page"><PaymentSteps /><section className="payment-state error" role="alert"><span>!</span><p className="eyebrow">Không thể mở Payment</p><h1>Phiên truy cập không còn khả dụng</h1><p>{error || "Thông tin thanh toán không tồn tại."}</p><div><button className="primary-button" type="button" onClick={() => setReloadKey((value) => value + 1)}>Thử lại</button><Link className="secondary-link" href="/cart">Quay lại giỏ hàng</Link></div></section></main>;

  const instructions = summary.instructions;
  const canRetry = summary.method === "BANK_TRANSFER" && (statusResult?.canRetry || activeStatus === "EXPIRED" || countdown.expired);

  return (
    <main className="payment-page">
      <PaymentSteps />
      <section className={`payment-status-banner ${statusCopy.tone}`}>
        <div className="payment-status-symbol" aria-hidden="true">{activeStatus === "SUCCEEDED" ? "✓" : activeStatus === "COD_PENDING_COLLECTION" ? "▣" : activeStatus === "REQUIRES_RECONCILIATION" ? "!" : "◔"}</div>
        <div><span className="eyebrow">Order {summary.orderNumber}</span><h1>{statusCopy.title}</h1><p>{statusResult?.message ?? (summary.method === "COD" ? "Thanh toán sẽ được thu khi giao hàng thành công." : "Chỉ xác nhận thành công khi backend nhận bằng chứng hợp lệ từ nguồn tin cậy.")}</p></div>
        <div className="payment-countdown"><small>{countdown.expired ? "Thời hạn đã kết thúc" : "Thời gian còn lại của lần thử"}</small><strong>{countdown.text}</strong><span>phút : giây</span></div>
      </section>

      {summary.method === "BANK_TRANSFER" && instructions ? <div className="payment-grid">
        <section className="payment-qr-card" aria-labelledby="qr-title"><div className="payment-card-title"><div><span aria-hidden="true">⌗</span><h2 id="qr-title">Mã QR thanh toán demo</h2></div><b>Không thể quét trả tiền</b></div><div className="payment-qr-stage"><DemoQr /><p>Minh họa luồng VietQR trong môi trường local. Không chứa payload ngân hàng thật.</p></div><div className="payment-demo-warning"><span>!</span><p><strong>Không chuyển tiền thật.</strong> Tài khoản, QR và nội dung bên cạnh chỉ là fixture dùng kiểm tra UI.</p></div></section>
        <section className="payment-instruction-card" aria-labelledby="instruction-title"><div className="payment-card-title"><div><span aria-hidden="true">▤</span><h2 id="instruction-title">Thông tin chuyển khoản mô phỏng</h2></div><b>DEMO ONLY</b></div><dl className="payment-instructions"><div><dt>Đơn vị mô phỏng</dt><dd>{instructions.providerLabel}</dd></div><div><dt>Chủ tài khoản demo</dt><dd>{instructions.beneficiary}</dd></div><div><dt>Số tài khoản giả</dt><dd><strong>{instructions.accountNumberMasked}</strong><button type="button" onClick={() => copyValue(instructions.accountNumberMasked, "số tài khoản demo")}>Sao chép</button></dd></div><div className="important"><dt>Số tiền theo Order</dt><dd><strong>{VND.format(summary.amountVnd)}</strong><button type="button" onClick={() => copyValue(String(summary.amountVnd), "số tiền")}>Chép số tiền</button></dd></div><div className="reference"><dt>Nội dung tham chiếu demo</dt><dd><strong>{instructions.transferReference}</strong><button type="button" onClick={() => copyValue(instructions.transferReference, "mã tham chiếu")}>Chép mã</button></dd></div></dl><p className="payment-verification-note">Trang quay lại hoặc nút kiểm tra không tạo kết quả thành công. Payment chỉ đổi sang đã xác minh khi API trả trạng thái tin cậy.</p></section>
      </div> : null}

      {summary.method === "COD" ? <section className="payment-cod-card"><span aria-hidden="true">▣</span><div><p className="eyebrow">Thanh toán khi nhận hàng</p><h2>Đơn đã ghi nhận nghĩa vụ thu COD</h2><p>Bạn chưa thanh toán ở thời điểm này. Đơn không được đánh dấu <code>PAID</code>; khoản thu chỉ được ghi nhận sau xác nhận giao hàng/thu tiền hợp lệ.</p><dl><div><dt>Số tiền cần thu</dt><dd>{VND.format(summary.amountVnd)}</dd></div><div><dt>Trạng thái Payment</dt><dd>Chờ thu khi giao hàng</dd></div></dl></div></section> : null}

      <section className="payment-summary-card"><div><p className="eyebrow">Tóm tắt Payment</p><h2>{summary.orderNumber}</h2></div><dl><div><dt>Phương thức</dt><dd>{summary.method === "COD" ? "COD" : "Chuyển khoản/QR"}</dd></div><div><dt>Số tiền phải thu</dt><dd>{VND.format(summary.amountVnd)}</dd></div><div><dt>Payment ID</dt><dd>{summary.paymentId}</dd></div><div><dt>Trạng thái Order/Payment</dt><dd>{summary.orderPaymentState}</dd></div></dl></section>

      {error ? <div className="payment-action-error" role="alert">{error}</div> : null}
      {copyMessage ? <div className="payment-toast" role="status">{copyMessage}</div> : null}
      <section className="payment-actions"><div><button className="primary-button" type="button" disabled={checking} onClick={checkStatus}>{checking ? <><span className="spinner" />Đang kiểm tra…</> : "Kiểm tra trạng thái thanh toán"}</button><Link className="secondary-link" href={`/orders/${encodeURIComponent(orderId)}/confirmation`}>Xem xác nhận Order</Link>{canRetry ? <button className="secondary-button" type="button" disabled={retrying} onClick={retryPayment}>{retrying ? "Đang tạo…" : "Tạo lần thử mới"}</button> : null}</div><p><strong>Không thấy trạng thái mới?</strong> Việc xác minh có thể cần thêm thời gian. Đừng thanh toán lặp khi giao dịch cũ chưa rõ kết quả.</p><Link href="/products">Tiếp tục khám phá sản phẩm</Link></section>
    </main>
  );
}
