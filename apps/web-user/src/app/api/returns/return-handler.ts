import { NextRequest, NextResponse } from "next/server";
import type { CreateReturnCaseResult, ReturnCaseDetail, ReturnCaseMessage, ReturnCaseStatus, ReturnEligibility, ReturnReason, ReturnResolution } from "@/lib/returns/types";
import { parseReturnRoute, validateCreateReturnCase, validateSupplement } from "@/lib/returns/validation";

const CASE_STATUSES = new Set<ReturnCaseStatus>(["RETURN_REQUESTED", "EVIDENCE_REQUIRED", "REVIEWING", "APPROVED", "REJECTED", "RETURN_IN_TRANSIT", "RECEIVED", "REFUND_PENDING", "REFUNDED", "CLOSED"]);
const RESOLUTIONS = new Set<ReturnResolution>(["REPLACEMENT", "ORIGINAL_PAYMENT_REFUND", "LOYALTY_CREDIT"]);
const REASONS = new Set<ReturnReason>(["DAMAGED_IN_TRANSIT", "WRONG_ITEM", "QUALITY_ISSUE", "OTHER"]);

interface UpstreamErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  retryAfterSeconds?: number;
  existingCaseId?: string;
  errors?: Array<{ field: string; message: string }>;
}

class UpstreamHttpError extends Error {
  constructor(readonly status: number, readonly body: UpstreamErrorBody = {}) {
    super(body.message || `Return upstream returned ${status}`);
  }
}

function errorResponse(code: string, message: string, status: number, options: { errors?: Array<{ field: string; message: string }>; retryAfterSeconds?: number; existingCaseId?: string } = {}) {
  return NextResponse.json({ code, message, requestId: `BFF-RETURN-${code}`, ...options }, { status });
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
  const base = (process.env.RETURN_UPSTREAM_URL ?? "http://127.0.0.1:4018/api/v1").replace(/\/$/u, "");
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
  return payload;
}

function text(value: unknown, fallback = "") { return typeof value === "string" ? value : fallback; }
function integer(value: unknown, fallback = 0) { return Number.isInteger(value) && Number(value) >= 0 ? Number(value) : fallback; }
function dateTime(value: unknown, fallback = new Date(0).toISOString()) { return typeof value === "string" && !Number.isNaN(Date.parse(value)) ? value : fallback; }
function nullableText(value: unknown) { return typeof value === "string" ? value : null; }

function publicEligibility(value: Record<string, unknown>): ReturnEligibility {
  if (typeof value.orderId !== "string" || typeof value.orderNumber !== "string") throw new Error("Invalid return eligibility");
  const pickup = value.pickupSnapshot && typeof value.pickupSnapshot === "object" && !Array.isArray(value.pickupSnapshot) ? value.pickupSnapshot as Record<string, unknown> : {};
  const lines = Array.isArray(value.lines) ? value.lines.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const line = item as Record<string, unknown>;
    if (typeof line.lineId !== "string" || typeof line.productName !== "string") return [];
    return [{ lineId: line.lineId, productName: line.productName, skuLabel: text(line.skuLabel), purchasedQty: integer(line.purchasedQty), alreadyClaimedQty: integer(line.alreadyClaimedQty), maxReturnQty: integer(line.maxReturnQty), unitPriceVnd: integer(line.unitPriceVnd), eligible: line.eligible === true }];
  }) : [];
  const reasonOptions = Array.isArray(value.reasonOptions) ? value.reasonOptions.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const option = item as Record<string, unknown>;
    return REASONS.has(option.code as ReturnReason) && typeof option.label === "string" ? [{ code: option.code as ReturnReason, label: option.label }] : [];
  }) : [];
  const resolutionOptions = Array.isArray(value.resolutionOptions) ? value.resolutionOptions.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const option = item as Record<string, unknown>;
    return RESOLUTIONS.has(option.code as ReturnResolution) && typeof option.label === "string" ? [{ code: option.code as ReturnResolution, label: option.label, description: text(option.description) }] : [];
  }) : [];
  return { orderId: value.orderId, orderNumber: value.orderNumber, deliveredAt: dateTime(value.deliveredAt), policyWindowEndsAt: dateTime(value.policyWindowEndsAt), eligible: value.eligible === true, policyMessage: text(value.policyMessage), lines, reasonOptions, resolutionOptions, pickupSnapshot: { recipientName: text(pickup.recipientName), phoneDisplay: text(pickup.phoneDisplay), addressDisplay: text(pickup.addressDisplay) } };
}

