import type { CancelOrderResult, OrderDetail, OrderListItem, OrderStatus } from "./types";

interface Money { currency_code: string; units: number; nanos: number }

export interface CustomerCoreOrderListItem {
  order_id: string;
  status: string;
  total_items: number;
  final_amount: Money;
  created_at: string;
}

export interface CustomerCoreOrderDetail {
  order_id: string;
  status: string;
  shipping_address: {
    recipient_name: string;
    phone_number: string;
    street_address: string;
    ward: string;
    district: string;
    province: string;
  };
  items: Array<{ sku_code: string; quantity: number; price: Money }>;
  subtotal_amount: Money;
  discount_amount: Money;
  shipping_fee: Money;
  final_amount: Money;
  tracking_code: string | null;
  seal_code?: string | null;
  packing_video_url?: string | null;
  created_at: string;
}

export interface CustomerCoreTracking {
  tracking_code: string;
  carrier_name: string;
  current_status: string;
  checkpoints: Array<{ status: string; timestamp: string; location: string; description: string }>;
}

const STATUS_LABELS: Record<OrderStatus, string> = {
  PENDING_PAYMENT: "Chờ thanh toán", PAID: "Đã thanh toán", CONFIRMED: "Đã xác nhận",
  PROCESSING: "Đang chuẩn bị", PACKED: "Đã đóng gói", SHIPPED: "Đang giao",
  DELIVERED: "Đã giao", COMPLETED: "Hoàn tất", DELIVERY_FAILED: "Giao thất bại",
  EXPIRED: "Đã hết hạn", CANCELLED: "Đã hủy"
};

export function mapCustomerCoreOrderStatus(status: string): OrderStatus {
  if (status === "SHIPPING") return "SHIPPED";
  if (status === "CANCELLED_BY_USER") return "CANCELLED";
  return status in STATUS_LABELS ? status as OrderStatus : "PROCESSING";
}

export function mapCustomerCoreOrderListItem(order: CustomerCoreOrderListItem): OrderListItem {
  const status = mapCustomerCoreOrderStatus(order.status);
  return {
    orderId: order.order_id,
    orderNumber: order.order_id,
    placedAt: order.created_at,
    status,
    statusLabel: STATUS_LABELS[status],
    paymentStatus: status === "PENDING_PAYMENT" ? "UNPAID" : "PAID",
    shippingStatus: status === "SHIPPED" ? "IN_TRANSIT" : status === "COMPLETED" || status === "DELIVERED" ? "DELIVERED" : "NOT_SHIPPED",
    totalVnd: order.final_amount.units,
    currency: "VND",
    itemCount: order.total_items,
    previewItems: [],
    canCancel: status === "PENDING_PAYMENT",
    cancelPolicyMessage: status === "PENDING_PAYMENT" ? "Có thể hủy trước khi thanh toán." : "Đơn hiện không còn trong giai đoạn tự hủy."
  };
}

export function mapCustomerCoreOrderDetail(order: CustomerCoreOrderDetail, tracking?: CustomerCoreTracking): OrderDetail {
  const status = mapCustomerCoreOrderStatus(order.status);
  const canCancel = status === "PENDING_PAYMENT";
  const shippingStatus = tracking?.current_status === "IN_TRANSIT" || status === "SHIPPED" ? "IN_TRANSIT" : status === "COMPLETED" || status === "DELIVERED" ? "DELIVERED" : "NOT_SHIPPED";
  const timeline: OrderDetail["timeline"] = [{
    code: "ORDER_CREATED", label: "Đơn hàng đã được tạo", description: "Hệ thống đã ghi nhận thông tin đặt hàng.", occurredAt: order.created_at, state: tracking?.checkpoints.length ? "COMPLETED" : "CURRENT"
  }];
  tracking?.checkpoints.forEach((checkpoint, index) => timeline.push({
    code: `${checkpoint.status}-${index}`,
    label: checkpoint.location || "Cập nhật vận chuyển",
    description: checkpoint.description,
    occurredAt: checkpoint.timestamp,
    state: index === tracking.checkpoints.length - 1 ? "CURRENT" : "COMPLETED"
  }));
  return {
    orderId: order.order_id,
    orderNumber: order.order_id,
    source: "WEB_D2C",
    placedAt: order.created_at,
    status,
    statusLabel: STATUS_LABELS[status],
    statusDescription: status === "PENDING_PAYMENT" ? "Đơn đang chờ hoàn tất thanh toán." : status === "SHIPPED" ? "Đơn đang trên đường giao đến người nhận." : "Trạng thái mới nhất được đồng bộ từ hệ thống đơn hàng.",
    payment: { method: "BANK_TRANSFER", status: status === "PENDING_PAYMENT" ? "UNPAID" : "PAID", label: status === "PENDING_PAYMENT" ? "Chưa thanh toán" : "Đã thanh toán" },
    shipping: { methodName: tracking?.carrier_name ?? "Đơn vị vận chuyển", status: shippingStatus, label: shippingStatus === "IN_TRANSIT" ? "Đang vận chuyển" : shippingStatus === "DELIVERED" ? "Đã giao" : "Chưa bàn giao", trackingCode: tracking?.tracking_code ?? order.tracking_code, estimatedDelivery: null },
    lines: order.items.map((item, index) => ({ lineId: `${order.order_id}-${index + 1}`, productName: item.sku_code, skuLabel: `Mã SKU: ${item.sku_code}`, quantity: item.quantity, unitPriceVnd: item.price.units, lineSubtotalVnd: item.price.units * item.quantity })),
    pricing: { subtotalVnd: order.subtotal_amount.units, shippingFeeVnd: order.shipping_fee.units, discountVnd: order.discount_amount.units, totalVnd: order.final_amount.units, currency: "VND" },
    recipient: { fullName: order.shipping_address.recipient_name, phoneDisplay: order.shipping_address.phone_number, emailDisplay: "Không được API cung cấp", addressDisplay: [order.shipping_address.street_address, order.shipping_address.ward, order.shipping_address.province].filter(Boolean).join(", "), deliveryNote: null },
    timeline,
    cancellation: { canCancel, cancelBy: null, message: canCancel ? "Có thể hủy khi đơn vẫn đang chờ thanh toán." : "Đơn chỉ được tự hủy khi đang chờ thanh toán.", allowedReasons: canCancel ? [{ code: "CHANGED_MIND", label: "Thay đổi nhu cầu" }, { code: "WRONG_INFORMATION", label: "Thông tin đặt hàng chưa đúng" }, { code: "OTHER", label: "Lý do khác" }] : [] }
  };
}

export function mapCustomerCoreCancellation(orderId: string): CancelOrderResult {
  return { orderId, status: "CANCELLED", cancelledAt: new Date().toISOString(), reservationReleaseRequested: true, refundRequired: false, refundStatus: null, message: "Đơn hàng đã được hủy và yêu cầu giải phóng tồn kho đã được ghi nhận." };
}
