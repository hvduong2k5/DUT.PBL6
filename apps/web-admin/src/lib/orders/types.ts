export const ADMIN_ORDER_STATUSES = [
  "PENDING_PAYMENT",
  "PAID",
  "CONFIRMED",
  "PROCESSING",
  "PACKED",
  "SHIPPED",
  "DELIVERED",
  "COMPLETED",
  "DELIVERY_FAILED",
  "EXPIRED",
  "CANCELLED"
] as const;

export type AdminOrderStatus = (typeof ADMIN_ORDER_STATUSES)[number];
export type AdminPaymentStatus = "PENDING" | "PAID" | "UNPAID" | "REFUND_PENDING" | "COD_PENDING_COLLECTION";
export type AdminOrderSource = "WEB_D2C" | "B2B" | "MARKETPLACE" | "OFFLINE";
export type AdminOrderSlaState = "ON_TRACK" | "AT_RISK" | "OVERDUE" | "COMPLETE";

export interface AdminOrderSummary {
  orderId: string;
  orderNumber: string;
  source: AdminOrderSource;
  sourceLabel: string;
  placedAt: string;
  status: AdminOrderStatus;
  statusLabel: string;
  paymentStatus: AdminPaymentStatus;
  paymentStatusLabel: string;
  totalVnd: number;
  currency: "VND";
  itemCount: number;
  customerLabel: string;
  fulfillmentStageLabel: string;
  slaState: AdminOrderSlaState;
  slaLabel: string;
  slaDueAt: string | null;
  blockedReason: string | null;
  updatedAt: string;
  revision: number;
}

export interface AdminOrderList {
  items: AdminOrderSummary[];
  statusCounts: Array<{ status: "ALL" | AdminOrderStatus; label: string; count: number }>;
  sources: Array<{ source: AdminOrderSource; label: string }>;
  paymentStatuses: Array<{ status: AdminPaymentStatus; label: string }>;
  calculatedAt: string;
}

export interface AdminOrderLine {
  lineId: string;
  productName: string;
  skuCode: string;
  skuLabel: string;
  quantity: number;
  unitPriceVnd: number;
  lineSubtotalVnd: number;
}

export interface AdminOrderRelatedState {
  code: string;
  label: string;
  reference: string | null;
  updatedAt: string | null;
  note: string | null;
}

export interface AdminOrderTimelineEvent {
  eventId: string;
  fromStatus: AdminOrderStatus | null;
  toStatus: AdminOrderStatus;
  toStatusLabel: string;
  occurredAt: string;
  actorLabel: string;
  sourceLabel: string;
  reason: string | null;
}

export interface AdminOrderDetail extends AdminOrderSummary {
  recipient: {
    displayName: string;
    phoneDisplay: string;
    emailDisplay: string;
    addressDisplay: string;
    deliveryNote: string | null;
    masked: boolean;
  };
  lines: AdminOrderLine[];
  pricing: { subtotalVnd: number; shippingFeeVnd: number; discountVnd: number; totalVnd: number };
  payment: AdminOrderRelatedState & { methodLabel: string };
  reservation: AdminOrderRelatedState;
  packing: AdminOrderRelatedState;
  shipping: AdminOrderRelatedState & { carrierName: string | null; trackingCode: string | null; estimatedDeliveryAt: string | null };
  timeline: AdminOrderTimelineEvent[];
}

interface AdminOrderErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }

export class AdminOrderApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly errors: Array<{ field: string; message: string }>;

  constructor(status: number, body: AdminOrderErrorBody) {
    super(body.message || "Không thể xử lý yêu cầu quản lý đơn hàng.");
    this.name = "AdminOrderApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
  }
}
