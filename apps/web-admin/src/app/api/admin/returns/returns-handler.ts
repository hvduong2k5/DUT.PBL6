import { NextRequest, NextResponse } from "next/server";
import { parseAdminSession } from "@/lib/auth/types";
import { RETURN_CASE_STATUSES, type ReturnCaseDetail, type ReturnCaseList, type ReturnCaseStatus, type ReturnCaseSummary, type ReturnCaseType, type ReturnEvidenceState, type ReturnedGoodsState, type ReturnRefundState } from "@/lib/returns/types";
import { parseAdminReturnPath } from "@/lib/returns/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin return upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const nullableDate = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : null; };
const nullableString = (value: unknown) => typeof value === "string" && value.trim() ? value : null;
const nullableNumber = (value: unknown) => typeof value === "number" && Number.isFinite(value) ? Math.max(0, value) : null;
const STATUSES = new Set<ReturnCaseStatus>(RETURN_CASE_STATUSES);
const TYPES = new Set<ReturnCaseType>(["CANCELLATION", "RETURN_REFUND"]);
const EVIDENCE_STATES = new Set<ReturnEvidenceState>(["INCOMPLETE", "READY", "UNDER_REVIEW"]);
const GOODS_STATES = new Set<ReturnedGoodsState>(["NOT_REQUIRED", "AWAITING_RETURN", "IN_TRANSIT", "RECEIVED", "INSPECTING", "CLASSIFIED"]);
const REFUND_STATES = new Set<ReturnRefundState>(["NOT_REQUIRED", "NOT_CREATED", "PENDING", "COMPLETED", "FAILED"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) { return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-RETURN-${code}` }, { status }); }
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest) { const result = new Headers({ Accept: "application/json" }); const authorization = request.headers.get("authorization"); if (authorization) result.set("Authorization", authorization); const profile = mockProfile(); if (profile) result.set("X-Admin-Profile", profile); return result; }
async function upstream(request: NextRequest, path: string) { const response = await fetch(`${baseUrl()}/${path}`, { headers: headers(request), cache: "no-store", redirect: "manual" }); const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {}; if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {}); return record(payload) ? payload : {}; }
async function session(request: NextRequest) { const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" }); if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." }); const parsed = parseAdminSession(await response.json()); if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." }); return parsed; }
function caseStatus(value: unknown): ReturnCaseStatus { const result = string(value) as ReturnCaseStatus; return STATUSES.has(result) ? result : "RETURN_REQUESTED"; }
function caseType(value: unknown): ReturnCaseType { const result = string(value) as ReturnCaseType; return TYPES.has(result) ? result : "RETURN_REFUND"; }
function evidenceState(value: unknown): ReturnEvidenceState { const result = string(value) as ReturnEvidenceState; return EVIDENCE_STATES.has(result) ? result : "INCOMPLETE"; }
function goodsState(value: unknown): ReturnedGoodsState { const result = string(value) as ReturnedGoodsState; return GOODS_STATES.has(result) ? result : "NOT_REQUIRED"; }
function refundState(value: unknown): ReturnRefundState { const result = string(value) as ReturnRefundState; return REFUND_STATES.has(result) ? result : "NOT_REQUIRED"; }
function mapSummary(value: unknown, canViewFinancial: boolean): ReturnCaseSummary {
  const item = record(value) ? value : {};
  return { caseId: string(item.caseId), caseNumber: string(item.caseNumber), caseType: caseType(item.caseType), caseTypeLabel: string(item.caseTypeLabel), orderId: string(item.orderId), orderNumber: string(item.orderNumber), customerLabel: string(item.customerLabel), channelLabel: string(item.channelLabel), status: caseStatus(item.status), statusLabel: string(item.statusLabel), reasonLabel: string(item.reasonLabel), requestedItemCount: Math.max(0, number(item.requestedItemCount)), evidenceState: evidenceState(item.evidenceState), evidenceStateLabel: string(item.evidenceStateLabel), evidenceCount: Math.max(0, number(item.evidenceCount)), goodsState: goodsState(item.goodsState), goodsStateLabel: string(item.goodsStateLabel), refundState: refundState(item.refundState), refundStateLabel: string(item.refundStateLabel), refundAmountVnd: canViewFinancial ? nullableNumber(item.refundAmountVnd) : null, financialMasked: !canViewFinancial, assignedTo: nullableString(item.assignedTo), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1)) };
}
function mapList(payload: Record<string, unknown>, canViewFinancial: boolean): ReturnCaseList {
  const items = array(payload.items).map((item) => mapSummary(item, canViewFinancial)).filter((item) => item.caseId);
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | ReturnCaseStatus; return { status: raw === "ALL" || STATUSES.has(raw as ReturnCaseStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  if (!statusCounts.some((item) => item.status === "ALL")) statusCounts.unshift({ status: "ALL", label: "Tất cả", count: items.length });
  const caseTypes = array(payload.caseTypes).map((value) => { const item = record(value) ? value : {}; return { code: caseType(item.code), label: string(item.label) }; });
  return { items, statusCounts, caseTypes, calculatedAt: date(payload.calculatedAt) };
}
function mapDetail(payload: Record<string, unknown>, canViewEvidence: boolean, canViewFinancial: boolean): ReturnCaseDetail {
  const customer = record(payload.customer) ? payload.customer : {}; const order = record(payload.order) ? payload.order : {}; const payment = record(payload.payment) ? payload.payment : {}; const shipment = record(payload.shipment) ? payload.shipment : {}; const decision = record(payload.decision) ? payload.decision : null; const refund = record(payload.refund) ? payload.refund : {};
  return { ...mapSummary(payload, canViewFinancial), createdAt: date(payload.createdAt), customer: { displayName: string(customer.displayName), contactDisplay: string(customer.contactDisplay, "Đã ẩn") }, order: { status: string(order.status), statusLabel: string(order.statusLabel), placedAt: date(order.placedAt), deliveredAt: nullableDate(order.deliveredAt), totalVnd: Math.max(0, number(order.totalVnd)), currency: string(order.currency, "VND") }, lines: array(payload.lines).map((value) => { const item = record(value) ? value : {}; return { lineId: string(item.lineId), productName: string(item.productName), skuLabel: string(item.skuLabel), purchasedQuantity: Math.max(0, number(item.purchasedQuantity)), requestedQuantity: Math.max(0, number(item.requestedQuantity)), approvedQuantity: nullableNumber(item.approvedQuantity), unitPriceVnd: Math.max(0, number(item.unitPriceVnd)) }; }).filter((item) => item.lineId), payment: { paymentId: string(payment.paymentId), statusLabel: string(payment.statusLabel), methodLabel: string(payment.methodLabel), amountPaidVnd: canViewFinancial ? nullableNumber(payment.amountPaidVnd) : null, refundableVnd: canViewFinancial ? nullableNumber(payment.refundableVnd) : null, masked: !canViewFinancial }, shipment: { shipmentId: string(shipment.shipmentId), statusLabel: string(shipment.statusLabel), deliveredAt: nullableDate(shipment.deliveredAt), proofLabel: nullableString(shipment.proofLabel) }, evidences: canViewEvidence ? array(payload.evidences).map((value) => { const item = record(value) ? value : {}; return { evidenceId: string(item.evidenceId), fileName: string(item.fileName), mediaType: string(item.mediaType), storageStateLabel: string(item.storageStateLabel), submittedByLabel: string(item.submittedByLabel), submittedAt: date(item.submittedAt), originalPreserved: item.originalPreserved === true }; }).filter((item) => item.evidenceId) : [], evidenceMasked: !canViewEvidence, decision: decision ? { outcomeLabel: string(decision.outcomeLabel), decidedBy: nullableString(decision.decidedBy), decidedAt: nullableDate(decision.decidedAt), basis: nullableString(decision.basis) } : null, refund: { refundId: nullableString(refund.refundId), state: refundState(refund.state), stateLabel: string(refund.stateLabel), approvedAmountVnd: canViewFinancial ? nullableNumber(refund.approvedAmountVnd) : null, completedAmountVnd: canViewFinancial ? nullableNumber(refund.completedAmountVnd) : null, methodLabel: canViewFinancial ? nullableString(refund.methodLabel) : null, masked: !canViewFinancial }, history: array(payload.history).map((value) => { const item = record(value) ? value : {}; return { eventId: string(item.eventId), label: string(item.label), occurredAt: date(item.occurredAt), actorLabel: string(item.actorLabel), detail: nullableString(item.detail) }; }).filter((item) => item.eventId) };
}

export async function handleAdminReturns(request: NextRequest, path: string[]) {
  const route = parseAdminReturnPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin return route không tồn tại.", 404);
  if (request.method !== "GET") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("RETURN_CASE_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem Return Case.", 403);
    const financial = currentSession.permissions.includes("RETURN_FINANCIAL_VIEW");
    if (route.kind === "case-list") return NextResponse.json(mapList(await upstream(request, "admin/returns/cases"), financial));
    return NextResponse.json(mapDetail(await upstream(request, `admin/returns/cases/${encodeURIComponent(route.caseId)}`), currentSession.permissions.includes("RETURN_EVIDENCE_VIEW"), financial));
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_RETURN_ERROR", cause.body.message || "Không thể xử lý yêu cầu quản lý đổi trả.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_RETURN_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ đổi trả nội bộ.", 503);
  }
}
