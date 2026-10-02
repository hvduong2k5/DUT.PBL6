import { NextRequest, NextResponse } from "next/server";
import { parseAdminSession, type AdminSession } from "@/lib/auth/types";
import { PACKING_TASK_STATUSES, type PackingChecklistState, type PackingChecklistUpdateResult, type PackingPriority, type PackingSlaState, type PackingTaskCompleteResult, type PackingTaskDetail, type PackingTaskList, type PackingTaskStatus, type PackingTaskSummary } from "@/lib/packing/types";
import { parseAdminPackingPath, validatePackingChecklistUpdate, validatePackingTaskComplete } from "@/lib/packing/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin packing upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const nullableDate = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : null; };
const nullableString = (value: unknown) => typeof value === "string" && value.trim() ? value : null;
const STATUSES = new Set<PackingTaskStatus>(PACKING_TASK_STATUSES);
const PRIORITIES = new Set<PackingPriority>(["NORMAL", "HIGH", "URGENT"]);
const SLA_STATES = new Set<PackingSlaState>(["ON_TRACK", "AT_RISK", "OVERDUE", "COMPLETE"]);
const CHECK_STATES = new Set<PackingChecklistState>(["PENDING", "PASSED", "FAILED"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) { return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-PACKING-${code}` }, { status }); }
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest, body = false, idempotencyKey?: string) { const result = new Headers({ Accept: "application/json" }); const authorization = request.headers.get("authorization"); if (authorization) result.set("Authorization", authorization); const profile = mockProfile(); if (profile) result.set("X-Admin-Profile", profile); if (body) result.set("Content-Type", "application/json"); if (idempotencyKey) result.set("Idempotency-Key", idempotencyKey); return result; }
async function upstream(request: NextRequest, path: string, method = "GET", body?: object, idempotencyKey?: string) { const response = await fetch(`${baseUrl()}/${path}`, { method, headers: headers(request, body !== undefined, idempotencyKey), body: body ? JSON.stringify(body) : undefined, cache: "no-store", redirect: "manual" }); const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {}; if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {}); return record(payload) ? payload : {}; }
async function session(request: NextRequest): Promise<AdminSession> { const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" }); if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." }); const parsed = parseAdminSession(await response.json()); if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." }); return parsed; }
function trustedOrigin(request: NextRequest) { const origin = request.headers.get("origin"); if (!origin) return true; try { const source = new URL(origin); return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host; } catch { return false; } }
async function readJson(request: NextRequest): Promise<unknown> { try { return await request.json(); } catch { return undefined; } }
function assignedTo(detail: PackingTaskDetail, currentSession: AdminSession) { return detail.assignee?.employeeId === currentSession.employee.employeeId; }
function taskStatus(value: unknown): PackingTaskStatus { const result = string(value) as PackingTaskStatus; return STATUSES.has(result) ? result : "READY"; }
function priority(value: unknown): PackingPriority { const result = string(value) as PackingPriority; return PRIORITIES.has(result) ? result : "NORMAL"; }
function slaState(value: unknown): PackingSlaState { const result = string(value) as PackingSlaState; return SLA_STATES.has(result) ? result : "ON_TRACK"; }
function checklistState(value: unknown): PackingChecklistState { const result = string(value) as PackingChecklistState; return CHECK_STATES.has(result) ? result : "PENDING"; }
function mapSummary(value: unknown): PackingTaskSummary {
  const item = record(value) ? value : {}; const assignee = record(item.assignee) ? item.assignee : null;
  return { taskId: string(item.taskId), taskNumber: string(item.taskNumber), orderId: string(item.orderId), orderNumber: string(item.orderNumber), orderSource: string(item.orderSource), orderSourceLabel: string(item.orderSourceLabel), status: taskStatus(item.status), statusLabel: string(item.statusLabel), assignee: assignee && string(assignee.employeeId) ? { employeeId: string(assignee.employeeId), displayName: string(assignee.displayName) } : null, lineCount: Math.max(0, number(item.lineCount)), totalQuantity: Math.max(0, number(item.totalQuantity)), priority: priority(item.priority), priorityLabel: string(item.priorityLabel), slaState: slaState(item.slaState), slaLabel: string(item.slaLabel), dueAt: nullableDate(item.dueAt), blockedReason: nullableString(item.blockedReason), createdAt: date(item.createdAt), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1)) };
}
function mapList(payload: Record<string, unknown>): PackingTaskList {
  const items = array(payload.items).map(mapSummary).filter((item) => item.taskId);
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | PackingTaskStatus; return { status: raw === "ALL" || STATUSES.has(raw as PackingTaskStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  if (!statusCounts.some((item) => item.status === "ALL")) statusCounts.unshift({ status: "ALL", label: "Tất cả", count: items.length });
  const sources = array(payload.sources).map((value) => { const item = record(value) ? value : {}; return { code: string(item.code), label: string(item.label) }; }).filter((item) => item.code);
  const assignees = array(payload.assignees).map((value) => { const item = record(value) ? value : {}; return { employeeId: string(item.employeeId), displayName: string(item.displayName) }; }).filter((item) => item.employeeId);
  return { items, statusCounts, sources, assignees, calculatedAt: date(payload.calculatedAt) };
}
function mapDetail(payload: Record<string, unknown>): PackingTaskDetail {
  const readiness = record(payload.completionReadiness) ? payload.completionReadiness : {};
  return { ...mapSummary(payload),
    prerequisites: array(payload.prerequisites).map((value) => { const item = record(value) ? value : {}; return { code: string(item.code), label: string(item.label), ready: item.ready === true, detail: string(item.detail) }; }).filter((item) => item.code),
    pickLines: array(payload.pickLines).map((value) => { const item = record(value) ? value : {}; return { lineId: string(item.lineId), productName: string(item.productName), skuCode: string(item.skuCode), skuLabel: string(item.skuLabel), quantity: Math.max(0, number(item.quantity)), allocations: array(item.allocations).map((allocationValue) => { const allocation = record(allocationValue) ? allocationValue : {}; return { allocationId: string(allocation.allocationId), batchCode: string(allocation.batchCode), locationLabel: nullableString(allocation.locationLabel), quantity: Math.max(0, number(allocation.quantity)), expiresAt: nullableDate(allocation.expiresAt), eligibilityLabel: string(allocation.eligibilityLabel), eligible: allocation.eligible === true }; }).filter((allocation) => allocation.allocationId) }; }).filter((item) => item.lineId),
    checklist: array(payload.checklist).map((value) => { const item = record(value) ? value : {}; return { itemId: string(item.itemId), label: string(item.label), required: item.required === true, state: checklistState(item.state), stateLabel: string(item.stateLabel), confirmedBy: nullableString(item.confirmedBy), confirmedAt: nullableDate(item.confirmedAt), note: nullableString(item.note) }; }).filter((item) => item.itemId),
    completionReadiness: { ready: readiness.ready === true, label: string(readiness.label), missingConditions: array(readiness.missingConditions).map((item) => string(item)).filter(Boolean) },
    history: array(payload.history).map((value) => { const item = record(value) ? value : {}; return { eventId: string(item.eventId), label: string(item.label), occurredAt: date(item.occurredAt), actorLabel: string(item.actorLabel), detail: nullableString(item.detail) }; }).filter((item) => item.eventId)
  };
}

export async function handleAdminPacking(request: NextRequest, path: string[]) {
  const route = parseAdminPackingPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin packing route không tồn tại.", 404);
  const expectedMethod = route.kind === "checklist-update" ? "PATCH" : route.kind === "task-complete" ? "POST" : "GET";
  if (request.method !== expectedMethod) return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  if (request.method !== "GET" && !trustedOrigin(request)) return responseError("UNTRUSTED_ORIGIN", "Origin không hợp lệ.", 403);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("PACKING_TASK_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem Packing Task.", 403);
    if (route.kind === "task-list" || route.kind === "task-detail") {
      if (!currentSession.permissions.includes("PACKING_MANAGEMENT_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền truy cập trang quản lý đóng gói.", 403);
      if (route.kind === "task-list") return NextResponse.json(mapList(await upstream(request, "admin/packing/tasks")));
      return NextResponse.json(mapDetail(await upstream(request, `admin/packing/tasks/${encodeURIComponent(route.taskId)}`)));
    }
    if (!currentSession.permissions.includes("PACKING_WORKBENCH_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền truy cập Workbench đóng gói.", 403);
    if (!currentSession.scopes.some((scope) => scope.resource === "PACKING_TASK" && (scope.level === "ASSIGNED_ONLY" || scope.level === "ALL"))) return responseError("SCOPE_FORBIDDEN", "Nhân viên không có phạm vi dữ liệu Packing Task.", 403);
    if (route.kind === "workbench-list") {
      const result = mapList(await upstream(request, "admin/packing/workbench/tasks"));
      return NextResponse.json({ ...result, items: result.items.filter((item) => item.assignee?.employeeId === currentSession.employee.employeeId), assignees: [{ employeeId: currentSession.employee.employeeId, displayName: currentSession.employee.displayName }] });
    }
    const detail = mapDetail(await upstream(request, `admin/packing/workbench/tasks/${encodeURIComponent(route.taskId)}`));
    if (!assignedTo(detail, currentSession)) return responseError("PACKING_TASK_NOT_ASSIGNED", "Công việc này không được phân công cho bạn.", 403);
    if (route.kind === "workbench-detail") return NextResponse.json(detail);
    if (route.kind === "checklist-update") {
      if (!currentSession.permissions.includes("PACKING_CHECKLIST_UPDATE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền cập nhật checklist đóng gói.", 403);
      if (detail.status !== "IN_PROGRESS") return responseError("PACKING_TASK_NOT_IN_PROGRESS", "Chỉ có thể cập nhật checklist của Task đang đóng gói.", 409);
      if (!detail.checklist.some((item) => item.itemId === route.itemId)) return responseError("PACKING_CHECKLIST_ITEM_NOT_FOUND", "Bước checklist không tồn tại.", 404);
      const parsed = validatePackingChecklistUpdate(await readJson(request));
      if (!parsed.ok) return responseError("VALIDATION_ERROR", "Kết quả checklist chưa hợp lệ.", 422, parsed.errors);
      const { idempotencyKey, ...body } = parsed.data;
      await upstream(request, `admin/packing/workbench/tasks/${encodeURIComponent(route.taskId)}/checklist/${encodeURIComponent(route.itemId)}`, "PATCH", body, idempotencyKey);
      const now = new Date().toISOString();
      const sourceItem = detail.checklist.find((item) => item.itemId === route.itemId)!;
      const result: PackingChecklistUpdateResult = { taskId: route.taskId, item: { ...sourceItem, state: parsed.data.state, stateLabel: parsed.data.state === "PASSED" ? "Đã đạt" : "Không đạt", confirmedBy: currentSession.employee.displayName, confirmedAt: now, note: parsed.data.note ?? null }, revision: parsed.data.expectedRevision + 1, message: "Đã lưu kết quả bước kiểm tra." };
      return NextResponse.json(result);
    }
    if (!currentSession.permissions.includes("PACKING_TASK_COMPLETE")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền hoàn tất Packing Task.", 403);
    const parsed = validatePackingTaskComplete(await readJson(request));
    if (!parsed.ok) return responseError("VALIDATION_ERROR", "Yêu cầu hoàn tất chưa hợp lệ.", 422, parsed.errors);
    const { idempotencyKey, ...body } = parsed.data;
    const payload = await upstream(request, `admin/packing/workbench/tasks/${encodeURIComponent(route.taskId)}/complete`, "POST", body, idempotencyKey);
    const completedAt = date(payload.completedAt);
    const result: PackingTaskCompleteResult = { taskId: route.taskId, status: "COMPLETED", statusLabel: "Đã hoàn tất", completedAt, completedBy: currentSession.employee.displayName, revision: parsed.data.expectedRevision + 1, message: string(payload.message, "Packing Task đã hoàn tất và được bàn giao sang quy trình tiếp theo.") };
    return NextResponse.json(result);
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_PACKING_ERROR", cause.body.message || "Không thể xử lý yêu cầu quản lý đóng gói.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_PACKING_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ đóng gói nội bộ.", 503);
  }
}
