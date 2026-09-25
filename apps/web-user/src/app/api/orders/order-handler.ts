import type {
  CancelOrderResult,
  GuestChallenge,
  GuestVerification,
  OrderDetail,
  OrderListItem,
  OrderListResponse,
  OrderPaymentStatus,
  OrderShippingStatus,
  OrderStatus
} from "@/lib/orders/types";
import { parseOrderRoute, validateCancelOrder, validateGuestChallenge, validateGuestOtp } from "@/lib/orders/validation";
import { NextRequest, NextResponse } from "next/server";

const ORDER_STATUSES = new Set<OrderStatus>(["PENDING_PAYMENT", "PAID", "CONFIRMED", "PROCESSING", "PACKED", "SHIPPED", "DELIVERED", "COMPLETED", "DELIVERY_FAILED", "EXPIRED", "CANCELLED"]);
const PAYMENT_STATUSES = new Set<OrderPaymentStatus>(["PENDING", "PAID", "UNPAID", "REFUND_PENDING", "COD_PENDING_COLLECTION"]);
const SHIPPING_STATUSES = new Set<OrderShippingStatus>(["NOT_SHIPPED", "READY", "IN_TRANSIT", "DELIVERED", "FAILED"]);

interface UpstreamError {
  code?: string;
  message?: string;
  requestId?: string;
  retryAfterSeconds?: number;
  errors?: Array<{ field: string; message: string }>;
}

class UpstreamHttpError extends Error {
  constructor(readonly status: number, readonly body: UpstreamError = {}) {
    super(body.message || `Order upstream returned ${status}`);
  }
}

function errorResponse(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = [], retryAfterSeconds?: number) {
  return NextResponse.json({ code, message, errors, retryAfterSeconds, requestId: `BFF-ORDER-${code}` }, { status });
}

function hasTrustedOrigin(request: NextRequest) {
  const origin = request.headers.get("origin");
  if (!origin) return true;
  try {
    const source = new URL(origin);
    return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host;
  } catch { return false; }
}

async function readJson(request: NextRequest): Promise<unknown> {
  try { return await request.json(); } catch { return undefined; }
}

async function upstream(path: string, method: string, request: NextRequest, scenario?: string, body?: object, idempotencyKey?: string) {
  const base = (process.env.ORDER_UPSTREAM_URL ?? "http://127.0.0.1:4017/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json" });
  const cookie = request.headers.get("cookie");
  if (cookie) headers.set("Cookie", cookie);
  if (body) headers.set("Content-Type", "application/json");
  if (idempotencyKey) headers.set("Idempotency-Key", idempotencyKey);
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  const response = await fetch(`${base}/${path}`, { method, headers, body: body ? JSON.stringify(body) : undefined, cache: "no-store", redirect: "manual" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as Record<string, unknown> : {};
  if (!response.ok) throw new UpstreamHttpError(response.status, payload);
  return { response, payload };
}

function text(value: unknown, fallback = "") { return typeof value === "string" ? value : fallback; }
function integer(value: unknown, fallback = 0) { return Number.isInteger(value) && Number(value) >= 0 ? Number(value) : fallback; }
function dateTime(value: unknown, fallback = new Date(0).toISOString()) { return typeof value === "string" && !Number.isNaN(Date.parse(value)) ? value : fallback; }
function nullableText(value: unknown) { return typeof value === "string" ? value : null; }

function publicListItem(value: unknown): OrderListItem | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const source = value as Record<string, unknown>;
  if (typeof source.orderId !== "string" || typeof source.orderNumber !== "string" || !ORDER_STATUSES.has(source.status as OrderStatus)) return null;
  const previews = Array.isArray(source.previewItems) ? source.previewItems.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const preview = item as Record<string, unknown>;
    return typeof preview.name === "string" ? [{ name: preview.name, quantity: integer(preview.quantity, 1) }] : [];
  }).slice(0, 4) : [];
  return {
    orderId: source.orderId,
    orderNumber: source.orderNumber,
    placedAt: dateTime(source.placedAt),
    status: source.status as OrderStatus,
    statusLabel: text(source.statusLabel, source.status as string),
    paymentStatus: PAYMENT_STATUSES.has(source.paymentStatus as OrderPaymentStatus) ? source.paymentStatus as OrderPaymentStatus : "UNPAID",
    shippingStatus: SHIPPING_STATUSES.has(source.shippingStatus as OrderShippingStatus) ? source.shippingStatus as OrderShippingStatus : "NOT_SHIPPED",
    totalVnd: integer(source.totalVnd),
    currency: "VND",
    itemCount: integer(source.itemCount),
    previewItems: previews,
    canCancel: source.canCancel === true,
    cancelPolicyMessage: text(source.cancelPolicyMessage)
  };
}

