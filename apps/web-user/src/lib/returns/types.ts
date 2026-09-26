export type ReturnResolution = "REPLACEMENT" | "ORIGINAL_PAYMENT_REFUND" | "LOYALTY_CREDIT";
export type ReturnReason = "DAMAGED_IN_TRANSIT" | "WRONG_ITEM" | "QUALITY_ISSUE" | "OTHER";
export type ReturnCaseStatus = "RETURN_REQUESTED" | "EVIDENCE_REQUIRED" | "REVIEWING" | "APPROVED" | "REJECTED" | "RETURN_IN_TRANSIT" | "RECEIVED" | "REFUND_PENDING" | "REFUNDED" | "CLOSED";

export interface ReturnEligibilityLine {
  lineId: string;
  productName: string;
  skuLabel: string;
  purchasedQty: number;
  alreadyClaimedQty: number;
  maxReturnQty: number;
  unitPriceVnd: number;
  eligible: boolean;
}

export interface ReturnEligibility {
  orderId: string;
  orderNumber: string;
  deliveredAt: string;
  policyWindowEndsAt: string;
  eligible: boolean;
  policyMessage: string;
  lines: ReturnEligibilityLine[];
  reasonOptions: Array<{ code: ReturnReason; label: string }>;
  resolutionOptions: Array<{ code: ReturnResolution; label: string; description: string }>;
  pickupSnapshot: { recipientName: string; phoneDisplay: string; addressDisplay: string };
}

export interface EvidenceMetadata {
  clientReference: string;
  fileName: string;
  mediaType: "image/jpeg" | "image/png" | "image/webp" | "video/mp4";
  sizeBytes: number;
}

export interface CreateReturnCaseInput {
  orderId: string;
  lines: Array<{ lineId: string; quantity: number }>;
  reasonCode: ReturnReason;
  preferredResolution: ReturnResolution;
  details: string;
  pickupNote?: string;
  evidence: EvidenceMetadata[];
  idempotencyKey: string;
}

export interface CreateReturnCaseResult {
  caseId: string;
  caseNumber: string;
  status: "RETURN_REQUESTED";
  casePath: string;
  createdAt: string;
  message: string;
}

export interface ReturnCaseStep {
  code: string;
  label: string;
  state: "COMPLETED" | "CURRENT" | "UPCOMING";
  occurredAt: string | null;
}

export interface ReturnCaseMessage {
  messageId: string;
  senderRole: "CUSTOMER" | "SUPPORT" | "SYSTEM";
  senderDisplayName: string;
  message: string;
  occurredAt: string;
}

export interface ReturnCaseDetail {
  caseId: string;
  caseNumber: string;
  orderId: string;
  orderNumber: string;
  createdAt: string;
  status: ReturnCaseStatus;
  statusLabel: string;
  statusDescription: string;
  steps: ReturnCaseStep[];
  decision: { label: string; summary: string; decidedAt: string } | null;
  items: Array<{ lineId: string; productName: string; skuLabel: string; quantity: number; unitPriceVnd: number; requestedResolution: ReturnResolution; approvedResolution: ReturnResolution | null }>;
  evidence: Array<{ evidenceId: string; label: string; mediaType: string; state: "READY" | "PROCESSING" | "REJECTED"; previewKind: "PACKAGE" | "VIDEO" | "WORKSHOP" | "DOCUMENT" }>;
  pickup: { carrierLabel: string; windowLabel: string; trackingCode: string; statusLabel: string; addressDisplay: string } | null;
  refund: { required: boolean; methodLabel: string | null; amountVnd: number; status: "PENDING" | "COMPLETED" | "FAILED" | null };
  communications: ReturnCaseMessage[];
}

interface ReturnErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  retryAfterSeconds?: number;
  existingCaseId?: string;
  errors?: Array<{ field: string; message: string }>;
}

export class ReturnApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly retryAfterSeconds?: number;
  readonly existingCaseId?: string;
  readonly errors: Array<{ field: string; message: string }>;

  constructor(status: number, body: ReturnErrorBody) {
    super(body.message || "Không thể xử lý hồ sơ hậu mãi lúc này.");
    this.name = "ReturnApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.requestId = body.requestId;
    this.retryAfterSeconds = body.retryAfterSeconds;
    this.existingCaseId = body.existingCaseId;
    this.errors = body.errors ?? [];
  }
}
