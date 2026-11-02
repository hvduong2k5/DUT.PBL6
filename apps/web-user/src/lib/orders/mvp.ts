import {
  ContractDataError,
  parseVndAmount,
  type OrderAllowedAction,
  type OrderCancellation,
  type OrderCancellationStatus,
  type OrderCompensationStatus,
  type OrderPaymentStatus,
  type OrderQuote,
  type OrderStatus,
  type OrderStockStatus,
  type OrderTrackingResponse,
  type OwnedOrder,
  type OwnedOrdersResponse
} from "@/lib/api-contract";
import type { CancelOrderResult, OrderDetail, OrderListItem, OrderListResponse } from "./types";

const ORDER_STATUSES = new Set<OrderStatus>([
  "PENDING_PAYMENT", "PAYMENT_FINALIZING", "PAID", "CONFIRMED_COD", "PROCESSING", "PACKED", "SHIPPED",
  "DELIVERED", "DELIVERY_FAILED", "COMPLETED", "CANCELLED_TIMEOUT", "CANCELLED_BY_USER", "CANCELLED_BY_ADMIN"
]);
const PAYMENT_STATUSES = new Set<OrderPaymentStatus>([
  "UNPAID", "PENDING", "CONFIRMED", "RECONCILIATION_REQUIRED", "PARTIALLY_REFUNDED", "REFUNDED"
]);
const STOCK_STATUSES = new Set<OrderStockStatus>([
  "RESERVED", "COMMITTING", "COMMITTED", "RELEASING", "RELEASED", "EXPIRED", "REVERSED", "UNKNOWN"
]);
const COMPENSATION_STATUSES = new Set<OrderCompensationStatus>(["NOT_REQUIRED", "PENDING", "SUCCEEDED", "MANUAL_REVIEW"]);
const CANCELLATION_STATUSES = new Set<OrderCancellationStatus>(["NONE", "PENDING", "SUCCEEDED", "REJECTED", "MANUAL_REVIEW"]);
const ALLOWED_ACTIONS = new Set<OrderAllowedAction>(["PAY", "CANCEL", "REQUEST_CANCELLATION", "VIEW_TRACKING", "CONTACT_SUPPORT"]);
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu;

const STATUS_LABELS: Record<OrderStatus, string> = {
  PENDING_PAYMENT: "Chờ thanh toán",
  PAYMENT_FINALIZING: "Đang xác minh thanh toán",
  PAID: "Đã thanh toán",
  CONFIRMED_COD: "Đã xác nhận COD",
  PROCESSING: "Đang xử lý",
  PACKED: "Đã đóng gói",
  SHIPPED: "Đang giao",
  DELIVERED: "Đã giao",
  DELIVERY_FAILED: "Giao thất bại",
  COMPLETED: "Hoàn tất",
  CANCELLED_TIMEOUT: "Đã hủy do hết hạn",
  CANCELLED_BY_USER: "Khách hàng đã hủy",
  CANCELLED_BY_ADMIN: "Hệ thống đã hủy"
};

const PAYMENT_LABELS: Record<OrderPaymentStatus, string> = {
  UNPAID: "Chưa thanh toán",
  PENDING: "Đang xác minh",
  CONFIRMED: "Đã xác nhận",
  RECONCILIATION_REQUIRED: "Cần đối soát",
  PARTIALLY_REFUNDED: "Đã hoàn một phần",
  REFUNDED: "Đã hoàn tiền"
};

