import { NextRequest, NextResponse } from "next/server";
import { EmployeeAuthorizationError, requireEmployeeSession } from "@/features/employee/auth/server";
import { applyTrustedSessionCookie, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { usesIdentityBff } from "@/lib/auth/oidc-config";
import { PAYMENT_CASE_REASON_LABELS, PAYMENT_CASE_RESOLUTIONS, PAYMENT_CASE_RESOLUTION_LABELS, PAYMENT_CASE_STATUS_LABELS, type PaymentCaseDecision, type PaymentCaseDetail, type PaymentCaseList, type PaymentCaseSummary } from "@/features/employee/payments/types";
import { parseAdminPaymentPath } from "@/features/employee/payments/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin payment upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const nullableString = (value: unknown) => typeof value === "string" && value.trim() ? value : null;
const stringList = (value: unknown): string[] => array(value).filter((item): item is string => typeof item === "string" && item.length > 0);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) { return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-PAYMENT-${code}` }, { status }); }

// The live staff surface authenticates exactly like the orders live handler:
// in OIDC mode the browser session holds the Keycloak access token, and the
// owning API verifies that JWT itself — forward it as a Bearer credential to
// the public gateway base. The identity-bff deployment keeps the legacy
// session-cookie behaviour against ADMIN_API_UPSTREAM_URL.
function baseUrl() {
  if (usesIdentityBff()) return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, "");
  return (process.env.NEXT_PUBLIC_API_URL ?? process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, "");
}
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest) {
  const result = new Headers({ Accept: "application/json" });
  if (usesIdentityBff()) applyTrustedSessionCookie(result, request);
  else { const token = getTrustedAccessToken(request); if (token) result.set("Authorization", `Bearer ${token}`); }
  const profile = mockProfile(); if (profile) result.set("X-Admin-Profile", profile);
  return result;
}
async function upstream(request: NextRequest, path: string, init?: { method: string; body?: unknown; idempotencyKey?: string }) {
  const requestHeaders = headers(request);
  if (init?.body !== undefined) requestHeaders.set("Content-Type", "application/json");
  if (init?.idempotencyKey) requestHeaders.set("X-Idempotency-Key", init.idempotencyKey);
  const response = await fetch(`${baseUrl()}/${path}`, { method: init?.method ?? "GET", headers: requestHeaders, body: init?.body !== undefined ? JSON.stringify(init.body) : undefined, cache: "no-store", redirect: "manual" });
  const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) {
    const body = record(payload) ? payload : {};
    // Owning APIs answer with a nested {"error": {code, message}} envelope.
    const nested = record(body.error) ? body.error : {};
    throw new UpstreamHttpError(response.status, { code: string(nested.code) || string(body.code), message: string(nested.message) || string(body.message), errors: array(body.errors) as UpstreamError["errors"] });
  }
  return record(payload) ? payload : {};
}
async function session(request: NextRequest) { return requireEmployeeSession(request); }

// The upstream staff API returns the persistence PaymentCase shape
// (snake_case). Case views carry no bank or beneficiary data, so no
// field-level redaction applies at this layer; refund views are not proxied
// by this workspace.
function mapSummary(value: unknown): PaymentCaseSummary {
  const item = record(value) ? value : {};
  const reason = string(item.reason);
  const status = string(item.status);
  return {
    caseId: string(item.case_id),
    supportReference: string(item.support_reference),
    orderId: nullableString(item.order_id),
    receiptId: nullableString(item.receipt_id),
    reason,
    reasonLabel: PAYMENT_CASE_REASON_LABELS[reason] ?? reason,
    status,
    statusLabel: PAYMENT_CASE_STATUS_LABELS[status] ?? status,
    assignedTo: nullableString(item.assigned_to),
    version: Math.max(1, number(item.version, 1))
  };
}

function mapList(payload: Record<string, unknown>): PaymentCaseList {
  return {
    items: array(payload.items).map(mapSummary).filter((item) => item.caseId),
    nextCursor: nullableString(payload.next_cursor),
    calculatedAt: new Date().toISOString()
  };
}

function mapDetail(payload: Record<string, unknown>): PaymentCaseDetail {
  return {
    ...mapSummary(payload),
    observationId: nullableString(payload.observation_id),
    evidenceIds: stringList(payload.evidence_ids),
    refundIds: stringList(payload.refund_ids)
  };
}

function mapDecision(value: unknown): PaymentCaseDecision {
  const item = record(value) ? value : {};
  const resolution = string(item.resolution);
  return {
    decisionReference: string(item.decision_reference),
    caseId: string(item.case_id),
    resolution,
    resolutionLabel: PAYMENT_CASE_RESOLUTION_LABELS[resolution] ?? resolution,
    allowed: item.allowed === true,
    evidenceIds: stringList(item.evidence_ids),
    policyVersion: string(item.policy_version)
  };
}

function mapDecisionList(payload: Record<string, unknown>): { items: PaymentCaseDecision[] } {
  return { items: array(payload.items).map(mapDecision).filter((item) => item.decisionReference) };
}

export async function handleAdminPayments(request: NextRequest, path: string[]) {
  const route = parseAdminPaymentPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin payment route không tồn tại.", 404);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("PAYMENT_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem Payment.", 403);
    if (request.method === "GET") {
      if (route.kind === "case-list") return NextResponse.json(mapList(await upstream(request, "admin/payments/reconciliation-cases?limit=100")));
      if (route.kind !== "case-detail" && route.kind !== "case-decisions") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
      const canReadCases = currentSession.permissions.includes("PAYMENT_RECONCILIATION_VIEW");
      const canDecide = currentSession.permissions.includes("PAYMENT_CASE_DECIDE");
      if (!canReadCases && !(route.kind === "case-decisions" && canDecide)) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem hồ sơ đối soát.", 403);
      if (route.kind === "case-detail") return NextResponse.json(mapDetail(await upstream(request, `admin/payments/reconciliation-cases/${encodeURIComponent(route.caseId)}`)));
      return NextResponse.json(mapDecisionList(await upstream(request, `admin/payments/reconciliation-cases/${encodeURIComponent(route.caseId)}/decisions`)));
    }
    if (request.method !== "POST" || route.kind === "case-list" || route.kind === "case-detail") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
    const rawBody: unknown = await request.json().catch(() => null);
    const body = record(rawBody) ? rawBody : {};
    const casePath = `admin/payments/reconciliation-cases/${encodeURIComponent(route.caseId)}`;
    if (route.kind === "case-decisions") {
      if (!currentSession.permissions.includes("PAYMENT_CASE_DECIDE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền ra quyết định đối soát.", 403);
      const resolution = string(body.resolution);
      const reason = string(body.reason).trim();
      const evidenceIds = stringList(body.evidenceIds);
      if (!(PAYMENT_CASE_RESOLUTIONS as readonly string[]).includes(resolution) || !reason || reason.length > 1000 || !evidenceIds.length) return responseError("INVALID_REQUEST", "Quyết định cần hướng xử lý, lý do và ít nhất một tham chiếu bằng chứng.", 400);
      const created = await upstream(request, `${casePath}/decisions`, { method: "POST", body: { resolution, reason, evidence_ids: evidenceIds } });
      return NextResponse.json(mapDecision(created), { status: 201 });
    }
    // assign and resolve share the command envelope (expected version,
    // reason, idempotency key); order-service stays the authority on
    // permissions, freshness (auth_time) and case state.
    if (!currentSession.permissions.includes("PAYMENT_CASE_RESOLVE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xử lý hồ sơ đối soát.", 403);
    const expectedVersion = number(body.expectedVersion);
    const reason = string(body.reason).trim();
    const idempotencyKey = string(body.idempotencyKey).trim();
    if (!Number.isInteger(expectedVersion) || expectedVersion < 1 || !reason || reason.length > 1000 || !idempotencyKey) return responseError("INVALID_REQUEST", "Yêu cầu xử lý hồ sơ thiếu phiên bản, lý do hoặc khóa idempotency.", 400);
    if (route.kind === "case-assign") {
      const assigneeId = string(body.assigneeId).trim();
      if (!assigneeId) return responseError("INVALID_REQUEST", "Yêu cầu tiếp nhận thiếu người được giao hồ sơ.", 400);
      return NextResponse.json(mapDetail(await upstream(request, `${casePath}/assign`, { method: "POST", idempotencyKey, body: { expected_version: expectedVersion, reason, assignee_id: assigneeId } })));
    }
    const resolution = string(body.resolution);
    const decisionReference = string(body.decisionReference).trim();
    const evidenceIds = stringList(body.evidenceIds);
    if (!(PAYMENT_CASE_RESOLUTIONS as readonly string[]).includes(resolution) || !decisionReference || !evidenceIds.length) return responseError("INVALID_REQUEST", "Yêu cầu hoàn tất cần hướng xử lý, mã quyết định và bằng chứng khớp quyết định.", 400);
    return NextResponse.json(mapDetail(await upstream(request, `${casePath}/resolve`, { method: "POST", idempotencyKey, body: { expected_version: expectedVersion, reason, resolution, decision_reference: decisionReference, evidence_ids: evidenceIds } })));
  } catch (cause) { if (cause instanceof EmployeeAuthorizationError) return responseError(cause.code, cause.message, cause.status);
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_PAYMENT_ERROR", cause.body.message || "Không thể xử lý yêu cầu vận hành thanh toán.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_PAYMENT_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ thanh toán nội bộ.", 503);
  }
}
