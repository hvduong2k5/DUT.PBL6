import type { CancelOrderInput, CancelReasonCode, OrderStatus } from "./types";

const ORDER_ID_PATTERN = /^[A-Za-z0-9-]{8,100}$/u;
const CHALLENGE_ID_PATTERN = /^[A-Za-z0-9-]{8,100}$/u;
const ORDER_NUMBER_PATTERN = /^OMA-[A-Z0-9-]{6,30}$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const IDEMPOTENCY_PATTERN = /^order-cancel-[0-9a-f-]{36}$/u;
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/u;
const PHONE_PATTERN = /^(?:\+?84|0)[0-9]{9,10}$/u;
const ORDER_STATUSES = new Set<OrderStatus>(["PENDING_PAYMENT", "PAID", "CONFIRMED", "PROCESSING", "PACKED", "SHIPPED", "DELIVERED", "COMPLETED", "DELIVERY_FAILED", "EXPIRED", "CANCELLED"]);
const REASONS = new Set<CancelReasonCode>(["CHANGED_MIND", "WRONG_INFORMATION", "OTHER"]);

export type OrderOperation = "list" | "detail" | "cancel" | "guest-challenge" | "guest-verify";

export interface OrderRouteValidation {
  operation?: OrderOperation;
  orderId?: string;
  challengeId?: string;
  mockScenario?: string;
  filters?: { q: string; status: OrderStatus | "ALL"; year?: number; page: number; pageSize: number };
  error?: { field: string; message: string };
}

function scenario(params: URLSearchParams) {
  const value = params.get("mockScenario") || undefined;
  return value && !SCENARIO_PATTERN.test(value) ? undefined : value;
}

export function parseOrderRoute(method: string, path: string[], params: URLSearchParams): OrderRouteValidation {
  const mockScenario = scenario(params);
  if (params.has("mockScenario") && !mockScenario) return { error: { field: "mockScenario", message: "Mock scenario không hợp lệ." } };

  if (method === "GET" && path.length === 0) {
    const allowed = new Set(["q", "status", "year", "page", "pageSize", "mockScenario"]);
    const unexpected = [...params.keys()].find((key) => !allowed.has(key));
    if (unexpected) return { error: { field: unexpected, message: "Query không được hỗ trợ cho Order list." } };
    const q = (params.get("q") ?? "").trim();
    if (q && (q.length < 2 || q.length > 80)) return { error: { field: "q", message: "Từ khóa phải có 2–80 ký tự." } };
    const statusValue = params.get("status") ?? "ALL";
    if (statusValue !== "ALL" && !ORDER_STATUSES.has(statusValue as OrderStatus)) return { error: { field: "status", message: "Trạng thái Order không hợp lệ." } };
    const currentYear = new Date().getFullYear();
    const yearValue = params.get("year");
    const year = yearValue ? Number(yearValue) : undefined;
    if (year !== undefined && (!Number.isInteger(year) || year < 2020 || year > currentYear)) return { error: { field: "year", message: "Năm tra cứu không hợp lệ." } };
    const page = Number(params.get("page") ?? "1");
    const pageSize = Number(params.get("pageSize") ?? "10");
    if (!Number.isInteger(page) || page < 1 || page > 1000) return { error: { field: "page", message: "Trang không hợp lệ." } };
    if (!Number.isInteger(pageSize) || pageSize < 1 || pageSize > 20) return { error: { field: "pageSize", message: "Kích thước trang không hợp lệ." } };
    return { operation: "list", mockScenario, filters: { q, status: statusValue as OrderStatus | "ALL", year, page, pageSize } };
  }

  const allowed = new Set(["mockScenario"]);
  const unexpected = [...params.keys()].find((key) => !allowed.has(key));
  if (unexpected) return { error: { field: unexpected, message: "Query không được hỗ trợ cho Order." } };
  if (method === "POST" && path.length === 2 && path[0] === "guest-access" && path[1] === "challenges") return { operation: "guest-challenge", mockScenario };
  if (method === "POST" && path.length === 4 && path[0] === "guest-access" && path[1] === "challenges" && path[3] === "verify" && CHALLENGE_ID_PATTERN.test(path[2])) return { operation: "guest-verify", challengeId: path[2], mockScenario };
  if (!path[0] || !ORDER_ID_PATTERN.test(path[0])) return { error: { field: "path", message: "Order route is not available." } };
  if (method === "GET" && path.length === 1) return { operation: "detail", orderId: path[0], mockScenario };
  if (method === "POST" && path.length === 2 && path[1] === "cancellations") return { operation: "cancel", orderId: path[0], mockScenario };
  return { error: { field: "path", message: "Order route is not available." } };
}

