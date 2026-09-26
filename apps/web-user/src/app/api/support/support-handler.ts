import { NextRequest, NextResponse } from "next/server";
import type { CreateSupportTicketResult, SupportContext, SupportMessage, SupportPriority, SupportPublicAttachment, SupportSubjectCode, SupportTicketDetail, SupportTicketStatus } from "@/lib/support/types";
import { parseSupportRoute, validateCreateSupportTicket, validateSupportMessage } from "@/lib/support/validation";

const SUBJECTS = new Set<SupportSubjectCode>(["PRODUCT_ADVICE", "ORDER_HELP", "DELIVERY_HELP", "PAYMENT_HELP", "RETURN_HELP", "OTHER"]);
const PRIORITIES = new Set<SupportPriority>(["NORMAL", "IMPORTANT", "URGENT"]);
const STATUSES = new Set<SupportTicketStatus>(["NEW", "IN_PROGRESS", "WAITING_CUSTOMER", "RESOLVED", "CLOSED"]);

interface UpstreamErrorBody { code?: string; message?: string; requestId?: string; retryAfterSeconds?: number; existingTicketId?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamErrorBody = {}) { super(body.message || `Support upstream returned ${status}`); } }

const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const boolean = (value: unknown, fallback = false) => typeof value === "boolean" ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown, fallback = new Date().toISOString()) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : fallback; };

function errorResponse(code: string, message: string, status: number, options: { errors?: Array<{ field: string; message: string }>; retryAfterSeconds?: number; existingTicketId?: string } = {}) {
  return NextResponse.json({ code, message, requestId: `BFF-SUPPORT-${code}`, ...options }, { status });
}

function hasTrustedOrigin(request: NextRequest) {
  const origin = request.headers.get("origin");
  if (!origin) return true;
  try { const source = new URL(origin); return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host; }
  catch { return false; }
}

async function readJson(request: NextRequest): Promise<unknown> { try { return await request.json(); } catch { return undefined; } }

async function upstream(path: string, method: string, request: NextRequest, scenario?: string, body?: object, idempotencyKey?: string) {
  const base = (process.env.SUPPORT_UPSTREAM_URL ?? "http://127.0.0.1:4019/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json" });
  const cookie = request.headers.get("cookie");
  if (cookie) headers.set("Cookie", cookie);
  if (body) headers.set("Content-Type", "application/json");
  if (idempotencyKey) headers.set("Idempotency-Key", idempotencyKey);
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  const response = await fetch(`${base}/${path}`, { method, headers, body: body ? JSON.stringify(body) : undefined, cache: "no-store", redirect: "manual" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload: unknown = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {});
  return record(payload) ? payload : {};
}

function mapAttachment(value: unknown): SupportPublicAttachment {
  const item = record(value) ? value : {};
  const rawState = string(item.state);
  const state: SupportPublicAttachment["state"] = rawState === "PROCESSING" || rawState === "REJECTED" ? rawState : "READY";
  return { attachmentId: string(item.attachmentId), label: string(item.label, "Tệp đính kèm"), mediaType: string(item.mediaType, "application/octet-stream"), state };
}

function mapMessage(value: unknown): SupportMessage {
  const item = record(value) ? value : {};
  const role = string(item.senderRole);
  return { messageId: string(item.messageId), senderRole: role === "SUPPORT" || role === "SYSTEM" ? role : "CUSTOMER", senderDisplayName: string(item.senderDisplayName, role === "SUPPORT" ? "Chăm sóc Tri Kỷ" : "Khách hàng"), message: string(item.message), occurredAt: date(item.occurredAt), attachments: array(item.attachments).map(mapAttachment) };
}

function mapContext(payload: Record<string, unknown>): SupportContext {
  const rawMode = string(payload.customerMode);
  const customer = record(payload.customer) ? payload.customer : null;
  const policy = record(payload.attachmentPolicy) ? payload.attachmentPolicy : {};
  return {
    customerMode: rawMode === "GUEST" ? "GUEST" : "REGISTERED",
    customer: customer ? { displayName: string(customer.displayName), emailDisplay: string(customer.emailDisplay), phoneDisplay: string(customer.phoneDisplay), membershipLabel: string(customer.membershipLabel) || undefined } : null,
    subjects: array(payload.subjects).map((value) => { const item = record(value) ? value : {}; const raw = string(item.code) as SupportSubjectCode; return { code: SUBJECTS.has(raw) ? raw : "OTHER", label: string(item.label, "Chủ đề khác"), description: string(item.description) }; }),
    priorities: array(payload.priorities).map((value) => { const item = record(value) ? value : {}; const raw = string(item.code) as SupportPriority; return { code: PRIORITIES.has(raw) ? raw : "NORMAL", label: string(item.label, "Bình thường"), description: string(item.description), responsePromiseLabel: string(item.responsePromiseLabel) }; }),
    linkableOrders: array(payload.linkableOrders).map((value) => { const item = record(value) ? value : {}; return { orderId: string(item.orderId), orderNumber: string(item.orderNumber), statusLabel: string(item.statusLabel), placedAt: date(item.placedAt), summary: string(item.summary) }; }),
    servicePromise: string(payload.servicePromise),
    attachmentPolicy: { allowedMediaTypes: array(policy.allowedMediaTypes).filter((value): value is SupportContext["attachmentPolicy"]["allowedMediaTypes"][number] => ["image/jpeg", "image/png", "image/webp", "application/pdf"].includes(string(value))), maxFiles: Math.min(5, Math.max(0, number(policy.maxFiles, 5))), maxBytesPerFile: Math.min(10 * 1024 * 1024, Math.max(1, number(policy.maxBytesPerFile, 10 * 1024 * 1024))) }
  };
}

