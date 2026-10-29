import type {
  OrderAllowedAction as ContractOrderAllowedAction,
  OrderCancellationStatus as ContractOrderCancellationStatus,
  OrderCompensationStatus as ContractOrderCompensationStatus,
  OrderPaymentStatus as ContractOrderPaymentStatus,
  OrderStatus as ContractOrderStatus,
  OrderStockStatus as ContractOrderStockStatus
} from "@/lib/api-contract";

export type OrderStatus = ContractOrderStatus;
export type OrderPaymentStatus = ContractOrderPaymentStatus;
export type OrderStockStatus = ContractOrderStockStatus;
export type OrderCompensationStatus = ContractOrderCompensationStatus;
export type OrderCancellationStatus = ContractOrderCancellationStatus;
export type OrderAllowedAction = ContractOrderAllowedAction;

export interface OrderPreviewItem {
  name: string;
  quantity: number;
}

export interface OrderListItem {
  orderId: string;
  orderNumber: string;
  placedAt: string;
  updatedAt: string;
  version: number;
  status: OrderStatus;
  statusLabel: string;
  paymentMethod: string;
  paymentStatus: OrderPaymentStatus;
  paymentStatusLabel: string;
  stockStatus: OrderStockStatus;
  totalVnd: number;
  itemCount: number | null;
  previewItems: OrderPreviewItem[];
  allowedActions: OrderAllowedAction[];
  cancellationStatus: OrderCancellationStatus;
  compensationStatus: OrderCompensationStatus;
  canCancel: boolean;
  canViewTracking: boolean;
  cancelPolicyMessage: string;
}

export interface OrderListResponse {
  items: OrderListItem[];
  nextCursor: string;
  resultCount: number;
}

export interface OrderLine {
  lineId: string;
  productName: string;
  skuLabel: string;
  quantity: number;
  unitPriceVnd?: number;
  lineSubtotalVnd?: number;
}

export interface OrderTimelineEvent {
  code: string;
  label: string;
  description: string;
  occurredAt: string;
  state: "COMPLETED" | "CURRENT";
}

export interface OrderDetail {
  orderId: string;
  operationId: string;
  orderNumber: string;
  placedAt: string;
  updatedAt: string;
  version: number;
  status: OrderStatus;
  statusLabel: string;
  statusDescription: string;
  payment: {
    method: string;
    status: OrderPaymentStatus;
    label: string;
    reference: string;
    expiresAt: string;
  };
  stock: { status: OrderStockStatus; label: string };
  shipmentId: string | null;
  lines: OrderLine[];
  snapshotAvailable: boolean;
  pricing: { subtotalVnd: number; shippingFeeVnd: number; totalVnd: number; currency: "VND" };
  recipient?: { fullName: string; phoneDisplay: string; addressDisplay: string };
  timeline: OrderTimelineEvent[];
  allowedActions: OrderAllowedAction[];
  operationalHold: boolean;
  hasActiveCase: boolean;
  cancellation: {
    status: OrderCancellationStatus;
    compensationStatus: OrderCompensationStatus;
    canCancel: boolean;
    canRequestPaidCancellation: boolean;
    canViewTracking: boolean;
    message: string;
    allowedReasons: Array<{ code: CancelReasonCode; label: string }>;
  };
}

export type CancelReasonCode = "CHANGED_MIND" | "WRONG_INFORMATION" | "OTHER";

export interface CancelOrderInput {
  reasonCode: CancelReasonCode;
  note?: string;
  version: number;
  idempotencyKey: string;
}

export interface CancelOrderResult {
  cancellationId: string;
  orderId: string;
  status: Exclude<OrderCancellationStatus, "NONE">;
  compensationStatus: OrderCompensationStatus;
  version: number;
  reviewReference?: string | null;
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
