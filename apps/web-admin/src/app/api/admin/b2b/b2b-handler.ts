import { NextRequest, NextResponse } from "next/server";
import { parseAdminSession, type AdminPermission, type AdminSession } from "@/lib/auth/types";
import type { AdminB2BAction, AdminB2BActionResult, AdminB2BFileHistory, AdminB2BFileRecord, AdminB2BFileScanStatus, AdminB2BList, AdminB2BQuoteVersion, AdminB2BQuoteVersionHistory, AdminB2BQuoteVersionStatus, AdminB2BRequestDetail, AdminB2BRequestSummary, AdminB2BSlaState, AdminB2BStatus } from "@/lib/b2b/types";
import { parseAdminB2BPath, validateAdminB2BAction } from "@/lib/b2b/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const STATUSES = new Set<AdminB2BStatus>(["REQUESTED", "NEEDS_INFO", "DRAFT", "SENT", "ACCEPTED", "REJECTED", "EXPIRED", "WITHDRAWN", "CONVERTED"]);
const SLA = new Set<AdminB2BSlaState>(["ON_TRACK", "AT_RISK", "OVERDUE"]);
const ACTIONS = new Set<AdminB2BAction>(["VIEW", "ASSIGN", "REQUEST_INFO", "CREATE_DRAFT", "ISSUE", "WITHDRAW", "REJECT", "CONVERT_ORDER"]);
const VERSION_STATUSES = new Set<AdminB2BQuoteVersionStatus>(["DRAFT", "SENT", "ACCEPTED", "SUPERSEDED", "EXPIRED", "WITHDRAWN"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) { return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-B2B-${code}` }, { status }); }
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest, body = false, idempotencyKey?: string) {
  const result = new Headers({ Accept: "application/json" });
  const authorization = request.headers.get("authorization");
  if (authorization) result.set("Authorization", authorization);
  const profile = mockProfile();
  if (profile) result.set("X-Admin-Profile", profile);
  if (body) result.set("Content-Type", "application/json");
  if (idempotencyKey) result.set("Idempotency-Key", idempotencyKey);
  const scenario = request.nextUrl.searchParams.get("mockScenario");
  if (process.env.NODE_ENV !== "production" && scenario && /^[a-z0-9-]+$/u.test(scenario)) result.set("X-Mock-Scenario", scenario);
  return result;
}
async function upstream(request: NextRequest, path: string, method = "GET", body?: object, idempotencyKey?: string) {
  const response = await fetch(`${baseUrl()}/${path}`, { method, headers: headers(request, body !== undefined, idempotencyKey), body: body ? JSON.stringify(body) : undefined, cache: "no-store", redirect: "manual" });
  const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {});
  return record(payload) ? payload : {};
}
async function session(request: NextRequest): Promise<AdminSession> {
  const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" });
  if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." });
  const parsed = parseAdminSession(await response.json());
  if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." });
  return parsed;
}
function trustedOrigin(request: NextRequest) { const origin = request.headers.get("origin"); if (!origin) return true; try { const source = new URL(origin); return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host; } catch { return false; } }
async function readJson(request: NextRequest): Promise<unknown> { try { return await request.json(); } catch { return undefined; } }

function mapSummary(value: unknown): AdminB2BRequestSummary {
  const item = record(value) ? value : {};
  const statusRaw = string(item.status) as AdminB2BStatus;
  const slaRaw = string(item.slaState) as AdminB2BSlaState;
  const owner = record(item.owner) ? item.owner : null;
  return { requestId: string(item.requestId), requestNumber: string(item.requestNumber), companyName: string(item.companyName), taxCode: string(item.taxCode), submittedAt: date(item.submittedAt), lineCount: Math.max(0, number(item.lineCount)), totalQuantity: Math.max(0, number(item.totalQuantity)), status: STATUSES.has(statusRaw) ? statusRaw : "REQUESTED", statusLabel: string(item.statusLabel), slaState: SLA.has(slaRaw) ? slaRaw : "ON_TRACK", slaLabel: string(item.slaLabel), owner: owner ? { employeeId: string(owner.employeeId), displayName: string(owner.displayName) } : null, currentQuoteVersion: typeof item.currentQuoteVersion === "number" ? item.currentQuoteVersion : null, quoteExpiresAt: typeof item.quoteExpiresAt === "string" ? item.quoteExpiresAt : null, estimatedValueVnd: typeof item.estimatedValueVnd === "number" ? item.estimatedValueVnd : null, revision: Math.max(1, number(item.revision, 1)) };
}
function mapList(payload: Record<string, unknown>): AdminB2BList {
  const items = array(payload.items).map(mapSummary);
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | AdminB2BStatus; return { status: raw === "ALL" || STATUSES.has(raw as AdminB2BStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  for (const terminal of [{ status: "WITHDRAWN", label: "Đã thu hồi" }, { status: "CONVERTED", label: "Đã chuyển Order" }] as const) if (!statusCounts.some((item) => item.status === terminal.status)) statusCounts.push({ ...terminal, count: items.filter((item) => item.status === terminal.status).length });
  return { items, statusCounts, owners: array(payload.owners).map((value) => { const item = record(value) ? value : {}; return { employeeId: string(item.employeeId), displayName: string(item.displayName) }; }), updatedAt: date(payload.updatedAt) };
}
function mapDetail(payload: Record<string, unknown>, canViewFiles: boolean): AdminB2BRequestDetail {
  const summary = mapSummary(payload);
  const company = record(payload.company) ? payload.company : {};
  const requester = record(payload.requester) ? payload.requester : {};
  const quote = record(payload.quote) ? payload.quote : null;
  return { ...summary, company: { legalName: string(company.legalName), taxCode: string(company.taxCode), invoiceAddress: string(company.invoiceAddress), verificationStatus: company.verificationStatus === "PENDING" ? "PENDING" : "VERIFIED" }, requester: { displayName: string(requester.displayName), title: string(requester.title), phone: string(requester.phone), email: string(requester.email) }, purposeLabel: string(payload.purposeLabel), requestedDeliveryDate: date(payload.requestedDeliveryDate), deliveryLocation: string(payload.deliveryLocation), notes: string(payload.notes), items: array(payload.items).map((value) => { const item = record(value) ? value : {}; return { skuId: string(item.skuId), name: string(item.name), variant: string(item.variant), quantity: Math.max(1, number(item.quantity, 1)), catalogPriceVnd: Math.max(0, number(item.catalogPriceVnd)) }; }), customizations: array(payload.customizations).map((value) => string(value)).filter(Boolean), files: canViewFiles ? array(payload.files).map((value) => { const item = record(value) ? value : {}; const scan = string(item.scanStatus); return { fileId: string(item.fileId), fileName: string(item.fileName), mediaType: string(item.mediaType), scanStatus: scan === "SAFE" || scan === "REJECTED" ? scan : "PROCESSING" }; }) : [], quote: quote ? { quoteId: string(quote.quoteId), version: Math.max(1, number(quote.version, 1)), status: ["DRAFT", "SENT", "ACCEPTED", "SUPERSEDED"].includes(string(quote.status)) ? string(quote.status) as "DRAFT" | "SENT" | "ACCEPTED" | "SUPERSEDED" : "DRAFT", subtotalVnd: number(quote.subtotalVnd), discountPercent: number(quote.discountPercent), customizationVnd: number(quote.customizationVnd), shippingVnd: number(quote.shippingVnd), vatPercent: number(quote.vatPercent), grandTotalVnd: number(quote.grandTotalVnd), expiresAt: date(quote.expiresAt) } : null, activity: array(payload.activity).map((value) => { const item = record(value) ? value : {}; return { occurredAt: date(item.occurredAt), actorLabel: string(item.actorLabel), description: string(item.description) }; }), allowedActions: array(payload.allowedActions).filter((value): value is AdminB2BAction => ACTIONS.has(string(value) as AdminB2BAction)) };
}

function mapVersion(value: unknown): AdminB2BQuoteVersion {
  const item = record(value) ? value : {};
  const organization = record(item.organization) ? item.organization : {};
  const contact = record(item.contact) ? item.contact : {};
  const totals = record(item.totals) ? item.totals : {};
  const payment = record(item.payment) ? item.payment : {};
  const statusRaw = string(item.status) as AdminB2BQuoteVersionStatus;
  const status = VERSION_STATUSES.has(statusRaw) ? statusRaw : "DRAFT";
  return {
    quoteId: string(item.quoteId), quoteNumber: string(item.quoteNumber), version: Math.max(1, number(item.version, 1)), status, statusLabel: string(item.statusLabel), immutable: item.immutable === true || status !== "DRAFT", createdAt: date(item.createdAt), createdBy: string(item.createdBy), issuedAt: typeof item.issuedAt === "string" ? date(item.issuedAt) : null, expiresAt: date(item.expiresAt), supersedesVersion: typeof item.supersedesVersion === "number" ? item.supersedesVersion : null,
    organization: { legalName: string(organization.legalName), taxCode: string(organization.taxCode) }, contact: { displayName: string(contact.displayName), title: string(contact.title), phone: string(contact.phone), email: string(contact.email) },
    items: array(item.items).map((value) => { const line = record(value) ? value : {}; return { skuId: string(line.skuId), name: string(line.name), variant: string(line.variant), quantity: Math.max(1, number(line.quantity, 1)), unitPriceVnd: Math.max(0, number(line.unitPriceVnd)), lineTotalVnd: Math.max(0, number(line.lineTotalVnd)) }; }),
    customizations: array(item.customizations).map((entry) => string(entry)).filter(Boolean), totals: { merchandiseVnd: number(totals.merchandiseVnd), discountVnd: number(totals.discountVnd), customizationVnd: number(totals.customizationVnd), vatVnd: number(totals.vatVnd), grandTotalVnd: number(totals.grandTotalVnd), depositPercent: number(totals.depositPercent), depositVnd: number(totals.depositVnd), remainingVnd: number(totals.remainingVnd) }, terms: array(item.terms).map((value) => { const term = record(value) ? value : {}; return { title: string(term.title), description: string(term.description) }; }), payment: { bankName: string(payment.bankName), accountName: string(payment.accountName), accountNumberMasked: string(payment.accountNumberMasked), transferContent: string(payment.transferContent) }, acceptedAt: typeof item.acceptedAt === "string" ? date(item.acceptedAt) : undefined, orderId: string(item.orderId) || undefined
  };
}

function mapVersionHistory(payload: Record<string, unknown>): AdminB2BQuoteVersionHistory {
  return { requestId: string(payload.requestId), requestNumber: string(payload.requestNumber), items: array(payload.items).map(mapVersion).sort((left, right) => right.version - left.version) };
}

function fileScanStatus(value: unknown): AdminB2BFileScanStatus {
  const status = string(value);
  return status === "SAFE" || status === "REJECTED" ? status : "PROCESSING";
}

function mapFileRecord(value: unknown): AdminB2BFileRecord {
  const item = record(value) ? value : {};
  const versions = array(item.versions).map((value) => {
    const version = record(value) ? value : {};
    return {
      version: Math.max(1, number(version.version, 1)),
      isCurrent: version.isCurrent === true,
      fileName: string(version.fileName),
      mediaType: string(version.mediaType),
      sizeBytes: Math.max(0, number(version.sizeBytes)),
      scanStatus: fileScanStatus(version.scanStatus),
      uploadedAt: date(version.uploadedAt),
      uploadedByLabel: string(version.uploadedByLabel),
      referencedByQuoteVersions: array(version.referencedByQuoteVersions).map((entry) => number(entry)).filter((entry) => Number.isInteger(entry) && entry > 0)
    };
  }).sort((left, right) => right.version - left.version);
  const currentVersion = Math.max(1, number(item.currentVersion, versions.find((version) => version.isCurrent)?.version ?? versions[0]?.version ?? 1));
  return { fileId: string(item.fileId), purposeLabel: string(item.purposeLabel), ownerOrganizationId: string(item.ownerOrganizationId), ownerOrganizationName: string(item.ownerOrganizationName), currentVersion, versions: versions.map((version) => ({ ...version, isCurrent: version.version === currentVersion })) };
}

function mapFileHistory(payload: Record<string, unknown>): AdminB2BFileHistory {
  return { requestId: string(payload.requestId), requestNumber: string(payload.requestNumber), items: array(payload.items).map(mapFileRecord) };
}

const ACTION_PERMISSION: Record<Exclude<AdminB2BAction, "VIEW">, AdminPermission> = { ASSIGN: "B2B_REQUEST_ASSIGN", REQUEST_INFO: "B2B_REQUEST_INFO_REQUEST", CREATE_DRAFT: "B2B_QUOTE_DRAFT", ISSUE: "B2B_QUOTE_ISSUE", WITHDRAW: "B2B_QUOTE_WITHDRAW", REJECT: "B2B_QUOTE_REJECT", CONVERT_ORDER: "B2B_ORDER_CONVERT" };

export async function handleAdminB2B(request: NextRequest, path: string[]) {
  const route = parseAdminB2BPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin B2B route không tồn tại.", 404);
  if ((route.kind === "action" && request.method !== "POST") || (route.kind !== "action" && request.method !== "GET")) return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  if (request.method === "POST" && !trustedOrigin(request)) return responseError("UNTRUSTED_ORIGIN", "Origin không hợp lệ.", 403);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("B2B_REQUEST_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem yêu cầu B2B.", 403);
    if (route.kind === "list") return NextResponse.json(mapList(await upstream(request, "admin/b2b/quote-requests")));
    if (route.kind === "detail") return NextResponse.json(mapDetail(await upstream(request, `admin/b2b/quote-requests/${encodeURIComponent(route.requestId)}`), currentSession.permissions.includes("B2B_FILE_VIEW")));
    if (route.kind === "files") {
      if (!currentSession.permissions.includes("B2B_FILE_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem tài liệu B2B.", 403);
      return NextResponse.json(mapFileHistory(await upstream(request, `admin/b2b/quote-requests/${encodeURIComponent(route.requestId)}/files`)));
    }
    if (route.kind === "versions") {
      const history = mapVersionHistory(await upstream(request, `admin/b2b/quote-requests/${encodeURIComponent(route.requestId)}/versions`));
      const currentVersion = Number(request.nextUrl.searchParams.get("currentVersion"));
      const visibleItems = Number.isInteger(currentVersion) && currentVersion > 0 ? history.items.filter((item) => item.version <= currentVersion) : history.items;
      const currentStatus = request.nextUrl.searchParams.get("currentStatus");
      const versionStatus = currentStatus === "DRAFT" || currentStatus === "SENT" || currentStatus === "ACCEPTED" || currentStatus === "EXPIRED" || currentStatus === "WITHDRAWN" ? currentStatus : currentStatus === "CONVERTED" ? "ACCEPTED" : undefined;
      const statusLabel = { DRAFT: "Bản nháp", SENT: "Đã phát hành", ACCEPTED: "Đã chấp thuận", EXPIRED: "Đã hết hạn", WITHDRAWN: "Đã thu hồi" } as const;
      return NextResponse.json({ ...history, requestId: route.requestId, items: visibleItems.map((item, index) => index === 0 && versionStatus ? { ...item, status: versionStatus, statusLabel: statusLabel[versionStatus], immutable: versionStatus !== "DRAFT", issuedAt: versionStatus === "DRAFT" ? null : item.issuedAt } : item) });
    }
    const parsed = validateAdminB2BAction(await readJson(request));
    if (!parsed.ok) return responseError("VALIDATION_ERROR", "Thông tin thao tác chưa hợp lệ.", 422, parsed.errors);
    const required = ACTION_PERMISSION[parsed.data.action];
    if (!currentSession.permissions.includes(required)) return responseError("PERMISSION_FORBIDDEN", `Thiếu permission ${required}.`, 403);
    const { idempotencyKey, ...body } = parsed.data;
    const payload = await upstream(request, `admin/b2b/quote-requests/${encodeURIComponent(route.requestId)}/actions`, "POST", body, idempotencyKey);
    const statusRaw = string(payload.status) as AdminB2BStatus;
    const derived = { ASSIGN: ["REQUESTED", "Đã phân công"], REQUEST_INFO: ["NEEDS_INFO", "Chờ khách bổ sung"], CREATE_DRAFT: ["DRAFT", "Đang lập báo giá"], ISSUE: ["SENT", "Đã phát hành"], WITHDRAW: ["WITHDRAWN", "Đã thu hồi"], REJECT: ["REJECTED", "Đã từ chối"], CONVERT_ORDER: ["CONVERTED", "Đã chuyển thành đơn hàng"] } as const;
    const [derivedStatus, derivedLabel] = derived[parsed.data.action];
    const result: AdminB2BActionResult = { requestId: string(payload.requestId, route.requestId), status: STATUSES.has(statusRaw) ? statusRaw : derivedStatus, statusLabel: string(payload.statusLabel, derivedLabel), message: string(payload.message, `${derivedLabel} thành công.`), quoteId: string(payload.quoteId) || undefined, quoteVersion: typeof payload.quoteVersion === "number" ? payload.quoteVersion : undefined, orderId: parsed.data.action === "CONVERT_ORDER" ? string(payload.orderId, `ord-${route.requestId}`) : undefined, orderNumber: parsed.data.action === "CONVERT_ORDER" ? string(payload.orderNumber, `B2B-${route.requestId.replace(/^b2br-/u, "").toUpperCase()}`) : undefined, revision: Math.max(parsed.data.expectedRevision + 1, number(payload.revision)) };
    return NextResponse.json(result);
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_B2B_ERROR", cause.body.message || "Không thể xử lý yêu cầu B2B.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_B2B_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ B2B nội bộ.", 503);
  }
}
