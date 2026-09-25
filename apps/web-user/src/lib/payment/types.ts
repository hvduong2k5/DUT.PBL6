export type PaymentMethod = "BANK_TRANSFER" | "COD";
export type PaymentStatus = "PENDING" | "SUCCEEDED" | "FAILED" | "EXPIRED" | "REQUIRES_RECONCILIATION" | "COD_PENDING_COLLECTION";
export type OrderPaymentState = "PENDING_PAYMENT" | "PAID" | "PAYMENT_EXPIRED" | "PAYMENT_REVIEW" | "PAYMENT_ON_DELIVERY";

export interface PaymentInstructions {
  providerLabel: string;
  beneficiary: string;
  accountNumberMasked: string;
  transferReference: string;
  qrPayloadType: "DEMO_ONLY";
}

export interface PaymentSummary {
  orderId: string;
  orderNumber: string;
  method: PaymentMethod;
  amountVnd: number;
  currency: "VND";
  paymentId: string;
  status: PaymentStatus;
  orderPaymentState: OrderPaymentState;
  expiresAt: string;
  instructions: PaymentInstructions | null;
  checkedAt: string;
}

export interface PaymentStatusResult {
  paymentId: string;
  status: PaymentStatus;
  orderPaymentState: OrderPaymentState;
  verifiedAt: string | null;
  message: string;
  canRetry: boolean;
}

export interface PaymentRetryResult {
  paymentId: string;
  status: "PENDING";
  orderPaymentState: "PENDING_PAYMENT";
  expiresAt: string;
  message: string;
}

interface PaymentErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  errors?: Array<{ field: string; message: string }>;
}

export class PaymentApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly errors: Array<{ field: string; message: string }>;

  constructor(status: number, body: PaymentErrorBody) {
    super(body.message || "Không thể xử lý thanh toán lúc này.");
    this.name = "PaymentApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.requestId = body.requestId;
    this.errors = body.errors ?? [];
  }
}
