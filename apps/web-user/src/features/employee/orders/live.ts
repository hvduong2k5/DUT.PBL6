import { ADMIN_ORDER_STATUSES, type AdminOrderDetail, type AdminOrderList, type AdminOrderSlaState, type AdminOrderStatus, type AdminOrderSummary, type AdminPaymentStatus } from "./types";

interface OrderProjection {
  order_id: string;
  order_code: string;
  status: string;
  payment_status: string;
  payment_method: string;
  stock_status: string;
  customer_id: string | null;
  task_id: string | null;
  shipment_id: string | null;
  final_amount: number;
  item_count: number;
  version: number;
  operational_hold: boolean;
  has_active_case: boolean;
  sla_due_at: string | null;
  created_at: string;
  updated_at: string;
}

interface OrderDetailProjection {
  order: OrderProjection;
  calculated_at: string;
  subtotal: number;
  shipping_fee: number;
  discount: number;
  recipient: { name: string; phone: string; address: string } | null;
  lines: Array<{ sku_code: string; product_name: string; quantity: number; unit_price: number; line_total: number }>;
  history: Array<{ event_id: string; from_status: string | null; to_status: string; occurred_at: string }>;
}

export class AdminOrderProjectionError extends Error {}
const record = (value: unknown): value is Record<string, unknown> => typeof value === "object" && value !== null && !Array.isArray(value);
const timestamp = (value: unknown): value is string => typeof value === "string" && value.length > 0 && Number.isFinite(Date.parse(value));
const uuid = (value: unknown): value is string => typeof value === "string" && /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/iu.test(value);
const amount = (value: unknown): value is number => typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
function invalid(): never { throw new AdminOrderProjectionError("Dữ liệu đơn hàng từ dịch vụ không đúng định dạng."); }
function decodeOrder(value: unknown): OrderProjection {
  if (!record(value) || !uuid(value.order_id) || !timestamp(value.created_at) || !timestamp(value.updated_at)) return invalid();
  for (const key of ["order_code", "status", "payment_status", "payment_method", "stock_status"]) if (typeof value[key] !== "string" || !value[key]) return invalid();
  if (value.customer_id !== null && !uuid(value.customer_id)) return invalid();
  for (const key of ["task_id", "shipment_id"]) if (value[key] !== null && (typeof value[key] !== "string" || !value[key] || value[key].length > 200)) return invalid();
  if (!amount(value.final_amount) || !amount(value.item_count) || !amount(value.version) || value.version < 1 || typeof value.operational_hold !== "boolean" || typeof value.has_active_case !== "boolean") return invalid();
  if (value.sla_due_at !== null && !timestamp(value.sla_due_at)) return invalid();
  return value as unknown as OrderProjection;
}

const orderLabels: Record<AdminOrderStatus, string> = {
  PENDING_PAYMENT: "Chờ thanh toán", PAYMENT_FINALIZING: "Đang xác nhận thanh toán", PAID: "Đã thanh toán",
  CONFIRMED: "Đã xác nhận", PROCESSING: "Đang xử lý", PACKED: "Đã đóng gói", SHIPPED: "Đang giao",
  DELIVERED: "Đã giao", COMPLETED: "Hoàn tất", DELIVERY_FAILED: "Giao thất bại", EXPIRED: "Hết hạn",
  CANCELLED: "Đã hủy", UNKNOWN: "Chưa xác định",
};
const paymentLabels: Record<AdminPaymentStatus, string> = {
  UNPAID: "Chưa thanh toán", PENDING: "Đang xác nhận", PAID: "Đã thanh toán",
  REFUND_PENDING: "Chờ hoàn tiền", COD_PENDING_COLLECTION: "Chờ thu hộ",
  RECONCILIATION_REQUIRED: "Cần đối soát", PARTIALLY_REFUNDED: "Đã hoàn một phần",
  REFUNDED: "Đã hoàn tiền", UNKNOWN: "Chưa xác định",
};
const paymentMethodLabels: Record<string, string> = { VIETQR: "VietQR", COD: "COD" };
const slaLabels: Record<AdminOrderSlaState, string> = {
  ON_TRACK: "Trong SLA", AT_RISK: "Sắp quá SLA", OVERDUE: "Quá SLA",
  COMPLETE: "Đã kết thúc", UNAVAILABLE: "Chưa có hạn SLA",
};
const stockLabels: Record<string, string> = {
  RESERVED: "Đã giữ tồn", COMMITTING: "Đang chốt tồn", COMMITTED: "Đã chốt tồn",
  RELEASING: "Đang nhả tồn", RELEASED: "Đã nhả tồn", EXPIRED: "Giữ tồn đã hết hạn",
  REVERSED: "Đã đảo tồn",
};

function orderStatus(code: string): AdminOrderStatus {
  if (code === "CANCELLED_TIMEOUT") return "EXPIRED";
  if (code.startsWith("CANCELLED_")) return "CANCELLED";
  if (code === "CONFIRMED_COD") return "CONFIRMED";
  return ADMIN_ORDER_STATUSES.find((status) => status === code) ?? "UNKNOWN";
}

function slaState(order: OrderProjection, status: AdminOrderStatus, asOf: string): AdminOrderSlaState {
  if (!order.sla_due_at) return "UNAVAILABLE";
  if (["DELIVERED", "COMPLETED", "CANCELLED", "EXPIRED"].includes(status)) return "COMPLETE";
  const remaining = Date.parse(order.sla_due_at) - Date.parse(asOf);
  if (remaining <= 0) return "OVERDUE";
  return "ON_TRACK";
}

