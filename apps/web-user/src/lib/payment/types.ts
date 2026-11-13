import type {
  OrderPaymentStatus,
  OrderStatus,
  PaymentDisplayState,
  PaymentSupportReason
} from "@/lib/api-contract";

export interface PaymentInstructionsView {
  orderId: string;
  intentId: string;
  provider: "SEPAY";
  method: "VIETQR";
  amountVnd: number;
  currency: "VND";
  paymentCode: string;
  receiver: {
    bankCode: string;
    bankName: string;
    accountNumber: string;
    accountHolder: string;
    mode: "BANK_ACCOUNT" | "OFFICIAL_VA" | "CONTENT_VA";
  };
  transferContent: string;
  qrUrl: string;
  expiresAt: string;
  serverTime: string;
  canPay: true;
  version: number;
}

export interface PaymentStatusView {
  orderId: string;
  intentId: string;
  provider: "SEPAY";
  method: "VIETQR";
  orderStatus: Exclude<OrderStatus, "CONFIRMED_COD">;
  paymentStatus: OrderPaymentStatus;
  displayState: PaymentDisplayState;
  displayLabel: string;
  displayMessage: string;
  tone: "pending" | "success" | "error" | "review";
  intentStatus: "ACTIVE" | "SETTLED" | "EXPIRED" | "CANCELLED";
  expectedAmountVnd: number;
  allocatedAmountVnd: number;
  refundedAmountVnd: number;
  currency: "VND";
  expiresAt: string;
  serverTime: string;
  hasOpenReconciliation: boolean;
  canPay: boolean;
  version: number;
  verificationStartedAt: string | null;
  verificationDueAt: string | null;
  supportReference: string | null;
  supportReason: PaymentSupportReason | null;
  nextPollAfterSeconds: number;
  terminal: boolean;
}
