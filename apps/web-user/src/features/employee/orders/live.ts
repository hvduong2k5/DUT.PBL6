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

export function liveOrderList(payload: { items: OrderProjection[]; calculated_at: string; next_cursor: string | null }): AdminOrderList {
  return orderListFromItems(payload.items.map((order) => summary(order, payload.calculated_at)), payload.calculated_at, payload.next_cursor);
}

export function liveOrderDetail(payload: OrderDetailProjection): AdminOrderDetail {
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
    payment: { ...related(order.payment_status, row.paymentStatusLabel), methodLabel: order.payment_method },
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
