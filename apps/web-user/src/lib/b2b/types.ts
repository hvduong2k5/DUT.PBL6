export type B2BRequestStatus = "REQUESTED" | "NEEDS_INFO" | "QUOTED" | "ACCEPTED" | "REJECTED";
export type B2BQuoteStatus = "SENT" | "ACCEPTED" | "EXPIRED" | "WITHDRAWN" | "SUPERSEDED";
export type B2BPurpose = "VIP_CUSTOMER_GIFT" | "EMPLOYEE_GIFT" | "EVENT_GIFT" | "PARTNER_GIFT";

export interface B2BOrganization {
  organizationId: string;
  legalName: string;
  taxCode: string;
  representativeName: string;
  representativeTitle: string;
  phone: string;
  email: string;
  invoiceAddress: string;
  verificationStatus: "VERIFIED" | "PENDING";
}

export interface B2BCatalogItem {
  skuId: string;
  name: string;
  variant: string;
  unitPriceVnd: number;
  minimumQuantity: number;
  imageUrl?: string;
}

export interface B2BAttachmentMetadata {
  clientReference: string;
  fileName: string;
  mediaType: "image/png" | "image/jpeg" | "image/webp" | "application/pdf" | "image/svg+xml";
  sizeBytes: number;
}

export interface B2BRequestSummary {
  requestId: string;
  requestNumber: string;
  status: B2BRequestStatus;
  statusLabel: string;
  submittedAt: string;
  totalQuantity: number;
  purposeLabel: string;
  quoteId?: string;
}

export interface B2BContext {
  organization: B2BOrganization;
  catalog: B2BCatalogItem[];
  deliveryProvinces: Array<{ provinceCode: string; provinceName: string }>;
  recentRequests: B2BRequestSummary[];
  discountTiers: Array<{ minimumQuantity: number; maximumQuantity?: number; discountPercent: number; label: string }>;
  attachmentPolicy: { allowedMediaTypes: B2BAttachmentMetadata["mediaType"][]; maxFiles: number; maxBytesPerFile: number };
}

export interface CreateB2BQuoteRequestInput {
  organizationId: string;
  purpose: B2BPurpose;
  requestedDeliveryDate: string;
  deliveryProvinceCode: string;
  deliveryProvinceName: string;
  items: Array<{ skuId: string; quantity: number }>;
  branding: { engraveLogo: boolean; customSleeve: boolean; greetingCard: boolean; ribbon: boolean; attachments: B2BAttachmentMetadata[] };
  notes?: string;
  invoiceRequested: boolean;
  sampleRequested: boolean;
  idempotencyKey: string;
}

export interface CreateB2BQuoteRequestResult extends B2BRequestSummary {
  message: string;
}

export interface B2BQuoteDetail {
  quoteId: string;
  quoteNumber: string;
  version: number;
  status: B2BQuoteStatus;
  statusLabel: string;
  issuedAt: string;
  expiresAt: string;
  organization: Pick<B2BOrganization, "organizationId" | "legalName" | "taxCode">;
  contact: { displayName: string; title: string; phone: string; email: string };
  items: Array<{ skuId: string; name: string; variant: string; quantity: number; unitPriceVnd: number; lineTotalVnd: number }>;
  customizations: string[];
  totals: { merchandiseVnd: number; discountVnd: number; customizationVnd: number; vatVnd: number; grandTotalVnd: number; depositPercent: number; depositVnd: number; remainingVnd: number };
  terms: Array<{ title: string; description: string }>;
  payment: { bankName: string; accountName: string; accountNumberMasked: string; transferContent: string };
  canAccept: boolean;
  acceptedAt?: string;
  orderId?: string;
}

export interface AcceptB2BQuoteResult {
  quoteId: string;
  status: "ACCEPTED";
  acceptedAt: string;
  orderId: string;
  orderNumber: string;
  message: string;
}

interface B2BErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }>; requestId?: string }
export class B2BApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly errors: Array<{ field: string; message: string }>;
  readonly requestId?: string;
  constructor(status: number, body: B2BErrorBody) {
    super(body.message || "Không thể xử lý yêu cầu B2B lúc này.");
    this.name = "B2BApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
    this.requestId = body.requestId;
  }
}