function publicDetail(value: Record<string, unknown>): OrderDetail {
  const payment = value.payment && typeof value.payment === "object" && !Array.isArray(value.payment) ? value.payment as Record<string, unknown> : {};
  const shipping = value.shipping && typeof value.shipping === "object" && !Array.isArray(value.shipping) ? value.shipping as Record<string, unknown> : {};
  const pricing = value.pricing && typeof value.pricing === "object" && !Array.isArray(value.pricing) ? value.pricing as Record<string, unknown> : {};
  const recipient = value.recipient && typeof value.recipient === "object" && !Array.isArray(value.recipient) ? value.recipient as Record<string, unknown> : {};
  const cancellation = value.cancellation && typeof value.cancellation === "object" && !Array.isArray(value.cancellation) ? value.cancellation as Record<string, unknown> : {};
  if (typeof value.orderId !== "string" || typeof value.orderNumber !== "string" || !ORDER_STATUSES.has(value.status as OrderStatus)) throw new Error("Invalid Order detail projection");
  const lines = Array.isArray(value.lines) ? value.lines.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const line = item as Record<string, unknown>;
    if (typeof line.lineId !== "string" || typeof line.productName !== "string") return [];
    return [{ lineId: line.lineId, productName: line.productName, skuLabel: text(line.skuLabel), quantity: integer(line.quantity, 1), unitPriceVnd: integer(line.unitPriceVnd), lineSubtotalVnd: integer(line.lineSubtotalVnd) }];
  }) : [];
  const timeline = Array.isArray(value.timeline) ? value.timeline.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const event = item as Record<string, unknown>;
    const state = ["COMPLETED", "CURRENT", "UPCOMING"].includes(text(event.state)) ? text(event.state) as "COMPLETED" | "CURRENT" | "UPCOMING" : "UPCOMING";
    if (typeof event.code !== "string" || typeof event.label !== "string") return [];
    return [{ code: event.code, label: event.label, description: text(event.description), occurredAt: dateTime(event.occurredAt), state }];
  }) : [];
  const reasons = Array.isArray(cancellation.allowedReasons) ? cancellation.allowedReasons.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const reason = item as Record<string, unknown>;
    return ["CHANGED_MIND", "WRONG_INFORMATION", "OTHER"].includes(text(reason.code)) && typeof reason.label === "string" ? [{ code: reason.code as "CHANGED_MIND" | "WRONG_INFORMATION" | "OTHER", label: reason.label }] : [];
  }) : [];
  return {
    orderId: value.orderId,
    orderNumber: value.orderNumber,
    source: "WEB_D2C",
    placedAt: dateTime(value.placedAt),
    status: value.status as OrderStatus,
    statusLabel: text(value.statusLabel, value.status as string),
    statusDescription: text(value.statusDescription),
    payment: { method: payment.method === "COD" ? "COD" : "BANK_TRANSFER", status: PAYMENT_STATUSES.has(payment.status as OrderPaymentStatus) ? payment.status as OrderPaymentStatus : "UNPAID", label: text(payment.label) },
    shipping: { methodName: text(shipping.methodName), status: SHIPPING_STATUSES.has(shipping.status as OrderShippingStatus) ? shipping.status as OrderShippingStatus : "NOT_SHIPPED", label: text(shipping.label), trackingCode: nullableText(shipping.trackingCode), estimatedDelivery: nullableText(shipping.estimatedDelivery) },
    lines,
    pricing: { subtotalVnd: integer(pricing.subtotalVnd), shippingFeeVnd: integer(pricing.shippingFeeVnd), discountVnd: integer(pricing.discountVnd), totalVnd: integer(pricing.totalVnd), currency: "VND" },
    recipient: { fullName: text(recipient.fullName), phoneDisplay: text(recipient.phoneDisplay), emailDisplay: text(recipient.emailDisplay), addressDisplay: text(recipient.addressDisplay), deliveryNote: nullableText(recipient.deliveryNote) },
    timeline,
    cancellation: { canCancel: cancellation.canCancel === true, cancelBy: cancellation.cancelBy === null ? null : dateTime(cancellation.cancelBy, ""), message: text(cancellation.message), allowedReasons: reasons }
  };
}

function publicCancellation(value: Record<string, unknown>, orderId: string): CancelOrderResult {
  return {
    orderId,
    status: "CANCELLED",
    cancelledAt: dateTime(value.cancelledAt, new Date().toISOString()),
    reservationReleaseRequested: value.reservationReleaseRequested === true,
    refundRequired: value.refundRequired === true,
    refundStatus: value.refundRequired === true ? "PENDING_HANDOFF" : null,
    message: text(value.message, "Yêu cầu hủy Order đã được ghi nhận.")
  };
}

