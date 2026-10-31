import {
  ContractDataError,
  parseVndAmount,
  type OrderPaymentStatus,
  type OrderStatus,
  type PaymentDisplayState,
  type PaymentInstructions,
  type PaymentStatus,
  type PaymentSupportReason
} from "@/lib/api-contract";
import type { PaymentInstructionsView, PaymentStatusView } from "./types";

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu;
const PAYMENT_CODE_PATTERN = /^OMP[A-Z0-9]{12}$/u;
const MAX_PAYMENT_AMOUNT = 1_000_000_000_000;
const ORDER_STATUSES = new Set<Exclude<OrderStatus, "CONFIRMED_COD">>([
  "PENDING_PAYMENT", "PAYMENT_FINALIZING", "PAID", "PROCESSING", "PACKED", "SHIPPED", "DELIVERED",
  "DELIVERY_FAILED", "COMPLETED", "CANCELLED_TIMEOUT", "CANCELLED_BY_USER", "CANCELLED_BY_ADMIN"
]);
const PAYMENT_STATUSES = new Set<OrderPaymentStatus>([
  "UNPAID", "PENDING", "CONFIRMED", "RECONCILIATION_REQUIRED", "PARTIALLY_REFUNDED", "REFUNDED"
]);
const DISPLAY_STATES = new Set<PaymentDisplayState>([
  "WAITING_PAYMENT", "VERIFYING", "VERIFICATION_DELAYED", "FINALIZING", "PAID",
  "RECONCILIATION_REQUIRED", "EXPIRED", "CANCELLED", "PARTIALLY_REFUNDED", "REFUNDED"
]);
const SUPPORT_REASONS = new Set<PaymentSupportReason>([
  "VERIFICATION_TIMEOUT", "UNDERPAYMENT", "OVERPAYMENT", "MULTIPLE_PARTIAL_PAYMENTS",
  "EXTRA_RECEIPT", "LATE_PAYMENT", "OTHER_REVIEW"
]);
const RECEIVER_MODES = new Set(["BANK_ACCOUNT", "OFFICIAL_VA", "CONTENT_VA"]);

