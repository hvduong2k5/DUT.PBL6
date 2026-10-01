import { NextRequest, NextResponse } from "next/server";
import { fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import type { CustomerCapability } from "@/lib/auth/capabilities";
import type { AcceptB2BQuoteResult, B2BAttachmentMetadata, B2BContext, B2BOrganization, B2BQuoteDetail, B2BQuoteStatus, B2BRequestStatus, CreateB2BQuoteRequestResult } from "@/lib/b2b/types";
import { parseB2BRoute, validateCreateB2BQuoteRequest } from "@/lib/b2b/validation";

interface UpstreamErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamErrorBody = {}) { super(body.message || `B2B upstream returned ${status}`); } }

const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const boolean = (value: unknown, fallback = false) => typeof value === "boolean" ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const isoDate = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };

function errorResponse(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) {
  return NextResponse.json({ code, message, errors, requestId: `BFF-B2B-${code}` }, { status });
}

function trustedOrigin(request: NextRequest) {
  const origin = request.headers.get("origin");
  if (!origin) return true;
  try { const source = new URL(origin); return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host; }
  catch { return false; }
}

async function readJson(request: NextRequest): Promise<unknown> { try { return await request.json(); } catch { return undefined; } }

async function upstream(request: NextRequest, path: string, method: string, body?: object, idempotencyKey?: string) {
  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json" });
  const accessToken = request.cookies.get("oma_access_token")?.value;
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (body) headers.set("Content-Type", "application/json");
  if (idempotencyKey) headers.set("Idempotency-Key", idempotencyKey);
  const scenario = request.nextUrl.searchParams.get("mockScenario");
  if (process.env.NODE_ENV !== "production" && scenario && /^[a-z0-9-]+$/u.test(scenario)) headers.set("X-Mock-Scenario", scenario);
  const response = await fetch(`${base}/${path}`, { method, headers, body: body ? JSON.stringify(body) : undefined, cache: "no-store", redirect: "manual" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload: unknown = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {});
  return record(payload) ? payload : {};
}

function mapOrganization(value: unknown): B2BOrganization {
  const item = record(value) ? value : {};
  return { organizationId: string(item.organizationId), legalName: string(item.legalName), taxCode: string(item.taxCode), representativeName: string(item.representativeName), representativeTitle: string(item.representativeTitle), phone: string(item.phone), email: string(item.email), invoiceAddress: string(item.invoiceAddress), verificationStatus: item.verificationStatus === "PENDING" ? "PENDING" : "VERIFIED" };
}

function mapContext(payload: Record<string, unknown>): B2BContext {
  const policy = record(payload.attachmentPolicy) ? payload.attachmentPolicy : {};
  const mediaTypes = new Set<B2BAttachmentMetadata["mediaType"]>(["image/png", "image/jpeg", "image/webp", "application/pdf", "image/svg+xml"]);
  return {
    organization: mapOrganization(payload.organization),
    catalog: array(payload.catalog).map((value) => { const item = record(value) ? value : {}; return { skuId: string(item.skuId), name: string(item.name), variant: string(item.variant), unitPriceVnd: Math.max(0, number(item.unitPriceVnd)), minimumQuantity: Math.max(1, number(item.minimumQuantity, 1)), imageUrl: string(item.imageUrl) || undefined }; }),
    deliveryProvinces: array(payload.deliveryProvinces).map((value) => { const item = record(value) ? value : {}; return { provinceCode: string(item.provinceCode), provinceName: string(item.provinceName) }; }).filter((item) => item.provinceCode && item.provinceName),
    recentRequests: array(payload.recentRequests).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as B2BRequestStatus; const status: B2BRequestStatus = ["REQUESTED", "NEEDS_INFO", "QUOTED", "ACCEPTED", "REJECTED"].includes(raw) ? raw : "REQUESTED"; return { requestId: string(item.requestId), requestNumber: string(item.requestNumber), status, statusLabel: string(item.statusLabel), submittedAt: isoDate(item.submittedAt), totalQuantity: Math.max(0, number(item.totalQuantity)), purposeLabel: string(item.purposeLabel), quoteId: string(item.quoteId) || undefined }; }),
    discountTiers: array(payload.discountTiers).map((value) => { const item = record(value) ? value : {}; return { minimumQuantity: Math.max(1, number(item.minimumQuantity, 1)), maximumQuantity: number(item.maximumQuantity) || undefined, discountPercent: Math.max(0, number(item.discountPercent)), label: string(item.label) }; }),
    attachmentPolicy: { allowedMediaTypes: array(policy.allowedMediaTypes).filter((value): value is B2BAttachmentMetadata["mediaType"] => mediaTypes.has(string(value) as B2BAttachmentMetadata["mediaType"])), maxFiles: Math.min(3, Math.max(0, number(policy.maxFiles, 3))), maxBytesPerFile: Math.min(25 * 1024 * 1024, Math.max(1, number(policy.maxBytesPerFile, 25 * 1024 * 1024))) }
  };
}

