// Staff payment workspace contract, aligned with the real order-service
// staff API (F-42): GET /api/v1/admin/payments/reconciliation-cases[/{id}]
// returns persistence PaymentCase views — case_id, order_id, receipt_id,
// observation_id, support_reference, reason, status, assigned_to, version,
// evidence_ids, refund_ids. Amounts, attempts and provider transactions are
// not part of that contract and are not shown.

export const PAYMENT_CASE_REASONS = ["UNDERPAYMENT", "OVERPAYMENT", "LATE_PAYMENT", "UNMATCHED_REFERENCE", "EXTRA_RECEIPT", "BLOCKED_PAYMENT"] as const;
export type PaymentCaseReason = (typeof PAYMENT_CASE_REASONS)[number];
export const PAYMENT_CASE_STATUSES = ["OPEN", "ASSIGNED", "RESOLVED"] as const;
export type PaymentCaseStatus = (typeof PAYMENT_CASE_STATUSES)[number];

export const PAYMENT_CASE_REASON_LABELS: Record<string, string> = {
  UNDERPAYMENT: "Trả thiếu",
  OVERPAYMENT: "Trả thừa",
  LATE_PAYMENT: "Trả muộn sau khi đơn đã hết hạn",
  UNMATCHED_REFERENCE: "Không khớp tham chiếu",
  EXTRA_RECEIPT: "Giao dịch thừa",
  BLOCKED_PAYMENT: "Thanh toán bị chặn"
};

export const PAYMENT_CASE_STATUS_LABELS: Record<string, string> = {
  OPEN: "Chưa xử lý",
  ASSIGNED: "Đã tiếp nhận",
  RESOLVED: "Đã xử lý"
};

export interface PaymentCaseSummary {
  caseId: string;
  supportReference: string;
  orderId: string | null;
  receiptId: string | null;
  reason: string;
  reasonLabel: string;
  status: string;
  statusLabel: string;
  assignedTo: string | null;
  version: number;
}

export interface PaymentCaseList {
  items: PaymentCaseSummary[];
  nextCursor: string | null;
  calculatedAt: string;
}

export interface PaymentCaseDetail extends PaymentCaseSummary {
  observationId: string | null;
  evidenceIds: string[];
  refundIds: string[];
}

interface AdminPaymentErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminPaymentApiError extends Error {
  readonly code: string; readonly status: number; readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: AdminPaymentErrorBody) { super(body.message || "Không thể xử lý yêu cầu vận hành thanh toán."); this.name = "AdminPaymentApiError"; this.status = status; this.code = body.code || "UNKNOWN_ERROR"; this.errors = body.errors ?? []; }
}