const STOCK_LABELS: Record<OrderStockStatus, string> = {
  RESERVED: "Đã giữ tồn",
  COMMITTING: "Đang chốt tồn",
  COMMITTED: "Đã chốt tồn",
  RELEASING: "Đang giải phóng tồn",
  RELEASED: "Đã giải phóng tồn",
  EXPIRED: "Giữ tồn đã hết hạn",
  REVERSED: "Tồn kho đã hoàn tác",
  UNKNOWN: "Đang phục hồi trạng thái tồn"
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isText(value: unknown): value is string {
  return typeof value === "string";
}

function isDateTime(value: unknown): value is string {
  return isText(value) && value.length > 0 && !Number.isNaN(Date.parse(value));
}

function isNullableDateTime(value: unknown): value is string | null {
  return value === null || isDateTime(value);
}

function isSafeInteger(value: unknown, minimum = 0): value is number {
  return Number.isSafeInteger(value) && Number(value) >= minimum;
}

function isNullableText(value: unknown): value is string | null {
  return value === null || isText(value);
}

function isOrderItem(value: unknown): boolean {
  if (!isRecord(value) || !isText(value.sku_code) || !isSafeInteger(value.quantity, 1)) return false;
  return (value.item_id === undefined || isText(value.item_id))
    && (value.unit_price === undefined || isSafeInteger(value.unit_price))
    && (value.product_name === undefined || isText(value.product_name))
    && (value.weight_grams === undefined || isSafeInteger(value.weight_grams))
    && (value.catalog_version === undefined || isSafeInteger(value.catalog_version));
}

function isOrderAddress(value: unknown): boolean {
  return isRecord(value)
    && [value.recipient_name, value.phone, value.street, value.ward_code, value.province_code]
      .every((field) => isText(field));
}

function isOrderQuote(value: unknown): value is OrderQuote {
  return isRecord(value)
    && isText(value.catalog_snapshot_id)
    && isDateTime(value.catalog_expires_at)
    && isText(value.shipping_quote_reference)
    && isText(value.quote_id)
    && isSafeInteger(value.cart_revision)
    && (value.payment_method === "VIETQR" || value.payment_method === "COD")
    && Array.isArray(value.items)
    && value.items.every(isOrderItem)
    && isOrderAddress(value.address)
    && isSafeInteger(value.subtotal)
    && isSafeInteger(value.shipping_fee)
    && isSafeInteger(value.final_amount)
    && isDateTime(value.expires_at)
    && isDateTime(value.shipping_expires_at)
    && (value.source_address_id === undefined || isText(value.source_address_id))
    && (value.source_address_version === undefined || isSafeInteger(value.source_address_version))
    && (value.customer_id === undefined || isText(value.customer_id));
}

export function isOwnedOrder(value: unknown): value is OwnedOrder {
  if (!isRecord(value)) return false;
  const actions = value.allowed_actions;
  return isText(value.order_id)
    && isText(value.operation_id)
    && isText(value.order_code)
    && isNullableText(value.customer_id)
    && isText(value.warehouse_id)
    && isText(value.payment_method)
    && ORDER_STATUSES.has(value.status as OrderStatus)
    && PAYMENT_STATUSES.has(value.payment_status as OrderPaymentStatus)
    && STOCK_STATUSES.has(value.stock_status as OrderStockStatus)
    && isSafeInteger(value.subtotal)
    && isSafeInteger(value.shipping_fee)
    && isSafeInteger(value.final_amount)
    && isSafeInteger(value.version, 1)
    && isSafeInteger(value.ready_generation)
    && typeof value.ready_authorized === "boolean"
    && typeof value.operational_hold === "boolean"
    && typeof value.has_active_case === "boolean"
    && isText(value.payment_reference)
    && isNullableText(value.task_id)
    && isNullableText(value.shipment_id)
    && isNullableText(value.seal_code)
    && isDateTime(value.payment_expires_at)
    && isDateTime(value.updated_at)
    && isDateTime(value.created_at)
    && isNullableDateTime(value.sla_due_at)
    && isNullableDateTime(value.completion_due_at)
    && isText(value.items_digest)
    && /^[0-9a-f]{64}$/u.test(value.items_digest)
    && COMPENSATION_STATUSES.has(value.compensation_status as OrderCompensationStatus)
    && CANCELLATION_STATUSES.has(value.cancellation_status as OrderCancellationStatus)
    && Array.isArray(actions)
    && new Set(actions).size === actions.length
    && actions.every((action) => ALLOWED_ACTIONS.has(action as OrderAllowedAction))
    && (value.snapshot === undefined || value.snapshot === null || isOrderQuote(value.snapshot));
}

export function isOwnedOrdersResponse(value: unknown): value is OwnedOrdersResponse {
  return isRecord(value)
    && Array.isArray(value.items)
    && value.items.every(isOwnedOrder)
    && isText(value.next_cursor);
}

export function isOrderTrackingResponse(value: unknown): value is OrderTrackingResponse {
  return isRecord(value)
    && Object.keys(value).length === 2
    && (value.shipment_id === null || (isText(value.shipment_id) && UUID_PATTERN.test(value.shipment_id)))
    && isText(value.order_status)
    && value.order_status.length > 0;
}

export function isOrderCancellation(value: unknown): value is OrderCancellation {
  if (!isRecord(value)) return false;
  return isText(value.cancellation_id)
    && UUID_PATTERN.test(value.cancellation_id)
    && isText(value.order_id)
    && UUID_PATTERN.test(value.order_id)
    && ["PENDING", "SUCCEEDED", "REJECTED", "MANUAL_REVIEW"].includes(String(value.status))
    && COMPENSATION_STATUSES.has(value.compensation_status as OrderCompensationStatus)
    && isSafeInteger(value.version, 1)
    && (value.review_reference === undefined || value.review_reference === null || isText(value.review_reference));
}

function cancellationMessage(order: OwnedOrder): string {
  if (order.cancellation_status === "PENDING") return "Yêu cầu hủy đang được xử lý; chưa được coi là đã hủy.";
  if (order.cancellation_status === "MANUAL_REVIEW") return "Yêu cầu hủy cần nhân viên xem xét thủ công.";
  if (order.cancellation_status === "REJECTED") return "Yêu cầu hủy gần nhất đã bị từ chối.";
  if (order.cancellation_status === "SUCCEEDED") return "Order đã được hủy theo quyết định của backend.";
  if (order.allowed_actions.includes("CANCEL")) return "Backend cho phép gửi lệnh hủy trực tiếp ở version hiện tại.";
  if (order.allowed_actions.includes("REQUEST_CANCELLATION")) return "Order cần quy trình Care phê duyệt; capability này chưa được bật trong giai đoạn hiện tại.";
  return "Order hiện không có hành động hủy trong allowed_actions.";
}

function mapListItem(order: OwnedOrder): OrderListItem {
  const previewItems = (order.snapshot?.items ?? []).map((item) => ({
    name: item.product_name || item.sku_code,
    quantity: item.quantity
  }));
  return {
    orderId: order.order_id,
    orderNumber: order.order_code,
    placedAt: order.created_at,
    updatedAt: order.updated_at,
    version: order.version,
    status: order.status,
    statusLabel: STATUS_LABELS[order.status],
    paymentMethod: order.payment_method,
    paymentStatus: order.payment_status,
    paymentStatusLabel: PAYMENT_LABELS[order.payment_status],
    stockStatus: order.stock_status,
    totalVnd: parseVndAmount(order.final_amount),
    itemCount: order.snapshot ? order.snapshot.items.reduce((total, item) => total + item.quantity, 0) : null,
    previewItems,
    allowedActions: order.allowed_actions,
    cancellationStatus: order.cancellation_status,
    compensationStatus: order.compensation_status,
    canCancel: order.allowed_actions.includes("CANCEL"),
    canViewTracking: order.allowed_actions.includes("VIEW_TRACKING"),
    cancelPolicyMessage: cancellationMessage(order)
  };
}

export function mapMvpOrderListResponse(value: OwnedOrdersResponse): OrderListResponse {
  if (!isOwnedOrdersResponse(value)) throw new ContractDataError("Owned orders response does not match the public contract.");
  const items = value.items.map(mapListItem);
  return { items, nextCursor: value.next_cursor, resultCount: items.length };
}

export function mapMvpOrderDetail(value: OwnedOrder): OrderDetail {
  if (!isOwnedOrder(value)) throw new ContractDataError("Owned order does not match the public contract.");
  const snapshot = value.snapshot ?? undefined;
  const lines = (snapshot?.items ?? []).map((item, index) => {
    const unitPriceVnd = item.unit_price === undefined ? undefined : parseVndAmount(item.unit_price);
    const lineSubtotalVnd = unitPriceVnd === undefined ? undefined : unitPriceVnd * item.quantity;
    if (lineSubtotalVnd !== undefined && !Number.isSafeInteger(lineSubtotalVnd)) {
      throw new ContractDataError("Order line subtotal exceeds the safe integer range.");
    }
    return {
      lineId: item.item_id || `${value.order_id}-${index + 1}`,
      productName: item.product_name || item.sku_code,
      skuLabel: `Mã SKU: ${item.sku_code}`,
      quantity: item.quantity,
      ...(unitPriceVnd !== undefined ? { unitPriceVnd } : {}),
      ...(lineSubtotalVnd !== undefined ? { lineSubtotalVnd } : {})
    };
  });
  return {
    orderId: value.order_id,
    operationId: value.operation_id,
    orderNumber: value.order_code,
    placedAt: value.created_at,
    updatedAt: value.updated_at,
    version: value.version,
    status: value.status,
    statusLabel: STATUS_LABELS[value.status],
    statusDescription: value.operational_hold
      ? "Order đang có operational hold và cần được xử lý trước khi tiếp tục."
      : value.has_active_case
        ? "Order đang có vụ việc liên quan cần được theo dõi."
        : "Trạng thái được đọc trực tiếp từ Order service.",
    payment: {
      method: value.payment_method,
      status: value.payment_status,
      label: PAYMENT_LABELS[value.payment_status],
      reference: value.payment_reference,
      expiresAt: value.payment_expires_at
    },
    stock: { status: value.stock_status, label: STOCK_LABELS[value.stock_status] },
    shipmentId: value.shipment_id,
    lines,
    snapshotAvailable: Boolean(snapshot),
    pricing: {
      subtotalVnd: parseVndAmount(value.subtotal),
      shippingFeeVnd: parseVndAmount(value.shipping_fee),
      totalVnd: parseVndAmount(value.final_amount),
      currency: "VND"
    },
    ...(snapshot ? {
      recipient: {
        fullName: snapshot.address.recipient_name,
        phoneDisplay: snapshot.address.phone,
        addressDisplay: [snapshot.address.street, snapshot.address.ward_code, snapshot.address.province_code].filter(Boolean).join(", ")
      }
    } : {}),
    allowedActions: value.allowed_actions,
    operationalHold: value.operational_hold,
    hasActiveCase: value.has_active_case,
    cancellation: {
      status: value.cancellation_status,
      compensationStatus: value.compensation_status,
      canCancel: value.allowed_actions.includes("CANCEL"),
      canRequestPaidCancellation: value.allowed_actions.includes("REQUEST_CANCELLATION"),
      canViewTracking: value.allowed_actions.includes("VIEW_TRACKING"),
      message: cancellationMessage(value),
      allowedReasons: value.allowed_actions.includes("CANCEL")
        ? [
            { code: "CHANGED_MIND", label: "Thay đổi nhu cầu" },
            { code: "WRONG_INFORMATION", label: "Thông tin đặt hàng chưa đúng" },
            { code: "OTHER", label: "Lý do khác" }
          ]
        : []
    }
  };
}

export function mapMvpCancellation(value: OrderCancellation): CancelOrderResult {
  if (!isOrderCancellation(value)) throw new ContractDataError("Order cancellation does not match the public contract.");
  const messages: Record<OrderCancellation["status"], string> = {
    PENDING: "Yêu cầu hủy đã được ghi nhận và đang chờ quyết định/compensation từ backend.",
    SUCCEEDED: "Backend đã xác nhận hủy Order thành công.",
    REJECTED: "Backend đã từ chối yêu cầu hủy Order.",
    MANUAL_REVIEW: "Yêu cầu hủy cần được nhân viên xem xét thủ công."
  };
  return {
    cancellationId: value.cancellation_id,
    orderId: value.order_id,
    status: value.status,
    compensationStatus: value.compensation_status,
    version: value.version,
    reviewReference: value.review_reference,
    message: messages[value.status]
  };
}