function mapDetail(payload: Record<string, unknown>): SupportTicketDetail {
  const statusRaw = string(payload.status) as SupportTicketStatus;
  const priorityRaw = string(payload.requestedPriority) as SupportPriority;
  const promise = record(payload.servicePromise) ? payload.servicePromise : {};
  const assignee = record(payload.publicAssignee) ? payload.publicAssignee : null;
  const order = record(payload.linkedOrder) ? payload.linkedOrder : null;
  return {
    ticketId: string(payload.ticketId), ticketNumber: string(payload.ticketNumber), createdAt: date(payload.createdAt), updatedAt: date(payload.updatedAt),
    status: STATUSES.has(statusRaw) ? statusRaw : "NEW", statusLabel: string(payload.statusLabel, "Đã tiếp nhận"), statusDescription: string(payload.statusDescription), subjectLabel: string(payload.subjectLabel), title: string(payload.title),
    requestedPriority: PRIORITIES.has(priorityRaw) ? priorityRaw : "NORMAL", requestedPriorityLabel: string(payload.requestedPriorityLabel, "Bình thường"),
    servicePromise: { label: string(promise.label), targetAt: string(promise.targetAt) || null, met: promise.met === null || promise.met === undefined ? null : boolean(promise.met) },
    publicAssignee: assignee ? { displayName: string(assignee.displayName), roleLabel: string(assignee.roleLabel) } : null,
    linkedOrder: order ? { orderId: string(order.orderId), orderNumber: string(order.orderNumber), statusLabel: string(order.statusLabel), products: array(order.products).map((value) => { const item = record(value) ? value : {}; return { name: string(item.name), variant: string(item.variant), quantity: Math.max(1, number(item.quantity, 1)), priceVnd: Math.max(0, number(item.priceVnd)) }; }) } : null,
    messages: array(payload.messages).map(mapMessage), canReply: boolean(payload.canReply)
  };
}

export async function handleSupportRequest(request: NextRequest, path: string[]) {
  const route = parseSupportRoute(path);
  const scenario = request.nextUrl.searchParams.get("mockScenario") ?? undefined;
  if (route.kind === "invalid") return errorResponse("NOT_FOUND", "Support route không tồn tại.", 404);
  if (request.method === "GET" && route.kind === "context") {
    try { return NextResponse.json(mapContext(await upstream("support/context", "GET", request, scenario))); }
    catch (cause) { return handleError(cause); }
  }
  if (request.method === "POST" && route.kind === "tickets") {
    if (!hasTrustedOrigin(request)) return errorResponse("UNTRUSTED_ORIGIN", "Origin không hợp lệ.", 403);
    const parsed = validateCreateSupportTicket(await readJson(request));
    if (!parsed.ok) return errorResponse("VALIDATION_ERROR", "Thông tin yêu cầu chưa hợp lệ.", 422, { errors: parsed.errors });
    try {
      const { idempotencyKey, ...body } = parsed.data;
      const payload = await upstream("support/tickets", "POST", request, scenario, body, idempotencyKey);
      const result: CreateSupportTicketResult = { ticketId: string(payload.ticketId), ticketNumber: string(payload.ticketNumber), status: "NEW", ticketPath: `/support/tickets/${encodeURIComponent(string(payload.ticketId))}`, createdAt: date(payload.createdAt), message: string(payload.message, "Yêu cầu hỗ trợ đã được tiếp nhận.") };
      return NextResponse.json(result, { status: 201 });
    } catch (cause) { return handleError(cause); }
  }
  if (request.method === "GET" && route.kind === "ticket") {
    try { return NextResponse.json(mapDetail(await upstream(`support/tickets/${encodeURIComponent(route.ticketId)}`, "GET", request, scenario))); }
    catch (cause) { return handleError(cause); }
  }
  if (request.method === "POST" && route.kind === "messages") {
    if (!hasTrustedOrigin(request)) return errorResponse("UNTRUSTED_ORIGIN", "Origin không hợp lệ.", 403);
    const parsed = validateSupportMessage(await readJson(request));
    if (!parsed.ok) return errorResponse("VALIDATION_ERROR", "Nội dung phản hồi chưa hợp lệ.", 422, { errors: parsed.errors });
    try {
      const payload = await upstream(`support/tickets/${encodeURIComponent(route.ticketId)}/messages`, "POST", request, scenario, { message: parsed.data.message, attachments: parsed.data.attachments }, parsed.data.idempotencyKey);
      return NextResponse.json(mapMessage(payload), { status: 201 });
    } catch (cause) { return handleError(cause); }
  }
  return errorResponse("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
}

function handleError(cause: unknown) {
  if (cause instanceof UpstreamHttpError) return errorResponse(cause.body.code || "SUPPORT_ERROR", cause.body.message || "Không thể xử lý yêu cầu hỗ trợ lúc này.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, { errors: cause.body.errors ?? [], retryAfterSeconds: cause.body.retryAfterSeconds, existingTicketId: cause.body.existingTicketId });
  return errorResponse("SUPPORT_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ Support. Hãy kiểm tra Mockoon hoặc API Gateway.", 503);
}