const DISPLAY_COPY: Record<PaymentDisplayState, { label: string; message: string; tone: PaymentStatusView["tone"] }> = {
  WAITING_PAYMENT: { label: "Đang chờ thanh toán", message: "VietQR còn hiệu lực và backend chưa ghi nhận khoản tiền phù hợp.", tone: "pending" },
  VERIFYING: { label: "Đang xác minh giao dịch", message: "Backend đang xác minh bằng chứng từ nguồn thanh toán tin cậy.", tone: "pending" },
  VERIFICATION_DELAYED: { label: "Xác minh đang chậm", message: "Giao dịch cần thêm thời gian xác minh; không thanh toán lặp khi kết quả chưa rõ.", tone: "review" },
  FINALIZING: { label: "Đang hoàn tất thanh toán", message: "Khoản tiền đã được xác minh và Order đang hoàn tất các bước liên quan.", tone: "pending" },
  PAID: { label: "Thanh toán đã xác nhận", message: "Backend đã xác nhận thanh toán cho Order.", tone: "success" },
  RECONCILIATION_REQUIRED: { label: "Cần đối soát", message: "Khoản tiền cần được nhân viên đối soát; frontend không tự kết luận đã thanh toán.", tone: "review" },
  EXPIRED: { label: "Hướng dẫn đã hết hạn", message: "Không tiếp tục chuyển tiền theo VietQR đã hết hạn.", tone: "error" },
  CANCELLED: { label: "Thanh toán đã hủy", message: "Payment intent đã bị hủy và không còn nhận thanh toán.", tone: "error" },
  PARTIALLY_REFUNDED: { label: "Đã hoàn một phần", message: "Một phần khoản tiền đã được hoàn theo dữ liệu tài chính canonical.", tone: "review" },
  REFUNDED: { label: "Đã hoàn tiền", message: "Khoản tiền đã được hoàn theo dữ liệu tài chính canonical.", tone: "success" }
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isText(value: unknown): value is string {
  return typeof value === "string";
}

function isDateTime(value: unknown): value is string {
  return isText(value) && value.length > 0 && !Number.isNaN(Date.parse(value));
}

function isNullableDateTime(value: unknown): value is string | null {
  return value === null || isDateTime(value);
}

function isPaymentAmount(value: unknown, minimum = 0): value is number {
  return Number.isSafeInteger(value) && Number(value) >= minimum && Number(value) <= MAX_PAYMENT_AMOUNT;
}

function isPositiveVersion(value: unknown): value is number {
  return Number.isSafeInteger(value) && Number(value) >= 1;
}

function isHttpsUrl(value: unknown): value is string {
  if (!isText(value)) return false;
  try {
    return new URL(value).protocol === "https:";
  } catch {
    return false;
  }
}

export function isPaymentInstructions(value: unknown): value is PaymentInstructions {
  if (!isRecord(value) || !isRecord(value.receiver)) return false;
  return isText(value.order_id)
    && UUID_PATTERN.test(value.order_id)
    && isText(value.intent_id)
    && UUID_PATTERN.test(value.intent_id)
    && value.provider === "SEPAY"
    && value.method === "VIETQR"
    && isPaymentAmount(value.amount, 1)
    && value.currency === "VND"
    && isText(value.payment_code)
    && PAYMENT_CODE_PATTERN.test(value.payment_code)
    && isText(value.receiver.bank_code)
    && isText(value.receiver.bank_name)
    && isText(value.receiver.account_number)
    && isText(value.receiver.account_holder)
    && RECEIVER_MODES.has(String(value.receiver.mode))
    && isText(value.transfer_content)
    && isHttpsUrl(value.qr_url)
    && isDateTime(value.expires_at)
    && isDateTime(value.server_time)
    && value.can_pay === true
    && isPositiveVersion(value.version);
}

export function isPaymentStatus(value: unknown): value is PaymentStatus {
  if (!isRecord(value)) return false;
  const supportReason = value.support_reason;
  return isText(value.order_id)
    && UUID_PATTERN.test(value.order_id)
    && isText(value.intent_id)
    && UUID_PATTERN.test(value.intent_id)
    && value.provider === "SEPAY"
    && value.method === "VIETQR"
    && ORDER_STATUSES.has(value.order_status as Exclude<OrderStatus, "CONFIRMED_COD">)
    && PAYMENT_STATUSES.has(value.payment_status as OrderPaymentStatus)
    && DISPLAY_STATES.has(value.display_state as PaymentDisplayState)
    && ["ACTIVE", "SETTLED", "EXPIRED", "CANCELLED"].includes(String(value.intent_status))
    && isPaymentAmount(value.expected_amount)
    && isPaymentAmount(value.allocated_amount)
    && isPaymentAmount(value.refunded_amount)
    && Number(value.refunded_amount) <= Number(value.allocated_amount)
    && value.currency === "VND"
    && isDateTime(value.expires_at)
    && isDateTime(value.server_time)
    && typeof value.has_open_reconciliation === "boolean"
    && typeof value.can_pay === "boolean"
    && isPositiveVersion(value.version)
    && isNullableDateTime(value.verification_started_at)
    && isNullableDateTime(value.verification_due_at)
    && (value.support_reference === null || isText(value.support_reference))
    && (supportReason === null || SUPPORT_REASONS.has(supportReason as PaymentSupportReason))
    && Number.isSafeInteger(value.next_poll_after_seconds)
    && Number(value.next_poll_after_seconds) >= 0;
}

export function mapMvpPaymentInstructions(value: PaymentInstructions): PaymentInstructionsView {
  if (!isPaymentInstructions(value)) throw new ContractDataError("Payment instructions do not match the public contract.");
  return {
    orderId: value.order_id,
    intentId: value.intent_id,
    provider: value.provider,
    method: value.method,
    amountVnd: parseVndAmount(value.amount),
    currency: value.currency,
    paymentCode: value.payment_code,
    receiver: {
      bankCode: value.receiver.bank_code,
      bankName: value.receiver.bank_name,
      accountNumber: value.receiver.account_number,
      accountHolder: value.receiver.account_holder,
      mode: value.receiver.mode
    },
    transferContent: value.transfer_content,
    qrUrl: value.qr_url,
    expiresAt: value.expires_at,
    serverTime: value.server_time,
    canPay: value.can_pay,
    version: value.version
  };
}

export function mapMvpPaymentStatus(value: PaymentStatus): PaymentStatusView {
  if (!isPaymentStatus(value)) throw new ContractDataError("Payment status does not match the public contract.");
  const copy = DISPLAY_COPY[value.display_state];
  return {
    orderId: value.order_id,
    intentId: value.intent_id,
    provider: value.provider,
    method: value.method,
    orderStatus: value.order_status,
    paymentStatus: value.payment_status,
    displayState: value.display_state,
    displayLabel: copy.label,
    displayMessage: copy.message,
    tone: copy.tone,
    intentStatus: value.intent_status,
    expectedAmountVnd: parseVndAmount(value.expected_amount),
    allocatedAmountVnd: parseVndAmount(value.allocated_amount),
    refundedAmountVnd: parseVndAmount(value.refunded_amount),
    currency: value.currency,
    expiresAt: value.expires_at,
    serverTime: value.server_time,
    hasOpenReconciliation: value.has_open_reconciliation,
    canPay: value.can_pay,
    version: value.version,
    verificationStartedAt: value.verification_started_at,
    verificationDueAt: value.verification_due_at,
    supportReference: value.support_reference,
    supportReason: value.support_reason,
    nextPollAfterSeconds: value.next_poll_after_seconds,
    terminal: value.next_poll_after_seconds === 0
  };
}
