"use client";

import Image from "next/image";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { PublicApiError } from "@/lib/api-contract";
import type { PaymentInstructionsView, PaymentStatusView } from "@/lib/payment/types";
import { paymentService } from "@/services/payment-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

function formatCountdown(expiresAt: string, now: number) {
  const seconds = Math.max(0, Math.floor((Date.parse(expiresAt) - now) / 1000));
  return {
    expired: seconds === 0,
    text: `${String(Math.floor(seconds / 60)).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`
  };
}

function PaymentSteps() {
  return <ol className="payment-steps" aria-label="Tiến trình mua hàng"><li className="done"><span>✓</span><div><strong>Giỏ hàng</strong><small>Đã kiểm tra</small></div></li><li className="done"><span>✓</span><div><strong>Checkout</strong><small>Order đã tạo</small></div></li><li className="active" aria-current="step"><span>3</span><div><strong>Thanh toán</strong><small>Trạng thái từ backend</small></div></li></ol>;
}

export function PaymentFlow({ orderId }: { orderId: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [status, setStatus] = useState<PaymentStatusView | null>(null);
  const [instructions, setInstructions] = useState<PaymentInstructionsView | null>(null);
  const [loading, setLoading] = useState(true);
  const [checking, setChecking] = useState(false);
  const [pageError, setPageError] = useState("");
  const [instructionsError, setInstructionsError] = useState("");
  const [copyMessage, setCopyMessage] = useState("");
  const [clockOffset, setClockOffset] = useState(0);
  const [now, setNow] = useState(() => Date.now());
  const [reloadKey, setReloadKey] = useState(0);

  const load = useCallback(async (initial = false) => {
    if (initial) setLoading(true);
    setPageError("");
    try {
      const nextStatus = await paymentService.getStatus(orderId, scenario);
      setStatus(nextStatus);
      setClockOffset(Date.parse(nextStatus.serverTime) - Date.now());
      if (nextStatus.canPay) {
        setInstructionsError("");
        try {
          const nextInstructions = await paymentService.getInstructions(orderId, scenario);
          if (nextInstructions.intentId !== nextStatus.intentId || nextInstructions.orderId !== nextStatus.orderId) {
            throw new Error("Payment instructions do not belong to the current intent.");
          }
          setInstructions(nextInstructions);
        } catch (cause) {
          setInstructions(null);
          setInstructionsError(cause instanceof PublicApiError ? cause.message : "Chưa thể đọc VietQR instructions hiện tại.");
        }
      } else {
        setInstructions(null);
        setInstructionsError("");
      }
    } catch (cause) {
      setPageError(cause instanceof PublicApiError ? cause.message : "Không thể tải trạng thái Payment.");
      if (initial) setStatus(null);
    } finally {
      if (initial) setLoading(false);
    }
  }, [orderId, scenario]);

  useEffect(() => { void Promise.resolve().then(() => load(true)); }, [load, reloadKey]);

  useEffect(() => {
    if (!status || status.nextPollAfterSeconds <= 0) return;
    const delay = Math.min(60, Math.max(1, status.nextPollAfterSeconds)) * 1000;
    const timer = globalThis.setTimeout(() => void load(false), delay);
    return () => globalThis.clearTimeout(timer);
  }, [load, status]);

  useEffect(() => {
    const timer = globalThis.setInterval(() => setNow(Date.now()), 1000);
    return () => globalThis.clearInterval(timer);
  }, []);

  const countdown = useMemo(() => status
    ? formatCountdown(status.expiresAt, now + clockOffset)
    : { expired: false, text: "--:--" }, [clockOffset, now, status]);

  async function copyValue(value: string, label: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopyMessage(`Đã sao chép ${label}.`);
    } catch {
      setCopyMessage(`Không thể tự động sao chép ${label}.`);
    }
    globalThis.setTimeout(() => setCopyMessage(""), 2500);
  }

  async function checkStatus() {
    setChecking(true);
    await load(false);
    setChecking(false);
  }

  if (loading) {
    return <main className="payment-page"><PaymentSteps /><div className="payment-loading" aria-busy="true"><span className="large-spinner" /><p>Đang đọc Payment projection an toàn…</p></div></main>;
  }

  if (!status) {
    return <main className="payment-page"><PaymentSteps /><section className="payment-state error" role="alert"><span>!</span><p className="eyebrow">Không thể mở Payment</p><h1>Chưa có trạng thái thanh toán khả dụng</h1><p>{pageError || "Payment không tồn tại, không thuộc tài khoản hoặc phương thức chưa được hỗ trợ."}</p><div><button className="primary-button" type="button" onClick={() => setReloadKey((value) => value + 1)}>Thử lại</button><Link className="secondary-link" href={`/orders/${encodeURIComponent(orderId)}`}>Xem Order</Link></div></section></main>;
  }

  const statusSymbol = status.displayState === "PAID" || status.displayState === "REFUNDED"
    ? "✓"
    : status.tone === "review"
      ? "!"
      : status.tone === "error"
        ? "×"
        : "◔";

  return <main className="payment-page">
    <PaymentSteps />
    <section className={`payment-status-banner ${status.tone}`}>
      <div className="payment-status-symbol" aria-hidden="true">{statusSymbol}</div>
      <div><span className="eyebrow">Order {status.orderId}</span><h1>{status.displayLabel}</h1><p>{status.displayMessage}</p></div>
      <div className="payment-countdown"><small>{countdown.expired ? "Thời hạn Payment đã kết thúc" : "Thời gian đến expiry canonical"}</small><strong>{countdown.text}</strong><span>phút : giây</span></div>
    </section>

    {instructions ? <div className="payment-grid">
      <section className="payment-qr-card" aria-labelledby="qr-title"><div className="payment-card-title"><div><span aria-hidden="true">⌗</span><h2 id="qr-title">VietQR hiện hành</h2></div><b>{instructions.provider}</b></div><div className="payment-qr-stage"><Image src={instructions.qrUrl} alt={`Mã VietQR cho Order ${instructions.orderId}`} width={300} height={300} unoptimized /><p>Chỉ quét QR khi trạng thái vẫn cho phép thanh toán và chưa qua thời hạn.</p></div></section>
      <section className="payment-instruction-card" aria-labelledby="instruction-title"><div className="payment-card-title"><div><span aria-hidden="true">▤</span><h2 id="instruction-title">Thông tin chuyển khoản</h2></div><b>VERSION {instructions.version}</b></div><dl className="payment-instructions">
        <div className="important"><dt>Số tiền chính xác</dt><dd><strong>{VND.format(instructions.amountVnd)}</strong><button type="button" onClick={() => copyValue(String(instructions.amountVnd), "số tiền")}>Chép số tiền</button></dd></div>
        <div><dt>Ngân hàng</dt><dd>{instructions.receiver.bankName} ({instructions.receiver.bankCode})</dd></div>
        <div><dt>Chủ tài khoản</dt><dd>{instructions.receiver.accountHolder}</dd></div>
        <div><dt>Số tài khoản</dt><dd><strong>{instructions.receiver.accountNumber}</strong><button type="button" onClick={() => copyValue(instructions.receiver.accountNumber, "số tài khoản")}>Sao chép</button></dd></div>
        <div className="reference"><dt>Nội dung chuyển khoản</dt><dd><strong>{instructions.transferContent}</strong><button type="button" onClick={() => copyValue(instructions.transferContent, "nội dung chuyển khoản")}>Sao chép</button></dd></div>
        <div><dt>Payment code</dt><dd>{instructions.paymentCode}</dd></div>
        <div><dt>Hết hạn</dt><dd>{new Date(instructions.expiresAt).toLocaleString("vi-VN")}</dd></div>
      </dl><p className="payment-verification-note">QR và thông tin nhận tiền được đọc từ persisted Payment intent. Frontend không tạo intent mới và không tự xác nhận đã thanh toán.</p></section>
    </div> : null}

    {status.canPay && !instructions ? <div className="payment-action-error" role="alert">{instructionsError || "Backend cho phép thanh toán nhưng instructions chưa khả dụng. Không tự dựng QR hoặc thông tin tài khoản."}</div> : null}

    <section className="payment-summary-card"><div><p className="eyebrow">Payment projection version {status.version}</p><h2>{status.intentId}</h2></div><dl>
      <div><dt>Order status</dt><dd>{status.orderStatus}</dd></div>
      <div><dt>Payment status</dt><dd>{status.paymentStatus}</dd></div>
      <div><dt>Intent status</dt><dd>{status.intentStatus}</dd></div>
      <div><dt>Số tiền kỳ vọng</dt><dd>{VND.format(status.expectedAmountVnd)}</dd></div>
      <div><dt>Đã phân bổ</dt><dd>{VND.format(status.allocatedAmountVnd)}</dd></div>
      <div><dt>Đã hoàn</dt><dd>{VND.format(status.refundedAmountVnd)}</dd></div>
    </dl></section>

    {status.hasOpenReconciliation || status.supportReason ? <section className="payment-cod-card"><span aria-hidden="true">!</span><div><p className="eyebrow">Cần hỗ trợ/đối soát</p><h2>{status.supportReason ?? "Có reconciliation đang mở"}</h2><p>Không thanh toán lại hoặc tự suy diễn trạng thái. Hãy cung cấp support reference cho nhân viên hỗ trợ.</p><dl><div><dt>Support reference</dt><dd>{status.supportReference ?? "Chưa được cấp"}</dd></div><div><dt>Verification due</dt><dd>{status.verificationDueAt ? new Date(status.verificationDueAt).toLocaleString("vi-VN") : "Không có"}</dd></div></dl></div></section> : null}

    {pageError ? <div className="payment-action-error" role="alert">{pageError}</div> : null}
    {copyMessage ? <div className="payment-toast" role="status">{copyMessage}</div> : null}
    <section className="payment-actions"><div><button className="primary-button" type="button" disabled={checking} onClick={() => void checkStatus()}>{checking ? <><span className="spinner" />Đang kiểm tra…</> : "Kiểm tra trạng thái thanh toán"}</button><Link className="secondary-link" href={`/orders/${encodeURIComponent(orderId)}`}>Xem chi tiết Order</Link></div><p><strong>Polling:</strong> {status.nextPollAfterSeconds > 0 ? `Backend đề nghị kiểm tra lại sau ${status.nextPollAfterSeconds} giây.` : "Backend yêu cầu dừng tự động polling."}</p><Link href="/products">Tiếp tục khám phá sản phẩm</Link></section>
  </main>;
}