function summary(order: OrderProjection, asOf: string): AdminOrderSummary {
  const status = orderStatus(order.status);
  const paymentCode = order.payment_status === "CONFIRMED" ? "PAID" : order.payment_status;
  const paymentStatus = (Object.keys(paymentLabels).find((code) => code === paymentCode) ?? "UNKNOWN") as AdminPaymentStatus;
  const sla = slaState(order, status, asOf);
  return {
    orderId: order.order_id, orderNumber: order.order_code, status, statusLabel: orderLabels[status],
    paymentStatus, paymentStatusLabel: paymentLabels[paymentStatus],
    source: "WEB_D2C", sourceLabel: "Mua trực tuyến",
    totalVnd: order.final_amount, currency: "VND", itemCount: order.item_count,
    customerLabel: order.customer_id ? `Khách ${order.customer_id.slice(0, 8)}` : "Khách lẻ",
    placedAt: order.created_at, updatedAt: order.updated_at, revision: order.version,
    fulfillmentStageLabel: orderLabels[status], slaState: sla, slaLabel: slaLabels[sla], slaDueAt: order.sla_due_at,
    blockedReason: order.operational_hold ? "Đơn đang tạm dừng" : order.has_active_case ? "Đơn có yêu cầu hỗ trợ đang mở" : null,
  };
}

export function orderListFromItems(items: AdminOrderSummary[], calculatedAt: string, nextCursor: string | null): AdminOrderList {
  return {
    items, calculatedAt, nextCursor,
    statusCounts: [
      { status: "ALL", label: "Tất cả", count: items.length },
      ...ADMIN_ORDER_STATUSES.flatMap((status) => {
        const count = items.filter((item) => item.status === status).length;
        return count ? [{ status, label: orderLabels[status], count }] : [];
      }),
    ],
    sources: [{ source: "WEB_D2C", label: "Mua trực tuyến" }],
    paymentStatuses: Array.from(new Set(items.map((item) => item.paymentStatus)), (status) => ({ status, label: paymentLabels[status] })),
  };
}

export function liveOrderList(payload: unknown): AdminOrderList {
  if (!record(payload) || !Array.isArray(payload.items) || payload.items.length > 100 || !timestamp(payload.calculated_at) || !(payload.next_cursor === null || typeof payload.next_cursor === "string" && payload.next_cursor.length > 0 && payload.next_cursor.length <= 2048)) return invalid();
  const calculatedAt = payload.calculated_at;
  return orderListFromItems(payload.items.map((order) => summary(decodeOrder(order), calculatedAt)), calculatedAt, payload.next_cursor);
}

export function liveOrderDetail(value: unknown): AdminOrderDetail {
  if (!record(value) || !timestamp(value.calculated_at) || !amount(value.subtotal) || !amount(value.shipping_fee) || !amount(value.discount) || !Array.isArray(value.lines) || !Array.isArray(value.history)) return invalid();
  decodeOrder(value.order);
  if (value.recipient !== null && (!record(value.recipient) || !["name", "phone", "address"].every((key) => typeof (value.recipient as Record<string, unknown>)[key] === "string"))) return invalid();
  for (const line of value.lines) if (!record(line) || typeof line.sku_code !== "string" || !line.sku_code || typeof line.product_name !== "string" || !amount(line.quantity) || line.quantity < 1 || !amount(line.unit_price) || !amount(line.line_total)) return invalid();
  for (const entry of value.history) if (!record(entry) || !uuid(entry.event_id) || !(entry.from_status === null || typeof entry.from_status === "string") || typeof entry.to_status !== "string" || !timestamp(entry.occurred_at)) return invalid();
  const payload = value as unknown as OrderDetailProjection;
  const order = payload.order;
  const row = summary(order, payload.calculated_at);
  const hidden = "Đã ẩn theo quyền truy cập";
  const related = (code: string, label: string, reference: string | null = null) => ({ code, label, reference, updatedAt: null, note: null });
  return {
    ...row,
    recipient: payload.recipient ? {
      displayName: payload.recipient.name, phoneDisplay: payload.recipient.phone,
      addressDisplay: payload.recipient.address, emailDisplay: "Chưa có", deliveryNote: null, masked: false,
    } : { displayName: hidden, phoneDisplay: hidden, addressDisplay: hidden, emailDisplay: hidden, deliveryNote: null, masked: true },
    lines: payload.lines.map((line) => ({
      lineId: line.sku_code, productName: line.product_name, skuCode: line.sku_code, skuLabel: "",
      quantity: line.quantity, unitPriceVnd: line.unit_price, lineSubtotalVnd: line.line_total,
    })),
    pricing: { subtotalVnd: payload.subtotal, shippingFeeVnd: payload.shipping_fee, discountVnd: payload.discount, totalVnd: row.totalVnd },
    payment: { ...related(order.payment_status, row.paymentStatusLabel), methodLabel: paymentMethodLabels[order.payment_method] ?? order.payment_method },
    reservation: related(order.stock_status, stockLabels[order.stock_status] ?? "Chưa xác định"),
    packing: related(order.task_id ? "LINKED" : "UNAVAILABLE", order.task_id ? "Đã có task" : "Chưa có task", order.task_id),
    shipping: {
      ...related(order.shipment_id ? "LINKED" : "UNAVAILABLE", order.shipment_id ? "Đã có vận đơn" : "Chưa có vận đơn", order.shipment_id),
      carrierName: null, trackingCode: null, estimatedDeliveryAt: null,
    },
    timeline: payload.history.map((entry) => {
      const status = orderStatus(entry.to_status);
      return { eventId: entry.event_id, fromStatus: entry.from_status ? orderStatus(entry.from_status) : null,
        toStatus: status, toStatusLabel: orderLabels[status], occurredAt: entry.occurred_at, actorLabel: "", sourceLabel: "", reason: null };
    }),
  };
}
