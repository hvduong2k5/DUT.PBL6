export type AdminB2BStatus = "REQUESTED" | "NEEDS_INFO" | "DRAFT" | "SENT" | "ACCEPTED" | "REJECTED" | "EXPIRED" | "WITHDRAWN" | "CONVERTED";
export type AdminB2BSlaState = "ON_TRACK" | "AT_RISK" | "OVERDUE";
export type AdminB2BAction = "VIEW" | "ASSIGN" | "REQUEST_INFO" | "CREATE_DRAFT" | "ISSUE" | "WITHDRAW" | "REJECT" | "CONVERT_ORDER";
export type AdminB2BFileScanStatus = "SAFE" | "PROCESSING" | "REJECTED";

export interface AdminB2BFileSummary {
  fileId: string;
  fileName: string;
  mediaType: string;
  scanStatus: AdminB2BFileScanStatus;
}

export interface AdminB2BFileVersion {
  version: number;
  isCurrent: boolean;
  fileName: string;
  mediaType: string;
  sizeBytes: number;
  scanStatus: AdminB2BFileScanStatus;
  uploadedAt: string;
  uploadedByLabel: string;
  referencedByQuoteVersions: number[];
}

export interface AdminB2BFileRecord {
  fileId: string;
  purposeLabel: string;
  ownerOrganizationId: string;
  ownerOrganizationName: string;
  currentVersion: number;
  versions: AdminB2BFileVersion[];
}

export interface AdminB2BFileHistory {
  requestId: string;
  requestNumber: string;
  items: AdminB2BFileRecord[];
}

export interface AdminB2BRequestSummary {
  requestId: string;
  requestNumber: string;
  companyName: string;
  taxCode: string;
  submittedAt: string;
  lineCount: number;
  totalQuantity: number;
  status: AdminB2BStatus;
  statusLabel: string;
  slaState: AdminB2BSlaState;
  slaLabel: string;
  owner: { employeeId: string; displayName: string } | null;
  currentQuoteVersion: number | null;
  quoteExpiresAt: string | null;
  estimatedValueVnd: number | null;
  revision: number;
}

export interface AdminB2BList {
  items: AdminB2BRequestSummary[];
  statusCounts: Array<{ status: "ALL" | AdminB2BStatus; label: string; count: number }>;
  owners: Array<{ employeeId: string; displayName: string }>;
  updatedAt: string;
}

export interface AdminB2BRequestDetail extends AdminB2BRequestSummary {
  company: { legalName: string; taxCode: string; invoiceAddress: string; verificationStatus: "VERIFIED" | "PENDING" };
  requester: { displayName: string; title: string; phone: string; email: string };
  purposeLabel: string;
  requestedDeliveryDate: string;
  deliveryLocation: string;
  notes: string;
  items: Array<{ skuId: string; name: string; variant: string; quantity: number; catalogPriceVnd: number }>;
  customizations: string[];
  files: AdminB2BFileSummary[];
  quote: null | { quoteId: string; version: number; status: "DRAFT" | "SENT" | "ACCEPTED" | "SUPERSEDED"; subtotalVnd: number; discountPercent: number; customizationVnd: number; shippingVnd: number; vatPercent: number; grandTotalVnd: number; expiresAt: string };
  activity: Array<{ occurredAt: string; actorLabel: string; description: string }>;
  allowedActions: AdminB2BAction[];
}

export interface AdminB2BActionInput {
  action: Exclude<AdminB2BAction, "VIEW">;
  expectedRevision: number;
  reason?: string;
  assigneeId?: string;
  quote?: { discountPercent: number; customizationVnd: number; shippingVnd: number; vatPercent: number; expiresAt: string; terms: string };
  invoiceSnapshotConfirmed?: boolean;
  availabilityCheckAcknowledged?: boolean;
  idempotencyKey: string;
}

export interface AdminB2BActionResult {
  requestId: string;
  status: AdminB2BStatus;
  statusLabel: string;
  message: string;
  quoteId?: string;
  quoteVersion?: number;
  orderId?: string;
  orderNumber?: string;
  revision: number;
}

export type AdminB2BQuoteVersionStatus = "DRAFT" | "SENT" | "ACCEPTED" | "SUPERSEDED" | "EXPIRED" | "WITHDRAWN";

export interface AdminB2BQuoteVersion {
  quoteId: string;
  quoteNumber: string;
  version: number;
  status: AdminB2BQuoteVersionStatus;
  statusLabel: string;
  immutable: boolean;
  createdAt: string;
  createdBy: string;
  issuedAt: string | null;
  expiresAt: string;
  supersedesVersion: number | null;
  organization: { legalName: string; taxCode: string };
  contact: { displayName: string; title: string; phone: string; email: string };
  items: Array<{ skuId: string; name: string; variant: string; quantity: number; unitPriceVnd: number; lineTotalVnd: number }>;
  customizations: string[];
  totals: { merchandiseVnd: number; discountVnd: number; customizationVnd: number; vatVnd: number; grandTotalVnd: number; depositPercent: number; depositVnd: number; remainingVnd: number };
  terms: Array<{ title: string; description: string }>;
  payment: { bankName: string; accountName: string; accountNumberMasked: string; transferContent: string };
  acceptedAt?: string;
  orderId?: string;
}

export interface AdminB2BQuoteVersionHistory {
  requestId: string;
  requestNumber: string;
  items: AdminB2BQuoteVersion[];
}

interface AdminB2BErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminB2BApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: AdminB2BErrorBody) { super(body.message || "Không thể xử lý yêu cầu B2B."); this.name = "AdminB2BApiError"; this.status = status; this.code = body.code || "UNKNOWN_ERROR"; this.errors = body.errors ?? []; }
}
