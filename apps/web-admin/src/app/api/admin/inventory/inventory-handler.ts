import { NextRequest, NextResponse } from "next/server";
import { parseAdminSession, type AdminPermission, type AdminSession } from "@/lib/auth/types";
import type { BatchExpiryState, InventoryBalance, InventoryBatch, InventoryMovement, InventoryMovementType, InventoryOperationResult, InventoryRiskState, InventorySkuDetail, InventorySkuList, InventorySkuSummary, WarehouseWorkbenchBatch, WarehouseWorkbenchContext, WarehouseWorkbenchSku } from "@/lib/inventory/types";
import { parseAdminInventoryPath, validateInventoryAdjustment, validateInventoryIssue, validateInventoryReceipt } from "@/lib/inventory/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin inventory upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const RISK_STATES = new Set<InventoryRiskState>(["NORMAL", "EXPIRING", "EXPIRED_ONLY"]);
const EXPIRY_STATES = new Set<BatchExpiryState>(["SAFE", "EXPIRING", "EXPIRED"]);
const MOVEMENT_TYPES = new Set<InventoryMovementType>(["RECEIPT", "RESERVATION", "RELEASE", "ISSUE", "ADJUSTMENT", "DAMAGE", "LOSS"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) { return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-INVENTORY-${code}` }, { status }); }
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest, body = false, idempotencyKey?: string) { const result = new Headers({ Accept: "application/json" }); const authorization = request.headers.get("authorization"); if (authorization) result.set("Authorization", authorization); const profile = mockProfile(); if (profile) result.set("X-Admin-Profile", profile); if (body) result.set("Content-Type", "application/json"); if (idempotencyKey) result.set("Idempotency-Key", idempotencyKey); return result; }
async function upstream(request: NextRequest, path: string, method = "GET", body?: object, idempotencyKey?: string) { const response = await fetch(`${baseUrl()}/${path}`, { method, headers: headers(request, body !== undefined, idempotencyKey), body: body ? JSON.stringify(body) : undefined, cache: "no-store", redirect: "manual" }); const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {}; if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {}); return record(payload) ? payload : {}; }
async function session(request: NextRequest): Promise<AdminSession> { const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" }); if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." }); const parsed = parseAdminSession(await response.json()); if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." }); return parsed; }
function trustedOrigin(request: NextRequest) { const origin = request.headers.get("origin"); if (!origin) return true; try { const source = new URL(origin); return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host; } catch { return false; } }
async function readJson(request: NextRequest): Promise<unknown> { try { return await request.json(); } catch { return undefined; } }
function balance(value: unknown): InventoryBalance { const item = record(value) ? value : {}; return { onHand: Math.max(0, number(item.onHand)), reserved: Math.max(0, number(item.reserved)), nonSellable: Math.max(0, number(item.nonSellable)), available: Math.max(0, number(item.available)) }; }
function riskState(value: unknown): InventoryRiskState { const state = string(value) as InventoryRiskState; return RISK_STATES.has(state) ? state : "NORMAL"; }
function expiryState(value: unknown): BatchExpiryState { const state = string(value) as BatchExpiryState; return EXPIRY_STATES.has(state) ? state : "SAFE"; }
function movementType(value: unknown): InventoryMovementType { const type = string(value) as InventoryMovementType; return MOVEMENT_TYPES.has(type) ? type : "ADJUSTMENT"; }
function mapSummary(value: unknown): InventorySkuSummary { const item = record(value) ? value : {}; return { skuId: string(item.skuId), skuCode: string(item.skuCode), skuLabel: string(item.skuLabel), productId: string(item.productId), productName: string(item.productName), categoryId: string(item.categoryId), categoryName: string(item.categoryName), scopeLabel: string(item.scopeLabel), riskState: riskState(item.riskState), riskLabel: string(item.riskLabel), batchCount: Math.max(0, number(item.batchCount)), expiringBatchCount: Math.max(0, number(item.expiringBatchCount)), expiredBatchCount: Math.max(0, number(item.expiredBatchCount)), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1)), ...balance(item) }; }
function mapList(payload: Record<string, unknown>): InventorySkuList { const items = array(payload.items).map(mapSummary).filter((item) => item.skuId); const riskCounts = array(payload.riskCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.riskState) as "ALL" | InventoryRiskState; return { riskState: raw === "ALL" || RISK_STATES.has(raw as InventoryRiskState) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; }); for (const state of RISK_STATES) if (!riskCounts.some((item) => item.riskState === state)) riskCounts.push({ riskState: state, label: state, count: items.filter((item) => item.riskState === state).length }); const categories = array(payload.categories).map((value) => { const item = record(value) ? value : {}; return { categoryId: string(item.categoryId), categoryName: string(item.categoryName) }; }).filter((item) => item.categoryId); return { items, riskCounts, categories, calculatedAt: date(payload.calculatedAt) }; }
function mapMovement(value: unknown): InventoryMovement { const item = record(value) ? value : {}; return { movementId: string(item.movementId), type: movementType(item.type), typeLabel: string(item.typeLabel), quantity: number(item.quantity), occurredAt: date(item.occurredAt), actorLabel: string(item.actorLabel), reference: string(item.reference) }; }
function mapBatch(value: unknown): InventoryBatch { const item = record(value) ? value : {}; return { batchId: string(item.batchId), batchCode: string(item.batchCode), manufacturingDate: date(item.manufacturingDate), expiresAt: date(item.expiresAt), expiryState: expiryState(item.expiryState), expiryLabel: string(item.expiryLabel), sourceReference: string(item.sourceReference), balance: balance(item.balance), movements: array(item.movements).map(mapMovement).filter((movement) => movement.movementId) }; }
function mapDetail(payload: Record<string, unknown>): InventorySkuDetail { return { skuId: string(payload.skuId), skuCode: string(payload.skuCode), skuLabel: string(payload.skuLabel), productId: string(payload.productId), productName: string(payload.productName), scopeLabel: string(payload.scopeLabel), balance: balance(payload.balance), batches: array(payload.batches).map(mapBatch).filter((batch) => batch.batchId), calculatedAt: date(payload.calculatedAt), revision: Math.max(1, number(payload.revision, 1)) }; }
function mapWorkbenchBatch(value: unknown): WarehouseWorkbenchBatch { const item = record(value) ? value : {}; return { batchId: string(item.batchId), batchCode: string(item.batchCode), manufacturingDate: date(item.manufacturingDate), expiresAt: date(item.expiresAt), expiryState: expiryState(item.expiryState), expiryLabel: string(item.expiryLabel), balance: balance(item.balance) }; }
function mapWorkbenchSku(value: unknown): WarehouseWorkbenchSku { const item = record(value) ? value : {}; return { skuId: string(item.skuId), skuCode: string(item.skuCode), skuLabel: string(item.skuLabel), productName: string(item.productName), balance: balance(item.balance), batches: array(item.batches).map(mapWorkbenchBatch).filter((batch) => batch.batchId), revision: Math.max(1, number(item.revision, 1)) }; }
function mapWorkbench(payload: Record<string, unknown>): WarehouseWorkbenchContext { return { skus: array(payload.skus).map(mapWorkbenchSku).filter((item) => item.skuId), recentMovements: array(payload.recentMovements).map((value) => { const item = record(value) ? value : {}; return { ...mapMovement(item), skuCode: string(item.skuCode), batchCode: string(item.batchCode) }; }).filter((item) => item.movementId), pendingApprovalCount: Math.max(0, number(payload.pendingApprovalCount)), calculatedAt: date(payload.calculatedAt), policyNotice: string(payload.policyNotice) }; }
function mapOperation(payload: Record<string, unknown>): InventoryOperationResult { const status = payload.status === "PENDING_APPROVAL" ? "PENDING_APPROVAL" as const : "APPLIED" as const; return { operationId: string(payload.operationId), status, statusLabel: string(payload.statusLabel, status === "APPLIED" ? "Đã áp dụng" : "Chờ phê duyệt"), movementId: typeof payload.movementId === "string" && payload.movementId ? payload.movementId : null, approvalRequestId: typeof payload.approvalRequestId === "string" && payload.approvalRequestId ? payload.approvalRequestId : null, occurredAt: date(payload.occurredAt), revision: Math.max(1, number(payload.revision, 1)), message: string(payload.message, status === "APPLIED" ? "Đã ghi nhận biến động kho." : "Yêu cầu đã được chuyển sang chờ phê duyệt.") }; }
function hasScope(current: AdminSession) { return current.scopes.some((scope) => scope.resource === "INVENTORY" && (scope.level === "ALL" || scope.level === "ASSIGNED_ONLY")); }
function permissionError(current: AdminSession, permission: AdminPermission, message: string) { return current.permissions.includes(permission) ? null : responseError("PERMISSION_FORBIDDEN", message, 403); }

export async function handleAdminInventory(request: NextRequest, path: string[]) {
  const route = parseAdminInventoryPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin inventory route không tồn tại.", 404);
  const expectedMethod = route.kind === "receipt-create" || route.kind === "issue-create" || route.kind === "adjustment-create" ? "POST" : "GET";
  if (request.method !== expectedMethod) return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  if (request.method !== "GET" && !trustedOrigin(request)) return responseError("UNTRUSTED_ORIGIN", "Origin không hợp lệ.", 403);
  try {
    const current = await session(request);
    const viewError = permissionError(current, "INVENTORY_VIEW", "Nhân viên không có quyền xem dữ liệu tồn kho.");
    if (viewError) return viewError;
    if (route.kind === "sku-list" || route.kind === "sku-detail") {
      const managementError = permissionError(current, "INVENTORY_MANAGEMENT_VIEW", "Nhân viên không có quyền truy cập trang quản lý tồn kho.");
      if (managementError) return managementError;
      if (route.kind === "sku-list") return NextResponse.json(mapList(await upstream(request, "admin/inventory/skus")));
      const batchError = permissionError(current, "BATCH_VIEW", "Nhân viên không có quyền xem chi tiết Batch/Lot.");
      if (batchError) return batchError;
      return NextResponse.json(mapDetail(await upstream(request, `admin/inventory/skus/${encodeURIComponent(route.skuId)}`)));
    }
    const workbenchError = permissionError(current, "WAREHOUSE_WORKBENCH_VIEW", "Nhân viên không có quyền truy cập Warehouse Workbench.");
    if (workbenchError) return workbenchError;
    if (!hasScope(current)) return responseError("SCOPE_FORBIDDEN", "Nhân viên không có phạm vi dữ liệu Inventory.", 403);
    if (route.kind === "workbench-context") return NextResponse.json(mapWorkbench(await upstream(request, "admin/inventory/workbench")));
    const source = await readJson(request);
    const validation = route.kind === "receipt-create" ? validateInventoryReceipt(source) : route.kind === "issue-create" ? validateInventoryIssue(source) : validateInventoryAdjustment(source);
    if (!validation.ok) return responseError("VALIDATION_ERROR", "Dữ liệu biến động kho chưa hợp lệ.", 422, validation.errors);
    const requiredPermission: AdminPermission = route.kind === "receipt-create" ? "INVENTORY_RECEIPT_CREATE" : route.kind === "issue-create" ? "INVENTORY_ISSUE_CREATE" : "INVENTORY_ADJUSTMENT_CREATE";
    const actionError = permissionError(current, requiredPermission, "Nhân viên không có quyền thực hiện loại biến động kho này.");
    if (actionError) return actionError;
    const { idempotencyKey, ...body } = validation.data;
    const endpoint = route.kind === "receipt-create" ? "receipts" : route.kind === "issue-create" ? "issues" : "adjustments";
    return NextResponse.json(mapOperation(await upstream(request, `admin/inventory/workbench/${endpoint}`, "POST", body, idempotencyKey)));
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_INVENTORY_ERROR", cause.body.message || "Không thể xử lý yêu cầu tồn kho.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_INVENTORY_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ tồn kho nội bộ.", 503);
  }
}