function publicCase(value: Record<string, unknown>): ReturnCaseDetail {
  if (typeof value.caseId !== "string" || typeof value.caseNumber !== "string" || typeof value.orderId !== "string" || typeof value.orderNumber !== "string" || !CASE_STATUSES.has(value.status as ReturnCaseStatus)) throw new Error("Invalid return case projection");
  const decisionSource = value.decision && typeof value.decision === "object" && !Array.isArray(value.decision) ? value.decision as Record<string, unknown> : null;
  const pickupSource = value.pickup && typeof value.pickup === "object" && !Array.isArray(value.pickup) ? value.pickup as Record<string, unknown> : null;
  const refundSource = value.refund && typeof value.refund === "object" && !Array.isArray(value.refund) ? value.refund as Record<string, unknown> : {};
  const steps = Array.isArray(value.steps) ? value.steps.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const step = item as Record<string, unknown>;
    const state = ["COMPLETED", "CURRENT", "UPCOMING"].includes(text(step.state)) ? text(step.state) as "COMPLETED" | "CURRENT" | "UPCOMING" : "UPCOMING";
    return typeof step.code === "string" && typeof step.label === "string" ? [{ code: step.code, label: step.label, state, occurredAt: step.occurredAt === null ? null : dateTime(step.occurredAt) }] : [];
  }) : [];
  const items = Array.isArray(value.items) ? value.items.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const line = item as Record<string, unknown>;
    if (typeof line.lineId !== "string" || typeof line.productName !== "string" || !RESOLUTIONS.has(line.requestedResolution as ReturnResolution)) return [];
    return [{ lineId: line.lineId, productName: line.productName, skuLabel: text(line.skuLabel), quantity: integer(line.quantity, 1), unitPriceVnd: integer(line.unitPriceVnd), requestedResolution: line.requestedResolution as ReturnResolution, approvedResolution: RESOLUTIONS.has(line.approvedResolution as ReturnResolution) ? line.approvedResolution as ReturnResolution : null }];
  }) : [];
  const evidence = Array.isArray(value.evidence) ? value.evidence.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const file = item as Record<string, unknown>;
    const state = ["READY", "PROCESSING", "REJECTED"].includes(text(file.state)) ? text(file.state) as "READY" | "PROCESSING" | "REJECTED" : "PROCESSING";
    const previewKind = ["PACKAGE", "VIDEO", "WORKSHOP", "DOCUMENT"].includes(text(file.previewKind)) ? text(file.previewKind) as "PACKAGE" | "VIDEO" | "WORKSHOP" | "DOCUMENT" : "DOCUMENT";
    return typeof file.evidenceId === "string" ? [{ evidenceId: file.evidenceId, label: text(file.label), mediaType: text(file.mediaType), state, previewKind }] : [];
  }) : [];
  const communications = Array.isArray(value.communications) ? value.communications.flatMap((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const message = item as Record<string, unknown>;
    const senderRole = ["CUSTOMER", "SUPPORT", "SYSTEM"].includes(text(message.senderRole)) ? text(message.senderRole) as ReturnCaseMessage["senderRole"] : "SYSTEM";
    return typeof message.messageId === "string" ? [{ messageId: message.messageId, senderRole, senderDisplayName: text(message.senderDisplayName), message: text(message.message), occurredAt: dateTime(message.occurredAt) }] : [];
  }) : [];
  const refundStatus = ["PENDING", "COMPLETED", "FAILED"].includes(text(refundSource.status)) ? text(refundSource.status) as "PENDING" | "COMPLETED" | "FAILED" : null;
  return {
    caseId: value.caseId, caseNumber: value.caseNumber, orderId: value.orderId, orderNumber: value.orderNumber, createdAt: dateTime(value.createdAt), status: value.status as ReturnCaseStatus, statusLabel: text(value.statusLabel), statusDescription: text(value.statusDescription), steps,
    decision: decisionSource ? { label: text(decisionSource.label), summary: text(decisionSource.summary), decidedAt: dateTime(decisionSource.decidedAt) } : null,
    items, evidence,
    pickup: pickupSource ? { carrierLabel: text(pickupSource.carrierLabel), windowLabel: text(pickupSource.windowLabel), trackingCode: text(pickupSource.trackingCode), statusLabel: text(pickupSource.statusLabel), addressDisplay: text(pickupSource.addressDisplay) } : null,
    refund: { required: refundSource.required === true, methodLabel: nullableText(refundSource.methodLabel), amountVnd: integer(refundSource.amountVnd), status: refundStatus }, communications
  };
}