export async function handleOrderRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parseOrderRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation) return errorResponse(route.error?.field === "path" ? "NOT_FOUND" : "INVALID_REQUEST", route.error?.message ?? "Yêu cầu Order không hợp lệ.", route.error?.field === "path" ? 404 : 400, route.error ? [route.error] : []);
  if (process.env.NODE_ENV === "production" && route.mockScenario) return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400);
  if (request.method !== "GET" && !hasTrustedOrigin(request)) return errorResponse("ORIGIN_NOT_ALLOWED", "Request origin không được phép.", 403);

  try {
    if (route.operation === "list" && route.filters) {
      const { payload } = await upstream("customers/me/orders", "GET", request, route.mockScenario);
      const items = (Array.isArray(payload.items) ? payload.items : []).map(publicListItem).filter((item): item is OrderListItem => Boolean(item));
      const query = route.filters.q.toLocaleLowerCase("vi");
      const filtered = items.filter((item) => (!query || item.orderNumber.toLocaleLowerCase("vi").includes(query) || item.previewItems.some((preview) => preview.name.toLocaleLowerCase("vi").includes(query))) && (route.filters!.status === "ALL" || item.status === route.filters!.status) && (!route.filters!.year || new Date(item.placedAt).getUTCFullYear() === route.filters!.year));
      const start = (route.filters.page - 1) * route.filters.pageSize;
      const customerSource = payload.customer && typeof payload.customer === "object" && !Array.isArray(payload.customer) ? payload.customer as Record<string, unknown> : {};
      const result: OrderListResponse = { items: filtered.slice(start, start + route.filters.pageSize), page: route.filters.page, pageSize: route.filters.pageSize, totalItems: filtered.length, totalPages: Math.max(1, Math.ceil(filtered.length / route.filters.pageSize)), customer: { displayName: text(customerSource.displayName, "Thành viên Tri Kỷ"), email: text(customerSource.email) } };
      return NextResponse.json(result);
    }

    if (route.operation === "detail" && route.orderId) {
      const { payload } = await upstream(`orders/${encodeURIComponent(route.orderId)}`, "GET", request, route.mockScenario);
      return NextResponse.json(publicDetail(payload));
    }

    if (route.operation === "cancel" && route.orderId) {
      const parsed = validateCancelOrder(await readJson(request));
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Thông tin hủy Order chưa hợp lệ.", 422, parsed.errors);
      const { payload } = await upstream(`orders/${encodeURIComponent(route.orderId)}/cancellations`, "POST", request, route.mockScenario, { reasonCode: parsed.data.reasonCode, note: parsed.data.note }, parsed.data.idempotencyKey);
      return NextResponse.json(publicCancellation(payload, route.orderId));
    }

    if (route.operation === "guest-challenge") {
      const parsed = validateGuestChallenge(await readJson(request));
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Thông tin tra cứu chưa hợp lệ.", 422, parsed.errors);
      const { payload } = await upstream("orders/guest-access/challenges", "POST", request, route.mockScenario, parsed.data);
      const result: GuestChallenge = { challengeId: text(payload.challengeId), maskedDestination: text(payload.maskedDestination), expiresAt: dateTime(payload.expiresAt), resendAfterSeconds: Math.min(300, Math.max(1, integer(payload.resendAfterSeconds, 60))), message: text(payload.message) };
      return NextResponse.json(result, { status: 202 });
    }

    const parsed = validateGuestOtp(await readJson(request));
    if (!parsed.data || !route.challengeId) return errorResponse("VALIDATION_ERROR", "Mã xác minh chưa hợp lệ.", 422, parsed.errors);
    const { payload, response: upstreamResponse } = await upstream(`orders/guest-access/challenges/${encodeURIComponent(route.challengeId)}/verify`, "POST", request, route.mockScenario, parsed.data);
    const result: GuestVerification = { orderId: text(payload.orderId), orderNumber: text(payload.orderNumber), orderPath: text(payload.orderPath), accessExpiresAt: dateTime(payload.accessExpiresAt) };
    const response = NextResponse.json(result);
    const setCookie = upstreamResponse.headers.get("set-cookie");
    if (setCookie) response.headers.append("Set-Cookie", setCookie);
    return response;
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) {
      return errorResponse(cause.body.code || "ORDER_ERROR", cause.body.message || "Không thể xử lý Order lúc này.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? [], cause.body.retryAfterSeconds);
    }
    return errorResponse("ORDER_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ Order. Hãy kiểm tra Mockoon hoặc API Gateway.", 503);
  }
}
