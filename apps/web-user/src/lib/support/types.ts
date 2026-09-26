export type SupportCustomerMode = "REGISTERED" | "GUEST";
export type SupportPriority = "NORMAL" | "IMPORTANT" | "URGENT";
export type SupportTicketStatus = "NEW" | "IN_PROGRESS" | "WAITING_CUSTOMER" | "RESOLVED" | "CLOSED";
export type SupportSubjectCode = "PRODUCT_ADVICE" | "ORDER_HELP" | "DELIVERY_HELP" | "PAYMENT_HELP" | "RETURN_HELP" | "OTHER";

export interface SupportAttachmentMetadata {
  clientReference: string;
  fileName: string;
  mediaType: "image/jpeg" | "image/png" | "image/webp" | "application/pdf";
  sizeBytes: number;
}

export interface SupportContext {
  customerMode: SupportCustomerMode;
  customer: { displayName: string; emailDisplay: string; phoneDisplay: string; membershipLabel?: string } | null;
  subjects: Array<{ code: SupportSubjectCode; label: string; description: string }>;
  priorities: Array<{ code: SupportPriority; label: string; description: string; responsePromiseLabel: string }>;
  linkableOrders: Array<{ orderId: string; orderNumber: string; statusLabel: string; placedAt: string; summary: string }>;
  servicePromise: string;
  attachmentPolicy: { allowedMediaTypes: SupportAttachmentMetadata["mediaType"][]; maxFiles: number; maxBytesPerFile: number };
}

export interface CreateSupportTicketInput {
  subjectCode: SupportSubjectCode;
  title: string;
  message: string;
  requestedPriority: SupportPriority;
  linkedOrderId?: string;
  contact?: { displayName: string; email: string; phone?: string };
  attachments: SupportAttachmentMetadata[];
  idempotencyKey: string;
}

export interface CreateSupportTicketResult {
  ticketId: string;
  ticketNumber: string;
  status: "NEW";
  ticketPath: string;
  createdAt: string;
  message: string;
}

export interface SupportPublicAttachment {
  attachmentId: string;
  label: string;
  mediaType: string;
  state: "READY" | "PROCESSING" | "REJECTED";
}

export interface SupportMessage {
  messageId: string;
  senderRole: "CUSTOMER" | "SUPPORT" | "SYSTEM";
  senderDisplayName: string;
  message: string;
  occurredAt: string;
  attachments: SupportPublicAttachment[];
}

export interface SupportTicketDetail {
  ticketId: string;
  ticketNumber: string;
  createdAt: string;
  updatedAt: string;
  status: SupportTicketStatus;
  statusLabel: string;
  statusDescription: string;
  subjectLabel: string;
  title: string;
  requestedPriority: SupportPriority;
  requestedPriorityLabel: string;
  servicePromise: { label: string; targetAt: string | null; met: boolean | null };
  publicAssignee: { displayName: string; roleLabel: string } | null;
  linkedOrder: { orderId: string; orderNumber: string; statusLabel: string; products: Array<{ name: string; variant: string; quantity: number; priceVnd: number }> } | null;
  messages: SupportMessage[];
  canReply: boolean;
}

interface SupportErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  retryAfterSeconds?: number;
  existingTicketId?: string;
  errors?: Array<{ field: string; message: string }>;
}

export class SupportApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly retryAfterSeconds?: number;
  readonly existingTicketId?: string;
  readonly errors: Array<{ field: string; message: string }>;

  constructor(status: number, body: SupportErrorBody) {
    super(body.message || "Không thể xử lý yêu cầu hỗ trợ lúc này.");
    this.name = "SupportApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.requestId = body.requestId;
    this.retryAfterSeconds = body.retryAfterSeconds;
    this.existingTicketId = body.existingTicketId;
    this.errors = body.errors ?? [];
  }
}