export async function handleReturnRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parseReturnRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation) return errorResponse(route.error?.field === "path" ? "NOT_FOUND" : "INVALID_REQUEST", route.error?.message ?? "Yêu cầu hậu mãi không hợp lệ.", route.error?.field === "path" ? 404 : 400, { errors: route.error ? [route.error] : [] });
  if (process.env.NODE_ENV === "production" && route.mockScenario) return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400);
  if (request.method !== "GET" && !hasTrustedOrigin(request)) return errorResponse("ORIGIN_NOT_ALLOWED", "Request origin không được phép.", 403);

  try {
    if (route.operation === "eligibility" && route.orderId) {
      return NextResponse.json(publicEligibility(await upstream(`orders/${encodeURIComponent(route.orderId)}/return-eligibility`, "GET", request, route.mockScenario)));
    }
    if (route.operation === "create") {
      const parsed = validateCreateReturnCase(await readJson(request));
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Hồ sơ hậu mãi chưa hợp lệ.", 422, { errors: parsed.errors });
      const { idempotencyKey, ...body } = parsed.data;
      const payload = await upstream("return-cases", "POST", request, route.mockScenario, body, idempotencyKey);
      const result: CreateReturnCaseResult = { caseId: text(payload.caseId), caseNumber: text(payload.caseNumber), status: "RETURN_REQUESTED", casePath: text(payload.casePath), createdAt: dateTime(payload.createdAt), message: text(payload.message) };
      return NextResponse.json(result, { status: 201 });
    }
    if (route.operation === "detail" && route.caseId) {
      return NextResponse.json(publicCase(await upstream(`return-cases/${encodeURIComponent(route.caseId)}`, "GET", request, route.mockScenario)));
    }
    const parsed = validateSupplement(await readJson(request));
    if (!parsed.data || !route.caseId) return errorResponse("VALIDATION_ERROR", "Nội dung bổ sung chưa hợp lệ.", 422, { errors: parsed.errors });
    const payload = await upstream(`return-cases/${encodeURIComponent(route.caseId)}/supplements`, "POST", request, route.mockScenario, { message: parsed.data.message }, parsed.data.idempotencyKey);
    const result: ReturnCaseMessage = { messageId: text(payload.messageId), senderRole: "CUSTOMER", senderDisplayName: text(payload.senderDisplayName, "Khách hàng"), message: parsed.data.message, occurredAt: dateTime(payload.occurredAt, new Date().toISOString()) };
    return NextResponse.json(result, { status: 201 });
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return errorResponse(cause.body.code || "RETURN_ERROR", cause.body.message || "Không thể xử lý hồ sơ hậu mãi lúc này.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, { errors: cause.body.errors ?? [], retryAfterSeconds: cause.body.retryAfterSeconds, existingCaseId: cause.body.existingCaseId });
    return errorResponse("RETURN_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ Return. Hãy kiểm tra Mockoon hoặc API Gateway.", 503);
  }
}
