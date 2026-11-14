import { createHash } from "node:crypto";
import { NextRequest, NextResponse } from "next/server";
import { EmployeeAuthorizationError, requireEmployeeSession } from "@/features/employee/auth/server";
import { applyTrustedSessionCookie, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { usesIdentityBff } from "@/lib/auth/oidc-config";
import { PAYMENT_CASE_REASON_LABELS, PAYMENT_CASE_RESOLUTIONS, PAYMENT_CASE_RESOLUTION_LABELS, PAYMENT_CASE_STATUS_LABELS, PAYMENT_REFUND_EXECUTION_LABELS, PAYMENT_REFUND_STATUS_LABELS, REFUND_APPROVAL_STATE_LABELS, REFUND_EVIDENCE_DECISION_LABELS, type PaymentCaseDecision, type PaymentCaseDetail, type PaymentCaseList, type PaymentCaseSummary, type PaymentRefundList, type PaymentRefundSummary, type RefundApprovalItem, type RefundCertificateItem, type RefundStartInstructionView } from "@/features/employee/payments/types";
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

// The upstream staff API returns persistence views (snake_case).
// Case views carry no bank data. Refund views are masked upstream; the
// one place full beneficiary values cross this layer is the verify
// call's bank evidence, assembled server-side from the sealed
// certificate and never sent to the browser.
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
function mapRefund(value: unknown): PaymentRefundSummary {
  const item = record(value) ? value : {};
  const status = string(item.status);
  const executionState = string(item.execution_state);
  return {
    refundId: string(item.refund_id),
    receiptId: string(item.receipt_id),
    orderId: nullableString(item.order_id),
    approvalId: string(item.approval_id),
    amount: number(item.amount),
    currency: string(item.currency, "VND"),
    status,
    statusLabel: PAYMENT_REFUND_STATUS_LABELS[status] ?? status,
    executionState,
    executionLabel: PAYMENT_REFUND_EXECUTION_LABELS[executionState] ?? executionState,
    obligationState: string(item.obligation_state),
    executionMode: string(item.execution_mode),
    version: Math.max(1, number(item.version, 1)),
    attemptNumber: number(item.attempt_number),
    attemptId: nullableString(item.attempt_id),
    claimOwner: nullableString(item.claim_owner),
    claimExpiresAt: nullableString(item.claim_expires_at),
    beneficiaryMasked: string(item.beneficiary_masked),
    evidenceIds: stringList(item.evidence_ids)
  };
}

function mapRefundList(payload: Record<string, unknown>): PaymentRefundList {
  return { items: array(payload.items).map(mapRefund).filter((item) => item.refundId), nextCursor: nullableString(payload.next_cursor) };
}

function mapApproval(value: unknown): RefundApprovalItem {
  const item = record(value) ? value : {};
  const state = string(item.state);
  return {
    approvalId: string(item.approval_id),
    receiptId: string(item.receipt_id),
    orderId: nullableString(item.order_id),
    amount: number(item.amount),
    currency: string(item.currency, "VND"),
    state,
    stateLabel: REFUND_APPROVAL_STATE_LABELS[state] ?? state,
    beneficiaryMasked: string(item.beneficiary_masked),
    bank: string(item.beneficiary_bank),
    holder: string(item.beneficiary_holder),
    approverId: string(item.approver_id),
    version: Math.max(1, number(item.version, 1)),
    withdrawalReference: nullableString(item.withdrawal_reference),
    reason: string(item.reason),
    createdAt: string(item.created_at)
  };
}

function mapApprovalList(payload: Record<string, unknown>): { items: RefundApprovalItem[] } {
  return { items: array(payload.items).map(mapApproval).filter((item) => item.approvalId) };
}

// Certificates carry the full beneficiary account upstream; only the
// masked form ever leaves this server layer. The full values are used
// exactly once, server-side, to build the verify call's bank evidence.
function maskAccount(account: string) { return account.length >= 4 ? `****${account.slice(-4)}` : "****"; }

function mapCertificate(value: unknown): RefundCertificateItem {
  const item = record(value) ? value : {};
  const decision = string(item.decision);
  return {
    reference: string(item.reference),
    refundId: string(item.refund_id),
    attemptId: string(item.attempt_id),
    decision,
    decisionLabel: REFUND_EVIDENCE_DECISION_LABELS[decision] ?? decision,
    definitive: item.definitive === true,
    amount: number(item.amount),
    currency: string(item.currency, "VND"),
    beneficiaryMasked: maskAccount(string(item.beneficiary_account)),
    bank: string(item.bank),
    holder: string(item.beneficiary_holder),
    bankReference: string(item.bank_reference),
    occurredAt: string(item.occurred_at),
    sha256: string(item.sha256)
  };
}

function mapCertificateList(payload: Record<string, unknown>): { items: RefundCertificateItem[] } {
  return { items: array(payload.items).map(mapCertificate).filter((item) => item.reference) };
}

function mapInstruction(payload: Record<string, unknown>): RefundStartInstructionView {
  return {
    refund: mapRefund(payload.refund),
    attemptId: string(payload.attempt_id),
    amount: number(payload.amount),
    currency: string(payload.currency, "VND"),
    bankCode: string(payload.bank_code),
    account: string(payload.beneficiary_account),
    holder: string(payload.beneficiary_holder),
    transferContent: string(payload.transfer_content),
    startedAt: string(payload.started_at)
  };
}

// Mirrors the order-service hash of the canonical beneficiary snapshot
// (Go marshals the string map with sorted keys). Bank codes, account
// numbers and holder names in this flow are plain ASCII, where Go's
// JSON escaping and JSON.stringify agree.
function beneficiarySnapshotHash(account: string, bank: string, holder: string) {
  const raw = `{"account_holder":${JSON.stringify(holder)},"account_number":${JSON.stringify(account)},"bank_code":${JSON.stringify(bank)}}`;
  return createHash("sha256").update(raw).digest("hex");
}


export async function handleAdminPayments(request: NextRequest, path: string[]) {
  const route = parseAdminPaymentPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin payment route không tồn tại.", 404);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("PAYMENT_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem Payment.", 403);
    if (request.method === "GET") {
      const canReadRefunds = currentSession.permissions.includes("PAYMENT_REFUND_VIEW") && currentSession.permissions.includes("PAYMENT_SENSITIVE_VIEW");
      if (route.kind === "refund-list" || route.kind === "refund-detail") {
        if (!canReadRefunds) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem hoàn tiền.", 403);
        if (route.kind === "refund-list") return NextResponse.json(mapRefundList(await upstream(request, "admin/payments/refunds?limit=100")));
        return NextResponse.json(mapRefund(await upstream(request, `admin/payments/refunds/${encodeURIComponent(route.refundId)}`)));
      }
      if (route.kind === "refund-approvals") {
        if (!currentSession.permissions.includes("PAYMENT_REFUND_APPROVE") && !canReadRefunds) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem phê duyệt hoàn tiền.", 403);
        const receiptId = request.nextUrl.searchParams.get("receiptId");
        return NextResponse.json(mapApprovalList(await upstream(request, `admin/payments/refund-approvals${receiptId ? `?receipt_id=${encodeURIComponent(receiptId)}` : ""}`)));
      }
      if (route.kind === "refund-evidence-records") {
        if (!currentSession.permissions.includes("PAYMENT_REFUND_VERIFY") && !currentSession.permissions.includes("PAYMENT_REFUND_EXECUTE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem chứng nhận sao kê.", 403);
        const refundId = request.nextUrl.searchParams.get("refundId") ?? "";
        return NextResponse.json(mapCertificateList(await upstream(request, `admin/payments/refund-evidence?refund_id=${encodeURIComponent(refundId)}`)));
      }
      if (route.kind === "case-list") return NextResponse.json(mapList(await upstream(request, "admin/payments/reconciliation-cases?limit=100")));
      if (route.kind !== "case-detail" && route.kind !== "case-decisions") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
      const canReadCases = currentSession.permissions.includes("PAYMENT_RECONCILIATION_VIEW");
      const canDecide = currentSession.permissions.includes("PAYMENT_CASE_DECIDE");
      if (!canReadCases && !(route.kind === "case-decisions" && canDecide)) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem hồ sơ đối soát.", 403);
      if (route.kind === "case-detail") return NextResponse.json(mapDetail(await upstream(request, `admin/payments/reconciliation-cases/${encodeURIComponent(route.caseId)}`)));
      return NextResponse.json(mapDecisionList(await upstream(request, `admin/payments/reconciliation-cases/${encodeURIComponent(route.caseId)}/decisions`)));
    }
    if (request.method !== "POST" || route.kind === "case-list" || route.kind === "case-detail" || route.kind === "refund-list" || route.kind === "refund-detail") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
    const rawBody: unknown = await request.json().catch(() => null);
    const body = record(rawBody) ? rawBody : {};
    const casePath = `admin/payments/reconciliation-cases/${encodeURIComponent(path[1] ?? "")}`;
    if (route.kind === "case-decisions") {
      if (!currentSession.permissions.includes("PAYMENT_CASE_DECIDE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền ra quyết định đối soát.", 403);
      const resolution = string(body.resolution);
      const reason = string(body.reason).trim();
      const evidenceIds = stringList(body.evidenceIds);
      if (!(PAYMENT_CASE_RESOLUTIONS as readonly string[]).includes(resolution) || !reason || reason.length > 1000 || !evidenceIds.length) return responseError("INVALID_REQUEST", "Quyết định cần hướng xử lý, lý do và ít nhất một tham chiếu bằng chứng.", 400);
      const created = await upstream(request, `${casePath}/decisions`, { method: "POST", body: { resolution, reason, evidence_ids: evidenceIds } });
      return NextResponse.json(mapDecision(created), { status: 201 });
    }
    if (route.kind === "refund-approvals") {
      if (!currentSession.permissions.includes("PAYMENT_REFUND_APPROVE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền phê duyệt hoàn tiền.", 403);
      const receiptId = string(body.receiptId).trim();
      const amount = number(body.amount);
      const beneficiaryAccount = string(body.beneficiaryAccount).trim();
      const beneficiaryBank = string(body.beneficiaryBank).trim();
      const beneficiaryHolder = string(body.beneficiaryHolder).trim();
      const reason = string(body.reason).trim();
      if (!receiptId || !Number.isInteger(amount) || amount < 1 || beneficiaryAccount.length < 4 || !beneficiaryBank || !beneficiaryHolder || !reason || reason.length > 1000) return responseError("INVALID_REQUEST", "Phê duyệt cần biên lai, số tiền, tài khoản người nhận và lý do.", 400);
      const created = await upstream(request, "admin/payments/refund-approvals", { method: "POST", body: { receipt_id: receiptId, amount, beneficiary_account: beneficiaryAccount, beneficiary_bank: beneficiaryBank, beneficiary_holder: beneficiaryHolder, reason } });
      return NextResponse.json(mapApproval(created), { status: 201 });
    }
    if (route.kind === "refund-approval-withdraw") {
      if (!currentSession.permissions.includes("PAYMENT_REFUND_APPROVE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền rút phê duyệt hoàn tiền.", 403);
      const withdrawalReference = string(body.withdrawalReference).trim();
      const reason = string(body.reason).trim();
      if (!withdrawalReference || withdrawalReference.length > 128) return responseError("INVALID_REQUEST", "Yêu cầu rút phê duyệt cần mã tham chiếu rút.", 400);
      return NextResponse.json(mapApproval(await upstream(request, `admin/payments/refund-approvals/${encodeURIComponent(route.approvalId)}/withdraw`, { method: "POST", body: { withdrawal_reference: withdrawalReference, reason } })));
    }
    if (route.kind === "refund-approval-accept") {
      if (!currentSession.permissions.includes("PAYMENT_REFUND_APPROVE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền tiếp nhận phê duyệt hoàn tiền.", 403);
      const idempotencyKey = string(body.idempotencyKey).trim();
      if (!idempotencyKey) return responseError("INVALID_REQUEST", "Yêu cầu tiếp nhận thiếu khóa idempotency.", 400);
      const created = await upstream(request, `admin/payments/refund-approvals/${encodeURIComponent(route.approvalId)}/accept`, { method: "POST", idempotencyKey });
      return NextResponse.json(mapRefund(created), { status: 201 });
    }
    if (route.kind === "refund-evidence-records") {
      if (!currentSession.permissions.includes("PAYMENT_REFUND_VERIFY")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền cấp chứng nhận sao kê.", 403);
      const decision = string(body.decision);
      const sha256 = string(body.sha256).trim();
      const amount = number(body.amount);
      const occurredAt = string(body.occurredAt).trim();
      if (!string(body.refundId).trim() || !string(body.attemptId).trim() || !["SENT", "NOT_SENT", "INCONCLUSIVE"].includes(decision) || !/^[0-9a-f]{64}$/u.test(sha256) || !Number.isInteger(amount) || amount < 1 || string(body.beneficiaryAccount).trim().length < 4 || !string(body.bank).trim() || !string(body.holder).trim() || !string(body.bankIdentity).trim() || !string(body.bankReference).trim() || !occurredAt || Number.isNaN(Date.parse(occurredAt))) return responseError("INVALID_REQUEST", "Chứng nhận sao kê thiếu hoặc sai định dạng trường bắt buộc.", 400);
      const created = await upstream(request, "admin/payments/refund-evidence", { method: "POST", body: { refund_id: string(body.refundId).trim(), attempt_id: string(body.attemptId).trim(), decision, definitive: body.definitive === true, sha256, amount, beneficiary_account: string(body.beneficiaryAccount).trim(), bank: string(body.bank).trim(), beneficiary_holder: string(body.holder).trim(), bank_identity: string(body.bankIdentity).trim(), bank_reference: string(body.bankReference).trim(), occurred_at: occurredAt } });
      return NextResponse.json(mapCertificate(created), { status: 201 });
    }
    if (route.kind === "refund-action" || route.kind === "refund-evidence" || route.kind === "refund-verify") {
      const expectedVersion = number(body.expectedVersion);
      const reason = string(body.reason).trim();
      const idempotencyKey = string(body.idempotencyKey).trim();
      if (!Number.isInteger(expectedVersion) || expectedVersion < 1 || !reason || reason.length > 1000 || !idempotencyKey) return responseError("INVALID_REQUEST", "Yêu cầu hoàn tiền thiếu phiên bản, lý do hoặc khóa idempotency.", 400);
      const refundPath = `admin/payments/refunds/${encodeURIComponent(route.refundId)}`;
      if (route.kind === "refund-action") {
        const gate = route.action === "cancel-obligation" ? "PAYMENT_REFUND_APPROVE" : "PAYMENT_REFUND_EXECUTE";
        if (!currentSession.permissions.includes(gate)) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền thực thi hoàn tiền.", 403);
        const careDecisionReference = string(body.careDecisionReference).trim();
        if (route.action === "cancel-obligation" && !careDecisionReference) return responseError("INVALID_REQUEST", "Hủy nghĩa vụ cần mã tham chiếu rút phê duyệt.", 400);
        const result = await upstream(request, `${refundPath}/${route.action}`, { method: "POST", idempotencyKey, body: { expected_version: expectedVersion, reason, ...(careDecisionReference ? { care_decision_reference: careDecisionReference } : {}) } });
        return NextResponse.json(route.action === "start" ? mapInstruction(result) : mapRefund(result));
      }
      const attemptId = string(body.attemptId).trim();
      const evidenceIds = stringList(body.evidenceIds);
      if (!attemptId || !evidenceIds.length) return responseError("INVALID_REQUEST", "Yêu cầu cần lượt thực thi và ít nhất một chứng nhận sao kê.", 400);
      if (route.kind === "refund-evidence") {
        if (!currentSession.permissions.includes("PAYMENT_REFUND_EXECUTE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền đính kèm chứng từ hoàn tiền.", 403);
        return NextResponse.json(mapRefund(await upstream(request, `${refundPath}/evidence`, { method: "POST", idempotencyKey, body: { expected_version: expectedVersion, reason, attempt_id: attemptId, evidence_ids: evidenceIds } })));
      }
      if (!currentSession.permissions.includes("PAYMENT_REFUND_VERIFY")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xác minh hoàn tiền.", 403);
      const outcome = string(body.outcome);
      if (!["SENT", "NOT_SENT", "INCONCLUSIVE"].includes(outcome)) return responseError("INVALID_REQUEST", "Kết quả xác minh không hợp lệ.", 400);
      let bankEvidence: Record<string, unknown> | undefined;
      if (outcome === "SENT") {
        // The bank presentation is assembled here from the sealed
        // certificate, never from browser-supplied bank data.
        const certs = await upstream(request, `admin/payments/refund-evidence?refund_id=${encodeURIComponent(route.refundId)}`);
        const cert = array(certs.items).find((item) => record(item) && string(item.reference) === evidenceIds[0]);
        if (!record(cert)) return responseError("INVALID_REQUEST", "Không tìm thấy chứng nhận sao kê đã cấp cho hoàn tiền này.", 400);
        bankEvidence = { source_bank_account_id: string(cert.source_bank_account_id), outgoing_reference: string(cert.bank_reference), amount: number(cert.amount), currency: "VND", beneficiary_snapshot_hash: beneficiarySnapshotHash(string(cert.beneficiary_account), string(cert.bank), string(cert.beneficiary_holder)), confirmed_at: string(cert.occurred_at) };
      }
      return NextResponse.json(mapRefund(await upstream(request, `${refundPath}/verify`, { method: "POST", idempotencyKey, body: { expected_version: expectedVersion, reason, attempt_id: attemptId, outcome, evidence_ids: evidenceIds, ...(bankEvidence ? { bank_evidence: bankEvidence } : {}) } })));
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
