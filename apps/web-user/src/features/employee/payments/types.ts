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

export const PAYMENT_CASE_RESOLUTIONS = ["REFUNDED", "NO_PAYMENT_CONFIRMED", "EXTERNAL_FUNDS_RESOLVED"] as const;
export type PaymentCaseResolution = (typeof PAYMENT_CASE_RESOLUTIONS)[number];
export const PAYMENT_CASE_RESOLUTION_LABELS: Record<string, string> = {
  REFUNDED: "Đã hoàn tiền đầy đủ",
  NO_PAYMENT_CONFIRMED: "Xác nhận không phát sinh thanh toán",
  EXTERNAL_FUNDS_RESOLVED: "Đã xử lý nguồn tiền ngoài hệ thống"
};

// A decision record from the local case authority (F-55): created by a
// PAYMENT_CASE_DECIDE holder who is not the case assignee, consumed by
// reference when the assignee resolves the case.
export interface PaymentCaseDecision {
  decisionReference: string;
  caseId: string;
  resolution: string;
  resolutionLabel: string;
  allowed: boolean;
  evidenceIds: string[];
  policyVersion: string;
}

// Manual refund authority (F-42 slice 2), aligned with the order-service
// staff API: refunds, refund approvals and bank-statement certificates.
// Refund views are masked server-side; the full beneficiary account is
// revealed only inside a start instruction issued to the current
// executor, which this BFF passes through without persisting.

export const PAYMENT_REFUND_STATUS_LABELS: Record<string, string> = {
  MANUAL_REVIEW: "Chờ thực thi",
  SUBMITTED: "Đã bắt đầu chuyển",
  UNKNOWN: "Chưa rõ kết quả",
  SUCCEEDED: "Đã hoàn tiền",
  FAILED: "Không hoàn được"
};

export const PAYMENT_REFUND_EXECUTION_LABELS: Record<string, string> = {
  AWAITING_EXECUTOR: "Chờ người thực thi",
  CLAIMED: "Đã có người nhận thực thi",
  STARTED: "Đang chuyển tiền",
  EVIDENCE_PENDING: "Chờ xác minh chứng từ",
  UNKNOWN: "Chưa rõ kết quả",
  VERIFIED_SENT: "Đã xác minh: đã chuyển",
  VERIFIED_NOT_SENT: "Đã xác minh: chưa chuyển",
  CANCELLED: "Đã hủy nghĩa vụ"
};

export const REFUND_APPROVAL_STATE_LABELS: Record<string, string> = {
  APPROVED: "Đã phê duyệt",
  WITHDRAWN: "Đã rút phê duyệt"
};

export const REFUND_EVIDENCE_DECISION_LABELS: Record<string, string> = {
  SENT: "Đã chuyển tiền",
  NOT_SENT: "Chưa chuyển tiền",
  INCONCLUSIVE: "Chưa kết luận"
};

export interface PaymentRefundSummary {
  refundId: string;
  receiptId: string;
  orderId: string | null;
  approvalId: string;
  amount: number;
  currency: string;
  status: string;
  statusLabel: string;
  executionState: string;
  executionLabel: string;
  obligationState: string;
  executionMode: string;
  version: number;
  attemptNumber: number;
  attemptId: string | null;
  claimOwner: string | null;
  claimExpiresAt: string | null;
  beneficiaryMasked: string;
  evidenceIds: string[];
}

export interface PaymentRefundList {
  items: PaymentRefundSummary[];
  nextCursor: string | null;
}

export interface RefundApprovalItem {
  approvalId: string;
  receiptId: string;
  orderId: string | null;
  amount: number;
  currency: string;
  state: string;
  stateLabel: string;
  beneficiaryMasked: string;
  bank: string;
  holder: string;
  approverId: string;
  version: number;
  withdrawalReference: string | null;
  reason: string;
  createdAt: string;
}

export interface RefundCertificateItem {
  reference: string;
  refundId: string;
  attemptId: string;
  decision: string;
  decisionLabel: string;
  definitive: boolean;
  amount: number;
  currency: string;
  beneficiaryMasked: string;
  bank: string;
  holder: string;
  bankReference: string;
  occurredAt: string;
  sha256: string;
}

export interface RefundStartInstructionView {
  refund: PaymentRefundSummary;
  attemptId: string;
  amount: number;
  currency: string;
  bankCode: string;
  account: string;
  holder: string;
  transferContent: string;
  startedAt: string;
}
