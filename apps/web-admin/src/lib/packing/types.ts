export const PACKING_TASK_STATUSES = ["READY", "ASSIGNED", "IN_PROGRESS", "BLOCKED", "COMPLETED", "CANCELLED"] as const;
export type PackingTaskStatus = (typeof PACKING_TASK_STATUSES)[number];
export type PackingPriority = "NORMAL" | "HIGH" | "URGENT";
export type PackingSlaState = "ON_TRACK" | "AT_RISK" | "OVERDUE" | "COMPLETE";
export type PackingChecklistState = "PENDING" | "PASSED" | "FAILED";

export interface PackingTaskSummary {
  taskId: string;
  taskNumber: string;
  orderId: string;
  orderNumber: string;
  orderSource: string;
  orderSourceLabel: string;
  status: PackingTaskStatus;
  statusLabel: string;
  assignee: { employeeId: string; displayName: string } | null;
  lineCount: number;
  totalQuantity: number;
  priority: PackingPriority;
  priorityLabel: string;
  slaState: PackingSlaState;
  slaLabel: string;
  dueAt: string | null;
  blockedReason: string | null;
  createdAt: string;
  updatedAt: string;
  revision: number;
}

export interface PackingTaskList {
  items: PackingTaskSummary[];
  statusCounts: Array<{ status: "ALL" | PackingTaskStatus; label: string; count: number }>;
  sources: Array<{ code: string; label: string }>;
  assignees: Array<{ employeeId: string; displayName: string }>;
  calculatedAt: string;
}

export interface PackingAllocation {
  allocationId: string;
  batchCode: string;
  locationLabel: string | null;
  quantity: number;
  expiresAt: string | null;
  eligibilityLabel: string;
  eligible: boolean;
}

export interface PackingPickLine {
  lineId: string;
  productName: string;
  skuCode: string;
  skuLabel: string;
  quantity: number;
  allocations: PackingAllocation[];
}

export interface PackingChecklistItem {
  itemId: string;
  label: string;
  required: boolean;
  state: PackingChecklistState;
  stateLabel: string;
  confirmedBy: string | null;
  confirmedAt: string | null;
  note: string | null;
}

export interface PackingTaskDetail extends PackingTaskSummary {
  prerequisites: Array<{ code: string; label: string; ready: boolean; detail: string }>;
  pickLines: PackingPickLine[];
  checklist: PackingChecklistItem[];
  completionReadiness: { ready: boolean; label: string; missingConditions: string[] };
  history: Array<{ eventId: string; label: string; occurredAt: string; actorLabel: string; detail: string | null }>;
}

export interface PackingChecklistUpdateInput {
  state: "PASSED" | "FAILED";
  note?: string;
  expectedRevision: number;
  idempotencyKey: string;
}

export interface PackingChecklistUpdateResult {
  taskId: string;
  item: PackingChecklistItem;
  revision: number;
  message: string;
}

export interface PackingTaskCompleteInput {
  expectedRevision: number;
  idempotencyKey: string;
}

export interface PackingTaskCompleteResult {
  taskId: string;
  status: "COMPLETED";
  statusLabel: string;
  completedAt: string;
  completedBy: string;
  revision: number;
  message: string;
}

interface AdminPackingErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminPackingApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: AdminPackingErrorBody) {
    super(body.message || "Không thể xử lý yêu cầu quản lý đóng gói.");
    this.name = "AdminPackingApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
  }
}