export function validateCancelOrder(value: unknown): { data?: CancelOrderInput; errors: Array<{ field: string; message: string }> } {
  if (!value || typeof value !== "object" || Array.isArray(value)) return { errors: [{ field: "body", message: "Yêu cầu hủy không hợp lệ." }] };
  const source = value as Record<string, unknown>;
  const errors: Array<{ field: string; message: string }> = [];
  if (Object.keys(source).some((key) => !["reasonCode", "note", "idempotencyKey"].includes(key))) errors.push({ field: "body", message: "Yêu cầu chứa trường không được hỗ trợ." });
  if (typeof source.reasonCode !== "string" || !REASONS.has(source.reasonCode as CancelReasonCode)) errors.push({ field: "reasonCode", message: "Hãy chọn lý do hủy hợp lệ." });
  const note = typeof source.note === "string" ? source.note.trim() : "";
  if (source.note !== undefined && typeof source.note !== "string" || note.length > 300) errors.push({ field: "note", message: "Ghi chú tối đa 300 ký tự." });
  if (source.reasonCode === "OTHER" && note.length < 5) errors.push({ field: "note", message: "Hãy mô tả lý do khác tối thiểu 5 ký tự." });
  if (typeof source.idempotencyKey !== "string" || !IDEMPOTENCY_PATTERN.test(source.idempotencyKey)) errors.push({ field: "idempotencyKey", message: "Khóa chống gửi lặp không hợp lệ." });
  if (errors.length) return { errors };
  return { data: { reasonCode: source.reasonCode as CancelReasonCode, note: note || undefined, idempotencyKey: source.idempotencyKey as string }, errors: [] };
}

export function validateGuestChallenge(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return { errors: [{ field: "body", message: "Thông tin tra cứu không hợp lệ." }] };
  const source = value as Record<string, unknown>;
  const orderNumber = typeof source.orderNumber === "string" ? source.orderNumber.trim().replace(/^#/u, "").toUpperCase() : "";
  const contact = typeof source.contact === "string" ? source.contact.trim() : "";
  const normalizedPhone = contact.replace(/[\s.-]/gu, "");
  const errors: Array<{ field: string; message: string }> = [];
  if (Object.keys(source).some((key) => !["orderNumber", "contact"].includes(key))) errors.push({ field: "body", message: "Thông tin chứa trường không được hỗ trợ." });
  if (!ORDER_NUMBER_PATTERN.test(orderNumber)) errors.push({ field: "orderNumber", message: "Mã Order không đúng định dạng." });
  if (!EMAIL_PATTERN.test(contact.toLowerCase()) && !PHONE_PATTERN.test(normalizedPhone)) errors.push({ field: "contact", message: "Nhập email hoặc số điện thoại đã dùng đặt hàng." });
  return errors.length ? { errors } : { data: { orderNumber, contact: EMAIL_PATTERN.test(contact.toLowerCase()) ? contact.toLowerCase() : normalizedPhone }, errors };
}

export function validateGuestOtp(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return { errors: [{ field: "body", message: "Mã xác minh không hợp lệ." }] };
  const source = value as Record<string, unknown>;
  const otp = typeof source.otp === "string" ? source.otp.trim() : "";
  if (Object.keys(source).some((key) => key !== "otp") || !/^\d{6}$/u.test(otp)) return { errors: [{ field: "otp", message: "Mã xác minh gồm đúng 6 chữ số." }] };
  return { data: { otp }, errors: [] };
}
