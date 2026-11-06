export const PAYMENT_STATUSES = ["PENDING", "CONFIRMED", "FAILED", "EXPIRED", "CANCELLED", "COD_PENDING_COLLECTION", "REFUND_PENDING"] as const;
export const RECONCILIATION_RESULTS = ["MATCHED", "PENDING", "AMOUNT_MISMATCH", "REFERENCE_MISMATCH", "STATUS_MISMATCH", "UNMATCHED_TRANSACTION", "DUPLICATE_SUSPECTED"] as const;
export type PaymentStatus = (typeof PAYMENT_STATUSES)[number];
export type ReconciliationResult = (typeof RECONCILIATION_RESULTS)[number];
export type PaymentRiskState = "NONE" | "ATTENTION" | "BLOCKED";

export interface PaymentCaseSummary {
  caseId: string; caseNumber: string; orderId: string; orderNumber: string; orderStatus: string; orderStatusLabel: string;
  paymentId: string; methodCode: string; methodLabel: string; paymentStatus: PaymentStatus; paymentStatusLabel: string;
  amountDueVnd: number; amountConfirmedVnd: number; transactionCount: number; result: ReconciliationResult; resultLabel: string;
  riskState: PaymentRiskState; riskLabel: string; riskReason: string | null; referenceDisplay: string | null;
  lastTransactionAt: string | null; updatedAt: string; revision: number;
}

export interface PaymentCaseList {
  items: PaymentCaseSummary[];
  resultCounts: Array<{ result: "ALL" | ReconciliationResult; label: string; count: number }>;
  methods: Array<{ code: string; label: string }>;
  paymentStatuses: Array<{ code: PaymentStatus; label: string }>;
  calculatedAt: string;
}

export interface PaymentAttempt { attemptId: string; status: string; statusLabel: string; amountVnd: number; createdAt: string; expiresAt: string | null; }
export interface PaymentTransaction { transactionId: string; providerLabel: string; providerReferenceDisplay: string; bankReferenceDisplay: string | null; receivedAmountVnd: number; currency: string; providerStatusCode: string; verified: boolean; occurredAt: string; receivedAt: string; duplicate: boolean; masked: boolean; }
export interface PaymentComparison { field: string; label: string; expectedDisplay: string; actualDisplay: string; matched: boolean; }
export interface PaymentHistoryEvent { eventId: string; label: string; occurredAt: string; actorLabel: string; detail: string | null; }

export interface PaymentCaseDetail extends PaymentCaseSummary {
  order: { placedAt: string; totalVnd: number; currency: string };
  attempts: PaymentAttempt[];
  transactions: PaymentTransaction[];
  comparisons: PaymentComparison[];
  history: PaymentHistoryEvent[];
  originalEvidencePreserved: boolean;
}

interface AdminPaymentErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminPaymentApiError extends Error {
  readonly code: string; readonly status: number; readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: AdminPaymentErrorBody) { super(body.message || "Không thể xử lý yêu cầu vận hành thanh toán."); this.name = "AdminPaymentApiError"; this.status = status; this.code = body.code || "UNKNOWN_ERROR"; this.errors = body.errors ?? []; }
}
