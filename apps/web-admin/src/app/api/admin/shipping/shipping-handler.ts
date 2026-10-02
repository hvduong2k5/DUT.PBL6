import { NextRequest, NextResponse } from "next/server";
import { parseAdminSession } from "@/lib/auth/types";
import { SHIPMENT_STATUSES, type ShipmentDetail, type ShipmentExceptionState, type ShipmentList, type ShipmentMappingResult, type ShipmentStatus, type ShipmentSummary } from "@/lib/shipping/types";
import { parseAdminShippingPath } from "@/lib/shipping/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin shipping upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const nullableDate = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : null; };
const nullableString = (value: unknown) => typeof value === "string" && value.trim() ? value : null;
const nullableNumber = (value: unknown) => typeof value === "number" && Number.isFinite(value) ? value : null;
const STATUSES = new Set<ShipmentStatus>(SHIPMENT_STATUSES);
const EXCEPTIONS = new Set<ShipmentExceptionState>(["NONE", "ATTENTION", "BLOCKED"]);
const MAPPING_RESULTS = new Set<ShipmentMappingResult>(["APPLIED", "IGNORED_DUPLICATE", "IGNORED_STALE", "UNMAPPED", "REJECTED_SOURCE"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) { return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-SHIPPING-${code}` }, { status }); }
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest) { const result = new Headers({ Accept: "application/json" }); const authorization = request.headers.get("authorization"); if (authorization) result.set("Authorization", authorization); const profile = mockProfile(); if (profile) result.set("X-Admin-Profile", profile); return result; }
async function upstream(request: NextRequest, path: string) { const response = await fetch(`${baseUrl()}/${path}`, { headers: headers(request), cache: "no-store", redirect: "manual" }); const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {}; if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {}); return record(payload) ? payload : {}; }
async function session(request: NextRequest) { const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" }); if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." }); const parsed = parseAdminSession(await response.json()); if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." }); return parsed; }
function shipmentStatus(value: unknown): ShipmentStatus { const result = string(value) as ShipmentStatus; return STATUSES.has(result) ? result : "CREATING"; }
function exceptionState(value: unknown): ShipmentExceptionState { const result = string(value) as ShipmentExceptionState; return EXCEPTIONS.has(result) ? result : "NONE"; }
function mappingResult(value: unknown): ShipmentMappingResult { const result = string(value) as ShipmentMappingResult; return MAPPING_RESULTS.has(result) ? result : "UNMAPPED"; }
function mapSummary(value: unknown): ShipmentSummary {
  const item = record(value) ? value : {}; const assignee = record(item.deliveryAssignee) ? item.deliveryAssignee : null;
  return { shipmentId: string(item.shipmentId), shipmentNumber: string(item.shipmentNumber), orderId: string(item.orderId), orderNumber: string(item.orderNumber), orderSource: string(item.orderSource), orderSourceLabel: string(item.orderSourceLabel), packageReference: string(item.packageReference), providerCode: string(item.providerCode), providerLabel: string(item.providerLabel), serviceLabel: string(item.serviceLabel), trackingCode: nullableString(item.trackingCode), status: shipmentStatus(item.status), statusLabel: string(item.statusLabel), deliveryAssignee: assignee && string(assignee.employeeId) ? { employeeId: string(assignee.employeeId), displayName: string(assignee.displayName) } : null, orderStatus: string(item.orderStatus), orderStatusLabel: string(item.orderStatusLabel), codAmountVnd: Math.max(0, number(item.codAmountVnd)), codStatusLabel: string(item.codStatusLabel), exceptionState: exceptionState(item.exceptionState), exceptionLabel: string(item.exceptionLabel), exceptionReason: nullableString(item.exceptionReason), estimatedDeliveryAt: nullableDate(item.estimatedDeliveryAt), lastEventAt: nullableDate(item.lastEventAt), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1)) };
}
function mapList(payload: Record<string, unknown>): ShipmentList {
  const items = array(payload.items).map(mapSummary).filter((item) => item.shipmentId);
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | ShipmentStatus; return { status: raw === "ALL" || STATUSES.has(raw as ShipmentStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  if (!statusCounts.some((item) => item.status === "ALL")) statusCounts.unshift({ status: "ALL", label: "Tất cả", count: items.length });
  const providers = array(payload.providers).map((value) => { const item = record(value) ? value : {}; return { code: string(item.code), label: string(item.label) }; }).filter((item) => item.code);
  const sources = array(payload.sources).map((value) => { const item = record(value) ? value : {}; return { code: string(item.code), label: string(item.label) }; }).filter((item) => item.code);
  return { items, statusCounts, providers, sources, calculatedAt: date(payload.calculatedAt) };
}
function mapDetail(payload: Record<string, unknown>, canViewSensitive: boolean): ShipmentDetail {
  const recipient = record(payload.recipient) ? payload.recipient : {}; const packageData = record(payload.package) ? payload.package : {}; const fees = record(payload.fees) ? payload.fees : {}; const handover = record(payload.handover) ? payload.handover : {}; const hidden = "Đã ẩn theo quyền truy cập";
  return { ...mapSummary(payload),
    recipient: canViewSensitive ? { displayName: string(recipient.displayName), phoneDisplay: string(recipient.phoneDisplay), addressDisplay: string(recipient.addressDisplay), masked: false } : { displayName: string(recipient.displayName, "Người nhận"), phoneDisplay: hidden, addressDisplay: hidden, masked: true },
    package: { weightGrams: Math.max(0, number(packageData.weightGrams)), lengthCm: nullableNumber(packageData.lengthCm), widthCm: nullableNumber(packageData.widthCm), heightCm: nullableNumber(packageData.heightCm), packedAt: nullableDate(packageData.packedAt) },
    fees: { checkoutFeeVnd: Math.max(0, number(fees.checkoutFeeVnd)), providerEstimatedFeeVnd: Math.max(0, number(fees.providerEstimatedFeeVnd)), actualFeeVnd: nullableNumber(fees.actualFeeVnd) },
    handover: { ready: handover.ready === true, handedOverAt: nullableDate(handover.handedOverAt), confirmedBy: nullableString(handover.confirmedBy), note: nullableString(handover.note) },
    trackingEvents: array(payload.trackingEvents).map((value) => { const item = record(value) ? value : {}; const mapped = nullableString(item.mappedStatus); return { eventId: string(item.eventId), providerStatusCode: string(item.providerStatusCode), providerStatusLabel: string(item.providerStatusLabel), mappedStatus: mapped && STATUSES.has(mapped as ShipmentStatus) ? mapped as ShipmentStatus : null, mappedStatusLabel: string(item.mappedStatusLabel), mappingResult: mappingResult(item.mappingResult), mappingResultLabel: string(item.mappingResultLabel), eventAt: date(item.eventAt), receivedAt: date(item.receivedAt), sourceLabel: string(item.sourceLabel), publicDescription: string(item.publicDescription), internalNote: nullableString(item.internalNote) }; }).filter((item) => item.eventId)
  };
}

export async function handleAdminShipping(request: NextRequest, path: string[]) {
  const route = parseAdminShippingPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin shipping route không tồn tại.", 404);
  if (request.method !== "GET") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("SHIPMENT_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem Shipment.", 403);
    if (route.kind === "shipment-list" || route.kind === "shipment-detail") {
      if (!currentSession.permissions.includes("SHIPMENT_MANAGEMENT_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền truy cập trang quản lý giao vận.", 403);
      if (route.kind === "shipment-list") return NextResponse.json(mapList(await upstream(request, "admin/shipping/shipments")));
      return NextResponse.json(mapDetail(await upstream(request, `admin/shipping/shipments/${encodeURIComponent(route.shipmentId)}`), currentSession.permissions.includes("SHIPMENT_SENSITIVE_VIEW")));
    }
    if (!currentSession.permissions.includes("DELIVERY_WORKBENCH_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền truy cập Workbench giao hàng.", 403);
    if (!currentSession.permissions.includes("SHIPMENT_SENSITIVE_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Công việc giao hàng yêu cầu quyền xem thông tin người nhận.", 403);
    if (!currentSession.scopes.some((scope) => scope.resource === "SHIPMENT" && (scope.level === "ASSIGNED_ONLY" || scope.level === "ALL"))) return responseError("SCOPE_FORBIDDEN", "Nhân viên không có phạm vi Shipment được phân công.", 403);
    if (route.kind === "workbench-list") {
      const result = mapList(await upstream(request, "admin/shipping/workbench/shipments"));
      return NextResponse.json({ ...result, items: result.items.filter((item) => item.deliveryAssignee?.employeeId === currentSession.employee.employeeId) });
    }
    const detail = mapDetail(await upstream(request, `admin/shipping/workbench/shipments/${encodeURIComponent(route.shipmentId)}`), true);
    if (detail.deliveryAssignee?.employeeId !== currentSession.employee.employeeId) return responseError("SHIPMENT_NOT_ASSIGNED", "Shipment này không được phân công cho bạn.", 403);
    return NextResponse.json(detail);
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_SHIPPING_ERROR", cause.body.message || "Không thể xử lý yêu cầu quản lý giao vận.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_SHIPPING_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ giao vận nội bộ.", 503);
  }
}
