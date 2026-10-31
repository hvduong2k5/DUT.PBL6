import { NextRequest, NextResponse } from "next/server";
import { parseAdminSession } from "@/lib/auth/types";
import { PAYMENT_STATUSES, RECONCILIATION_RESULTS, type PaymentCaseDetail, type PaymentCaseList, type PaymentCaseSummary, type PaymentRiskState, type PaymentStatus, type ReconciliationResult } from "@/lib/payments/types";
import { parseAdminPaymentPath } from "@/lib/payments/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin payment upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const nullableDate = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : null; };
const nullableString = (value: unknown) => typeof value === "string" && value.trim() ? value : null;
const STATUSES = new Set<PaymentStatus>(PAYMENT_STATUSES);
const RESULTS = new Set<ReconciliationResult>(RECONCILIATION_RESULTS);
const RISKS = new Set<PaymentRiskState>(["NONE", "ATTENTION", "BLOCKED"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) { return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-PAYMENT-${code}` }, { status }); }
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest) { const result = new Headers({ Accept: "application/json" }); const authorization = request.headers.get("authorization"); if (authorization) result.set("Authorization", authorization); const profile = mockProfile(); if (profile) result.set("X-Admin-Profile", profile); return result; }
async function upstream(request: NextRequest, path: string) { const response = await fetch(`${baseUrl()}/${path}`, { headers: headers(request), cache: "no-store", redirect: "manual" }); const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {}; if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {}); return record(payload) ? payload : {}; }
async function session(request: NextRequest) { const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" }); if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." }); const parsed = parseAdminSession(await response.json()); if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." }); return parsed; }
function paymentStatus(value: unknown): PaymentStatus { const result = string(value) as PaymentStatus; return STATUSES.has(result) ? result : "PENDING"; }
function reconciliationResult(value: unknown): ReconciliationResult { const result = string(value) as ReconciliationResult; return RESULTS.has(result) ? result : "PENDING"; }
function riskState(value: unknown): PaymentRiskState { const result = string(value) as PaymentRiskState; return RISKS.has(result) ? result : "NONE"; }
function maskReference(value: unknown, allowed: boolean) { const raw = nullableString(value); if (!raw) return null; if (allowed) return raw; const tail = raw.replace(/\s/gu, "").slice(-4); return `••••${tail}`; }
function mapSummary(value: unknown): PaymentCaseSummary {
  const item = record(value) ? value : {};
  return { caseId: string(item.caseId), caseNumber: string(item.caseNumber), orderId: string(item.orderId), orderNumber: string(item.orderNumber), orderStatus: string(item.orderStatus), orderStatusLabel: string(item.orderStatusLabel), paymentId: string(item.paymentId), methodCode: string(item.methodCode), methodLabel: string(item.methodLabel), paymentStatus: paymentStatus(item.paymentStatus), paymentStatusLabel: string(item.paymentStatusLabel), amountDueVnd: Math.max(0, number(item.amountDueVnd)), amountConfirmedVnd: Math.max(0, number(item.amountConfirmedVnd)), transactionCount: Math.max(0, number(item.transactionCount)), result: reconciliationResult(item.result), resultLabel: string(item.resultLabel), riskState: riskState(item.riskState), riskLabel: string(item.riskLabel), riskReason: nullableString(item.riskReason), referenceDisplay: nullableString(item.referenceDisplay), lastTransactionAt: nullableDate(item.lastTransactionAt), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1)) };
}
function mapList(payload: Record<string, unknown>): PaymentCaseList {
  const items = array(payload.items).map(mapSummary).filter((item) => item.caseId);
  const resultCounts = array(payload.resultCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.result) as "ALL" | ReconciliationResult; return { result: raw === "ALL" || RESULTS.has(raw as ReconciliationResult) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  if (!resultCounts.some((item) => item.result === "ALL")) resultCounts.unshift({ result: "ALL", label: "Tất cả", count: items.length });
  const methods = array(payload.methods).map((value) => { const item = record(value) ? value : {}; return { code: string(item.code), label: string(item.label) }; }).filter((item) => item.code);
  const paymentStatuses = array(payload.paymentStatuses).map((value) => { const item = record(value) ? value : {}; return { code: paymentStatus(item.code), label: string(item.label) }; });
  return { items, resultCounts, methods, paymentStatuses, calculatedAt: date(payload.calculatedAt) };
}
function mapDetail(payload: Record<string, unknown>, canViewSensitive: boolean): PaymentCaseDetail {
  const order = record(payload.order) ? payload.order : {};
  return { ...mapSummary(payload),
    order: { placedAt: date(order.placedAt), totalVnd: Math.max(0, number(order.totalVnd)), currency: string(order.currency, "VND") },
    attempts: array(payload.attempts).map((value) => { const item = record(value) ? value : {}; return { attemptId: string(item.attemptId), status: string(item.status), statusLabel: string(item.statusLabel), amountVnd: Math.max(0, number(item.amountVnd)), createdAt: date(item.createdAt), expiresAt: nullableDate(item.expiresAt) }; }).filter((item) => item.attemptId),
    transactions: array(payload.transactions).map((value) => { const item = record(value) ? value : {}; return { transactionId: string(item.transactionId), providerLabel: string(item.providerLabel), providerReferenceDisplay: maskReference(item.providerReference, canViewSensitive) ?? "Không có", bankReferenceDisplay: maskReference(item.bankReference, canViewSensitive), receivedAmountVnd: Math.max(0, number(item.receivedAmountVnd)), currency: string(item.currency, "VND"), providerStatusCode: string(item.providerStatusCode), verified: item.verified === true, occurredAt: date(item.occurredAt), receivedAt: date(item.receivedAt), duplicate: item.duplicate === true, masked: !canViewSensitive }; }).filter((item) => item.transactionId),
    comparisons: array(payload.comparisons).map((value) => { const item = record(value) ? value : {}; return { field: string(item.field), label: string(item.label), expectedDisplay: string(item.expectedDisplay), actualDisplay: string(item.actualDisplay), matched: item.matched === true }; }).filter((item) => item.field),
    history: array(payload.history).map((value) => { const item = record(value) ? value : {}; return { eventId: string(item.eventId), label: string(item.label), occurredAt: date(item.occurredAt), actorLabel: string(item.actorLabel), detail: nullableString(item.detail) }; }).filter((item) => item.eventId),
    originalEvidencePreserved: payload.originalEvidencePreserved === true
  };
}

export async function handleAdminPayments(request: NextRequest, path: string[]) {
  const route = parseAdminPaymentPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin payment route không tồn tại.", 404);
  if (request.method !== "GET") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("PAYMENT_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem Payment.", 403);
    if (route.kind === "case-list") return NextResponse.json(mapList(await upstream(request, "admin/payments/cases")));
    if (!currentSession.permissions.includes("PAYMENT_RECONCILIATION_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem hồ sơ đối soát.", 403);
    return NextResponse.json(mapDetail(await upstream(request, `admin/payments/cases/${encodeURIComponent(route.caseId)}`), currentSession.permissions.includes("PAYMENT_SENSITIVE_VIEW")));
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_PAYMENT_ERROR", cause.body.message || "Không thể xử lý yêu cầu vận hành thanh toán.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_PAYMENT_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ thanh toán nội bộ.", 503);
  }
}
