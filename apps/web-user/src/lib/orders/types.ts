export type OrderStatus = "PENDING_PAYMENT" | "PAID" | "CONFIRMED" | "PROCESSING" | "PACKED" | "SHIPPED" | "DELIVERED" | "COMPLETED" | "DELIVERY_FAILED" | "EXPIRED" | "CANCELLED";
export type OrderPaymentStatus = "PENDING" | "PAID" | "UNPAID" | "REFUND_PENDING" | "COD_PENDING_COLLECTION";
export type OrderShippingStatus = "NOT_SHIPPED" | "READY" | "IN_TRANSIT" | "DELIVERED" | "FAILED";

export interface OrderPreviewItem {
  name: string;
  quantity: number;
}

export interface OrderListItem {
  orderId: string;
  orderNumber: string;
  placedAt: string;
  status: OrderStatus;
  statusLabel: string;
  paymentStatus: OrderPaymentStatus;
  shippingStatus: OrderShippingStatus;
  totalVnd: number;
  currency: "VND";
  itemCount: number;
  previewItems: OrderPreviewItem[];
  canCancel: boolean;
  cancelPolicyMessage: string;
}

export interface OrderListResponse {
  items: OrderListItem[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
  customer: { displayName: string; email: string };
}

export interface OrderLine {
  lineId: string;
  productName: string;
  skuLabel: string;
  quantity: number;
  unitPriceVnd: number;
  lineSubtotalVnd: number;
}

export interface OrderTimelineEvent {
  code: string;
  label: string;
  description: string;
  occurredAt: string;
  state: "COMPLETED" | "CURRENT" | "UPCOMING";
}

export interface OrderDetail {
  orderId: string;
  orderNumber: string;
  source: "WEB_D2C";
  placedAt: string;
  status: OrderStatus;
  statusLabel: string;
  statusDescription: string;
  payment: { method: "BANK_TRANSFER" | "COD"; status: OrderPaymentStatus; label: string };
  shipping: { methodName: string; status: OrderShippingStatus; label: string; trackingCode: string | null; estimatedDelivery: string | null };
  lines: OrderLine[];
  pricing: { subtotalVnd: number; shippingFeeVnd: number; discountVnd: number; totalVnd: number; currency: "VND" };
  recipient: { fullName: string; phoneDisplay: string; emailDisplay: string; addressDisplay: string; deliveryNote: string | null };
  timeline: OrderTimelineEvent[];
  cancellation: { canCancel: boolean; cancelBy: string | null; message: string; allowedReasons: Array<{ code: CancelReasonCode; label: string }> };
}

export type CancelReasonCode = "CHANGED_MIND" | "WRONG_INFORMATION" | "OTHER";

export interface CancelOrderInput {
  reasonCode: CancelReasonCode;
  note?: string;
  idempotencyKey: string;
}

export interface CancelOrderResult {
  orderId: string;
  status: "CANCELLED";
  cancelledAt: string;
  reservationReleaseRequested: boolean;
  refundRequired: boolean;
  refundStatus: "PENDING_HANDOFF" | null;
  message: string;
}

export interface GuestChallenge {
  challengeId: string;
  maskedDestination: string;
  expiresAt: string;
  resendAfterSeconds: number;
  message: string;
}

export interface GuestVerification {
  orderId: string;
  orderNumber: string;
  orderPath: string;
  accessExpiresAt: string;
}

interface OrderErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  retryAfterSeconds?: number;
  errors?: Array<{ field: string; message: string }>;
}

export class OrderApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly retryAfterSeconds?: number;
  readonly errors: Array<{ field: string; message: string }>;

  constructor(status: number, body: OrderErrorBody) {
    super(body.message || "Không thể xử lý Order lúc này.");
    this.name = "OrderApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.requestId = body.requestId;
    this.retryAfterSeconds = body.retryAfterSeconds;
    this.errors = body.errors ?? [];
  }
}
