export type InventoryRiskState = "NORMAL" | "EXPIRING" | "EXPIRED_ONLY";
export type BatchExpiryState = "SAFE" | "EXPIRING" | "EXPIRED";
export type InventoryMovementType = "RECEIPT" | "RESERVATION" | "RELEASE" | "ISSUE" | "ADJUSTMENT" | "DAMAGE" | "LOSS";

export interface InventoryBalance {
  onHand: number;
  reserved: number;
  nonSellable: number;
  available: number;
}

export interface InventorySkuSummary extends InventoryBalance {
  skuId: string;
  skuCode: string;
  skuLabel: string;
  productId: string;
  productName: string;
  categoryId: string;
  categoryName: string;
  scopeLabel: string;
  riskState: InventoryRiskState;
  riskLabel: string;
  batchCount: number;
  expiringBatchCount: number;
  expiredBatchCount: number;
  updatedAt: string;
  revision: number;
}

export interface InventorySkuList {
  items: InventorySkuSummary[];
  riskCounts: Array<{ riskState: "ALL" | InventoryRiskState; label: string; count: number }>;
  categories: Array<{ categoryId: string; categoryName: string }>;
  calculatedAt: string;
}

export interface InventoryMovement {
  movementId: string;
  type: InventoryMovementType;
  typeLabel: string;
  quantity: number;
  occurredAt: string;
  actorLabel: string;
  reference: string;
}

export interface InventoryBatch {
  batchId: string;
  batchCode: string;
  manufacturingDate: string;
  expiresAt: string;
  expiryState: BatchExpiryState;
  expiryLabel: string;
  sourceReference: string;
  balance: InventoryBalance;
  movements: InventoryMovement[];
}

export interface InventorySkuDetail {
  skuId: string;
  skuCode: string;
  skuLabel: string;
  productId: string;
  productName: string;
  scopeLabel: string;
  balance: InventoryBalance;
  batches: InventoryBatch[];
  calculatedAt: string;
  revision: number;
}

export interface WarehouseWorkbenchBatch {
  batchId: string;
  batchCode: string;
  manufacturingDate: string;
  expiresAt: string;
  expiryState: BatchExpiryState;
  expiryLabel: string;
  balance: InventoryBalance;
}

export interface WarehouseWorkbenchSku {
  skuId: string;
  skuCode: string;
  skuLabel: string;
  productName: string;
  balance: InventoryBalance;
  batches: WarehouseWorkbenchBatch[];
  revision: number;
}

export interface WarehouseWorkbenchContext {
  skus: WarehouseWorkbenchSku[];
  recentMovements: Array<InventoryMovement & { skuCode: string; batchCode: string }>;
  pendingApprovalCount: number;
  calculatedAt: string;
  policyNotice: string;
}

export interface InventoryReceiptInput {
  skuId: string;
  batchMode: "EXISTING" | "NEW";
  batchId?: string;
  batchCode?: string;
  manufacturingDate?: string;
  expiresAt?: string;
  quantity: number;
  sourceReference: string;
  expectedRevision: number;
  idempotencyKey: string;
}

export interface InventoryIssueInput {
  skuId: string;
  batchId: string;
  purpose: "ORDER" | "APPROVED_NON_ORDER";
  quantity: number;
  reference: string;
  reason?: string;
  expectedRevision: number;
  idempotencyKey: string;
}

export interface InventoryAdjustmentInput {
  skuId: string;
  batchId: string;
  kind: "DAMAGE" | "LOSS" | "COUNT";
  quantity?: number;
  countedQuantity?: number;
  reference: string;
  reason: string;
  expectedRevision: number;
  idempotencyKey: string;
}

export interface InventoryOperationResult {
  operationId: string;
  status: "APPLIED" | "PENDING_APPROVAL";
  statusLabel: string;
  movementId: string | null;
  approvalRequestId: string | null;
  occurredAt: string;
  revision: number;
  message: string;
}

interface InventoryErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminInventoryApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: InventoryErrorBody) {
    super(body.message || "Không thể xử lý yêu cầu tồn kho.");
    this.name = "AdminInventoryApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
  }
}