function mapQuote(payload: Record<string, unknown>): B2BQuoteDetail {
  const totals = record(payload.totals) ? payload.totals : {};
  const contact = record(payload.contact) ? payload.contact : {};
  const payment = record(payload.payment) ? payload.payment : {};
  const rawStatus = string(payload.status) as B2BQuoteStatus;
  const status: B2BQuoteStatus = ["SENT", "ACCEPTED", "EXPIRED", "WITHDRAWN", "SUPERSEDED"].includes(rawStatus) ? rawStatus : "SENT";
  const organization = mapOrganization(payload.organization);
  return {
    quoteId: string(payload.quoteId), quoteNumber: string(payload.quoteNumber), version: Math.max(1, number(payload.version, 1)), status, statusLabel: string(payload.statusLabel), issuedAt: isoDate(payload.issuedAt), expiresAt: isoDate(payload.expiresAt),
    organization: { organizationId: organization.organizationId, legalName: organization.legalName, taxCode: organization.taxCode },
    contact: { displayName: string(contact.displayName), title: string(contact.title), phone: string(contact.phone), email: string(contact.email) },
    items: array(payload.items).map((value) => { const item = record(value) ? value : {}; return { skuId: string(item.skuId), name: string(item.name), variant: string(item.variant), quantity: Math.max(1, number(item.quantity, 1)), unitPriceVnd: Math.max(0, number(item.unitPriceVnd)), lineTotalVnd: Math.max(0, number(item.lineTotalVnd)) }; }),
    customizations: array(payload.customizations).map((value) => string(value)).filter(Boolean),
    totals: { merchandiseVnd: number(totals.merchandiseVnd), discountVnd: number(totals.discountVnd), customizationVnd: number(totals.customizationVnd), vatVnd: number(totals.vatVnd), grandTotalVnd: number(totals.grandTotalVnd), depositPercent: number(totals.depositPercent), depositVnd: number(totals.depositVnd), remainingVnd: number(totals.remainingVnd) },
    terms: array(payload.terms).map((value) => { const item = record(value) ? value : {}; return { title: string(item.title), description: string(item.description) }; }),
    payment: { bankName: string(payment.bankName), accountName: string(payment.accountName), accountNumberMasked: string(payment.accountNumberMasked), transferContent: string(payment.transferContent) },
    canAccept: boolean(payload.canAccept), acceptedAt: string(payload.acceptedAt) || undefined, orderId: string(payload.orderId) || undefined
  };
}

async function requireCapability(request: NextRequest, capability: CustomerCapability) {
  const accessToken = request.cookies.get("oma_access_token")?.value;
  if (!accessToken) return errorResponse("AUTH_REQUIRED", "Vui lòng đăng nhập bằng tài khoản doanh nghiệp.", 401);
  try {
    const projection = await fetchCustomerCapabilities(accessToken);
    if (projection.actor !== "B2B" || !projection.capabilities.includes(capability)) return errorResponse("B2B_CAPABILITY_FORBIDDEN", "Tài khoản không có quyền thực hiện chức năng B2B này.", 403);
    return null;
  } catch { return errorResponse("CAPABILITY_UNAVAILABLE", "Chưa thể kiểm tra quyền B2B lúc này.", 503); }
}

export async function handleB2BRequest(request: NextRequest, path: string[]) {
  const route = parseB2BRoute(path);
  if (route.kind === "invalid") return errorResponse("NOT_FOUND", "B2B route không tồn tại.", 404);
  const expectedMethod = route.kind === "context" || route.kind === "quote" ? "GET" : "POST";
  if (request.method !== expectedMethod) return errorResponse("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  if (request.method === "POST" && !trustedOrigin(request)) return errorResponse("UNTRUSTED_ORIGIN", "Origin không hợp lệ.", 403);
  const capability: CustomerCapability = route.kind === "context" ? "B2B_COMPANY_VIEW" : route.kind === "quote" ? "B2B_QUOTE_VIEW" : route.kind === "requests" ? "B2B_QUOTE_CREATE" : "B2B_QUOTE_ACCEPT";
  const forbidden = await requireCapability(request, capability);
  if (forbidden) return forbidden;

  try {
    if (route.kind === "context") return NextResponse.json(mapContext(await upstream(request, "b2b/context", "GET")));
    if (route.kind === "requests") {
      const parsed = validateCreateB2BQuoteRequest(await readJson(request));
      if (!parsed.ok) return errorResponse("VALIDATION_ERROR", "Thông tin yêu cầu báo giá chưa hợp lệ.", 422, parsed.errors);
      const { idempotencyKey, ...body } = parsed.data;
      const payload = await upstream(request, "b2b/quote-requests", "POST", body, idempotencyKey);
      const result: CreateB2BQuoteRequestResult = { requestId: string(payload.requestId), requestNumber: string(payload.requestNumber), status: "REQUESTED", statusLabel: string(payload.statusLabel, "Đã tiếp nhận"), submittedAt: isoDate(payload.submittedAt), totalQuantity: number(payload.totalQuantity), purposeLabel: string(payload.purposeLabel), quoteId: string(payload.quoteId) || undefined, message: string(payload.message, "Yêu cầu báo giá đã được tiếp nhận.") };
      return NextResponse.json(result, { status: 201 });
    }
    if (route.kind === "quote") return NextResponse.json(mapQuote(await upstream(request, `b2b/quotes/${encodeURIComponent(route.quoteId)}`, "GET")));
    const body = await readJson(request);
    if (!record(body) || body.confirmation !== true || string(body.idempotencyKey).length < 16) return errorResponse("VALIDATION_ERROR", "Xác nhận chấp thuận báo giá không hợp lệ.", 422);
    const payload = await upstream(request, `b2b/quotes/${encodeURIComponent(route.quoteId)}/accept`, "POST", { confirmation: true }, string(body.idempotencyKey));
    const result: AcceptB2BQuoteResult = { quoteId: string(payload.quoteId, route.quoteId), status: "ACCEPTED", acceptedAt: isoDate(payload.acceptedAt), orderId: string(payload.orderId), orderNumber: string(payload.orderNumber), message: string(payload.message, "Báo giá đã được chấp thuận.") };
    return NextResponse.json(result);
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return errorResponse(cause.body.code || "B2B_ERROR", cause.body.message || "Không thể xử lý yêu cầu B2B lúc này.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return errorResponse("B2B_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ B2B. Hãy kiểm tra Mockoon customer extensions.", 503);
  }
}
