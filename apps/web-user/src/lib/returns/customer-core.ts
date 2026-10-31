import type { CreateReturnCaseInput, CreateReturnCaseResult, ReturnCaseDetail, ReturnCaseStatus, ReturnEligibility } from "./types";
import type { CustomerCoreOrderDetail } from "@/lib/orders/customer-core";

interface Money { currency_code?: string; units?: number; nanos?: number }

export interface CustomerCoreReturnTicket {
  return_id: string;
  order_id: string;
  status: "REQUESTED" | "INSPECTING" | "APPROVED" | "REFUNDED" | "REJECTED";
  reason: string;
  refund_amount?: Money | null;
  created_at: string;
}

const reasonOptions: ReturnEligibility["reasonOptions"] = [
  { code: "DAMAGED_IN_TRANSIT", label: "Hư hỏng khi vận chuyển" },
  { code: "WRONG_ITEM", label: "Giao sai sản phẩm" },
  { code: "QUALITY_ISSUE", label: "Vấn đề chất lượng" },
  { code: "OTHER", label: "Lý do khác" }
];

export function mapOrderToReturnEligibility(order: CustomerCoreOrderDetail): ReturnEligibility {
  return {
    orderId: order.order_id,
    orderNumber: order.order_id,
    deliveredAt: null,
    policyWindowEndsAt: null,
    eligible: true,
    policyMessage: "Điều kiện và thời hạn đổi trả sẽ được hệ thống kiểm tra khi gửi yêu cầu.",
    lines: order.items.map((item) => ({
      lineId: item.sku_code,
      productName: item.sku_code,
      skuLabel: `Mã SKU: ${item.sku_code}`,
      purchasedQty: item.quantity,
      alreadyClaimedQty: 0,
      maxReturnQty: item.quantity,
      unitPriceVnd: item.price.units,
      eligible: true
    })),
    reasonOptions,
    resolutionOptions: [],
    pickupSnapshot: {
      recipientName: order.shipping_address.recipient_name,
      phoneDisplay: order.shipping_address.phone_number,
      addressDisplay: [order.shipping_address.street_address, order.shipping_address.ward, order.shipping_address.province].filter(Boolean).join(", ")
    }
  };
}

export function toCustomerCoreReturnRequest(input: CreateReturnCaseInput) {
  const line = input.lines[0];
  const reasonLabel = reasonOptions.find((item) => item.code === input.reasonCode)?.label ?? input.reasonCode;
  return {
    order_id: input.orderId,
    sku_code: line.lineId,
    quantity: line.quantity,
    reason: `${reasonLabel}: ${input.details}`,
    evidence_media_urls: input.evidenceMediaUrls,
    refund_bank_code: input.refundBankCode,
    refund_account_number: input.refundAccountNumber,
    refund_account_holder: input.refundAccountHolder
  };
}

export function mapCustomerCoreReturnCreated(ticket: CustomerCoreReturnTicket): CreateReturnCaseResult {
  return {
    caseId: ticket.return_id,
    caseNumber: ticket.return_id,
    status: "RETURN_REQUESTED",
    casePath: `/after-sales/${encodeURIComponent(ticket.return_id)}`,
    createdAt: ticket.created_at,
    message: "Yêu cầu đổi trả đã được tiếp nhận."
  };
}

const statusMap: Record<CustomerCoreReturnTicket["status"], { status: ReturnCaseStatus; label: string; description: string }> = {
  REQUESTED: { status: "RETURN_REQUESTED", label: "Đã tiếp nhận", description: "Yêu cầu đang chờ bộ phận hậu mãi kiểm tra." },
  INSPECTING: { status: "REVIEWING", label: "Đang thẩm định", description: "Bộ phận hậu mãi đang kiểm tra yêu cầu và bằng chứng." },
  APPROVED: { status: "APPROVED", label: "Đã chấp thuận", description: "Yêu cầu đổi trả đã được chấp thuận." },
  REFUNDED: { status: "REFUNDED", label: "Đã hoàn tiền", description: "Khoản hoàn tiền đã được xử lý." },
  REJECTED: { status: "REJECTED", label: "Không được chấp thuận", description: "Yêu cầu không đáp ứng điều kiện đổi trả." }
};

export function mapCustomerCoreReturnDetail(ticket: CustomerCoreReturnTicket): ReturnCaseDetail {
  const current = statusMap[ticket.status] ?? statusMap.REQUESTED;
  const flow: CustomerCoreReturnTicket["status"][] = ["REQUESTED", "INSPECTING", ticket.status === "REJECTED" ? "REJECTED" : "APPROVED", "REFUNDED"];
  const currentIndex = flow.indexOf(ticket.status);
  return {
    caseId: ticket.return_id,
    caseNumber: ticket.return_id,
    orderId: ticket.order_id,
    orderNumber: ticket.order_id,
    createdAt: ticket.created_at,
    status: current.status,
    statusLabel: current.label,
    statusDescription: current.description,
    reason: ticket.reason,
    steps: flow.map((status, index) => ({
      code: status,
      label: statusMap[status].label,
      state: index < currentIndex ? "COMPLETED" : index === currentIndex ? "CURRENT" : "UPCOMING",
      occurredAt: index === 0 ? ticket.created_at : null
    })),
    decision: null,
    items: [],
    evidence: [],
    pickup: null,
    refund: {
      required: Boolean(ticket.refund_amount),
      methodLabel: ticket.refund_amount ? "Chuyển khoản ngân hàng" : null,
      amountVnd: ticket.refund_amount?.units ?? 0,
      status: ticket.status === "REFUNDED" ? "COMPLETED" : ticket.refund_amount ? "PENDING" : null
    },
    communications: []
  };
}
