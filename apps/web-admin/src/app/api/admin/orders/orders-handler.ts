import { NextRequest, NextResponse } from "next/server";
import { parseAdminSession } from "@/lib/auth/types";
import { ADMIN_ORDER_STATUSES, type AdminOrderDetail, type AdminOrderList, type AdminOrderRelatedState, type AdminOrderSource, type AdminOrderSlaState, type AdminOrderStatus, type AdminOrderSummary, type AdminPaymentStatus } from "@/lib/orders/types";
import { parseAdminOrderPath } from "@/lib/orders/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin order upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const nullableDate = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : null; };
const nullableString = (value: unknown) => typeof value === "string" && value.trim() ? value : null;
const ORDER_STATUSES = new Set<AdminOrderStatus>(ADMIN_ORDER_STATUSES);
const PAYMENT_STATUSES = new Set<AdminPaymentStatus>(["PENDING", "PAID", "UNPAID", "REFUND_PENDING", "COD_PENDING_COLLECTION"]);
const SOURCES = new Set<AdminOrderSource>(["WEB_D2C", "B2B", "MARKETPLACE", "OFFLINE"]);
const SLA_STATES = new Set<AdminOrderSlaState>(["ON_TRACK", "AT_RISK", "OVERDUE", "COMPLETE"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) {
  return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-ORDER-${code}` }, { status });
}
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest) {
  const result = new Headers({ Accept: "application/json" });
  const authorization = request.headers.get("authorization");
  if (authorization) result.set("Authorization", authorization);
  const profile = mockProfile();
  if (profile) result.set("X-Admin-Profile", profile);
  return result;
}
async function upstream(request: NextRequest, path: string) {
  const response = await fetch(`${baseUrl()}/${path}`, { headers: headers(request), cache: "no-store", redirect: "manual" });
  const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {});
  return record(payload) ? payload : {};
}
async function session(request: NextRequest) {
  const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" });
  if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." });
  const parsed = parseAdminSession(await response.json());
  if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." });
  return parsed;
}
function orderStatus(value: unknown): AdminOrderStatus { const status = string(value) as AdminOrderStatus; return ORDER_STATUSES.has(status) ? status : "PENDING_PAYMENT"; }
function paymentStatus(value: unknown): AdminPaymentStatus { const status = string(value) as AdminPaymentStatus; return PAYMENT_STATUSES.has(status) ? status : "PENDING"; }
function source(value: unknown): AdminOrderSource { const result = string(value) as AdminOrderSource; return SOURCES.has(result) ? result : "WEB_D2C"; }
function slaState(value: unknown): AdminOrderSlaState { const result = string(value) as AdminOrderSlaState; return SLA_STATES.has(result) ? result : "ON_TRACK"; }
function mapSummary(value: unknown): AdminOrderSummary {
  const item = record(value) ? value : {};
  return {
    orderId: string(item.orderId), orderNumber: string(item.orderNumber), source: source(item.source), sourceLabel: string(item.sourceLabel), placedAt: date(item.placedAt),
    status: orderStatus(item.status), statusLabel: string(item.statusLabel), paymentStatus: paymentStatus(item.paymentStatus), paymentStatusLabel: string(item.paymentStatusLabel),
    totalVnd: Math.max(0, number(item.totalVnd)), currency: "VND", itemCount: Math.max(0, number(item.itemCount)), customerLabel: string(item.customerLabel),
    fulfillmentStageLabel: string(item.fulfillmentStageLabel), slaState: slaState(item.slaState), slaLabel: string(item.slaLabel), slaDueAt: nullableDate(item.slaDueAt),
    blockedReason: nullableString(item.blockedReason), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1))
  };
}
function mapList(payload: Record<string, unknown>): AdminOrderList {
  const items = array(payload.items).map(mapSummary).filter((item) => item.orderId);
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | AdminOrderStatus; return { status: raw === "ALL" || ORDER_STATUSES.has(raw as AdminOrderStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  if (!statusCounts.some((item) => item.status === "ALL")) statusCounts.unshift({ status: "ALL", label: "Tất cả", count: items.length });
  for (const status of ADMIN_ORDER_STATUSES) if (!statusCounts.some((item) => item.status === status) && items.some((item) => item.status === status)) statusCounts.push({ status, label: status, count: items.filter((item) => item.status === status).length });
  const sources = array(payload.sources).map((value) => { const item = record(value) ? value : {}; return { source: source(item.source), label: string(item.label) }; });
  const paymentStatuses = array(payload.paymentStatuses).map((value) => { const item = record(value) ? value : {}; return { status: paymentStatus(item.status), label: string(item.label) }; });
  return { items, statusCounts, sources, paymentStatuses, calculatedAt: date(payload.calculatedAt) };
}
function relatedState(value: unknown): AdminOrderRelatedState {
  const item = record(value) ? value : {};
  return { code: string(item.code), label: string(item.label), reference: nullableString(item.reference), updatedAt: nullableDate(item.updatedAt), note: nullableString(item.note) };
}
function mapDetail(payload: Record<string, unknown>, canViewSensitive: boolean): AdminOrderDetail {
  const recipient = record(payload.recipient) ? payload.recipient : {};
  const pricing = record(payload.pricing) ? payload.pricing : {};
  const payment = record(payload.payment) ? payload.payment : {};
  const shipping = record(payload.shipping) ? payload.shipping : {};
  const hidden = "Đã ẩn theo quyền truy cập";
  return {
    ...mapSummary(payload),
    recipient: canViewSensitive ? { displayName: string(recipient.displayName), phoneDisplay: string(recipient.phoneDisplay), emailDisplay: string(recipient.emailDisplay), addressDisplay: string(recipient.addressDisplay), deliveryNote: nullableString(recipient.deliveryNote), masked: false } : { displayName: string(recipient.displayName, "Khách hàng"), phoneDisplay: hidden, emailDisplay: hidden, addressDisplay: hidden, deliveryNote: null, masked: true },
    lines: array(payload.lines).map((value) => { const item = record(value) ? value : {}; return { lineId: string(item.lineId), productName: string(item.productName), skuCode: string(item.skuCode), skuLabel: string(item.skuLabel), quantity: Math.max(0, number(item.quantity)), unitPriceVnd: Math.max(0, number(item.unitPriceVnd)), lineSubtotalVnd: Math.max(0, number(item.lineSubtotalVnd)) }; }).filter((item) => item.lineId),
    pricing: { subtotalVnd: Math.max(0, number(pricing.subtotalVnd)), shippingFeeVnd: Math.max(0, number(pricing.shippingFeeVnd)), discountVnd: Math.max(0, number(pricing.discountVnd)), totalVnd: Math.max(0, number(pricing.totalVnd)) },
    payment: { ...relatedState(payment), methodLabel: string(payment.methodLabel) },
    reservation: relatedState(payload.reservation), packing: relatedState(payload.packing),
    shipping: { ...relatedState(shipping), carrierName: nullableString(shipping.carrierName), trackingCode: nullableString(shipping.trackingCode), estimatedDeliveryAt: nullableDate(shipping.estimatedDeliveryAt) },
    timeline: array(payload.timeline).map((value) => { const item = record(value) ? value : {}; const from = nullableString(item.fromStatus); return { eventId: string(item.eventId), fromStatus: from && ORDER_STATUSES.has(from as AdminOrderStatus) ? from as AdminOrderStatus : null, toStatus: orderStatus(item.toStatus), toStatusLabel: string(item.toStatusLabel), occurredAt: date(item.occurredAt), actorLabel: string(item.actorLabel), sourceLabel: string(item.sourceLabel), reason: nullableString(item.reason) }; }).filter((item) => item.eventId)
  };
}

export async function handleAdminOrders(request: NextRequest, path: string[]) {
  const route = parseAdminOrderPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin order route không tồn tại.", 404);
  if (request.method !== "GET") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("ORDER_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem đơn hàng.", 403);
    if (route.kind === "order-list") return NextResponse.json(mapList(await upstream(request, "admin/orders")));
    const canViewSensitive = currentSession.permissions.includes("ORDER_SENSITIVE_VIEW");
    return NextResponse.json(mapDetail(await upstream(request, `admin/orders/${encodeURIComponent(route.orderId)}`), canViewSensitive));
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_ORDER_ERROR", cause.body.message || "Không thể xử lý yêu cầu quản lý đơn hàng.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_ORDER_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ quản lý đơn hàng nội bộ.", 503);
  }
}
