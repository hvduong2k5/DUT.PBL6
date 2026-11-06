export const RETURN_CASE_STATUSES = ["RETURN_REQUESTED", "REVIEWING", "APPROVED", "REJECTED", "RETURNED", "REFUNDED"] as const;
export type ReturnCaseStatus = (typeof RETURN_CASE_STATUSES)[number];
export type ReturnCaseType = "CANCELLATION" | "RETURN_REFUND";
export type ReturnEvidenceState = "INCOMPLETE" | "READY" | "UNDER_REVIEW";
export type ReturnedGoodsState = "NOT_REQUIRED" | "AWAITING_RETURN" | "IN_TRANSIT" | "RECEIVED" | "INSPECTING" | "CLASSIFIED";
export type ReturnRefundState = "NOT_REQUIRED" | "NOT_CREATED" | "PENDING" | "COMPLETED" | "FAILED";

export interface ReturnCaseSummary {
  caseId: string; caseNumber: string; caseType: ReturnCaseType; caseTypeLabel: string; orderId: string; orderNumber: string;
  customerLabel: string; channelLabel: string; status: ReturnCaseStatus; statusLabel: string; reasonLabel: string;
  requestedItemCount: number; evidenceState: ReturnEvidenceState; evidenceStateLabel: string; evidenceCount: number;
  goodsState: ReturnedGoodsState; goodsStateLabel: string; refundState: ReturnRefundState; refundStateLabel: string;
  refundAmountVnd: number | null; financialMasked: boolean; assignedTo: string | null; updatedAt: string; revision: number;
}

export interface ReturnCaseList {
  items: ReturnCaseSummary[];
  statusCounts: Array<{ status: "ALL" | ReturnCaseStatus; label: string; count: number }>;
  caseTypes: Array<{ code: ReturnCaseType; label: string }>;
  calculatedAt: string;
}

export interface ReturnCaseDetail extends ReturnCaseSummary {
  createdAt: string;
  customer: { displayName: string; contactDisplay: string };
  order: { status: string; statusLabel: string; placedAt: string; deliveredAt: string | null; totalVnd: number; currency: string };
  lines: Array<{ lineId: string; productName: string; skuLabel: string; purchasedQuantity: number; requestedQuantity: number; approvedQuantity: number | null; unitPriceVnd: number }>;
  payment: { paymentId: string; statusLabel: string; methodLabel: string; amountPaidVnd: number | null; refundableVnd: number | null; masked: boolean };
  shipment: { shipmentId: string; statusLabel: string; deliveredAt: string | null; proofLabel: string | null };
  evidences: Array<{ evidenceId: string; fileName: string; mediaType: string; storageStateLabel: string; submittedByLabel: string; submittedAt: string; originalPreserved: boolean }>;
  evidenceMasked: boolean;
  decision: { outcomeLabel: string; decidedBy: string | null; decidedAt: string | null; basis: string | null } | null;
  refund: { refundId: string | null; state: ReturnRefundState; stateLabel: string; approvedAmountVnd: number | null; completedAmountVnd: number | null; methodLabel: string | null; masked: boolean };
  history: Array<{ eventId: string; label: string; occurredAt: string; actorLabel: string; detail: string | null }>;
}

interface AdminReturnErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminReturnApiError extends Error {
  readonly code: string; readonly status: number; readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: AdminReturnErrorBody) { super(body.message || "Không thể xử lý yêu cầu quản lý đổi trả."); this.name = "AdminReturnApiError"; this.status = status; this.code = body.code || "UNKNOWN_ERROR"; this.errors = body.errors ?? []; }
}
