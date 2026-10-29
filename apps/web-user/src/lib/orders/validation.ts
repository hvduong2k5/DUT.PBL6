import type { CancelOrderRequest } from "@/lib/api-contract";

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu;
const CHALLENGE_ID_PATTERN = /^[A-Za-z0-9-]{8,100}$/u;
const ORDER_NUMBER_PATTERN = /^(?:OMA|ORD)-[A-Z0-9-]{6,30}$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/u;
const PHONE_PATTERN = /^(?:\+?84|0)[0-9]{9,10}$/u;

export type OrderOperation = "list" | "detail" | "tracking" | "cancel" | "guest-challenge" | "guest-verify";

export interface OrderRouteValidation {
  operation?: OrderOperation;
  orderId?: string;
  challengeId?: string;
  mockScenario?: string;
  error?: { field: string; message: string };
}

function scenario(params: URLSearchParams) {
  const value = params.get("mockScenario") || undefined;
  return value && !SCENARIO_PATTERN.test(value) ? undefined : value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function parseOrderRoute(method: string, path: string[], params: URLSearchParams): OrderRouteValidation {
  const mockScenario = scenario(params);
  if (params.has("mockScenario") && !mockScenario) return { error: { field: "mockScenario", message: "Mock scenario không hợp lệ." } };
  const unexpected = [...params.keys()].find((key) => key !== "mockScenario");
  if (unexpected) return { error: { field: unexpected, message: "Query không thuộc public Order contract." } };

  if (method === "GET" && path.length === 0) return { operation: "list", mockScenario };
  if (method === "POST" && path.length === 2 && path[0] === "guest-access" && path[1] === "challenges") {
    return { operation: "guest-challenge", mockScenario };
  }
  if (method === "POST" && path.length === 4 && path[0] === "guest-access" && path[1] === "challenges" && path[3] === "verify" && CHALLENGE_ID_PATTERN.test(path[2])) {
    return { operation: "guest-verify", challengeId: path[2], mockScenario };
  }
  if (!path[0] || !UUID_PATTERN.test(path[0])) return { error: { field: "path", message: "Order route is not available." } };
  if (method === "GET" && path.length === 1) return { operation: "detail", orderId: path[0], mockScenario };
  if (method === "GET" && path.length === 2 && path[1] === "tracking") return { operation: "tracking", orderId: path[0], mockScenario };
  if (method === "POST" && path.length === 2 && path[1] === "cancel") return { operation: "cancel", orderId: path[0], mockScenario };
  return { error: { field: "path", message: "Order route is not available." } };
}

export function validateMvpCancelRequest(value: unknown): { data?: CancelOrderRequest; errors: Array<{ field: string; message: string }> } {
  if (!isRecord(value) || Object.keys(value).some((key) => !["active", "reason"].includes(key))) {
    return { errors: [{ field: "body", message: "Yêu cầu hủy không đúng contract." }] };
  }
  const errors: Array<{ field: string; message: string }> = [];
  if (value.active !== true) errors.push({ field: "active", message: "Cancellation intent phải ở trạng thái active." });
  if (typeof value.reason !== "string" || value.reason !== value.reason.trim() || value.reason.length < 1 || value.reason.length > 255) {
    errors.push({ field: "reason", message: "Lý do hủy cần từ 1 đến 255 ký tự." });
  }
  return errors.length ? { errors } : { data: value as unknown as CancelOrderRequest, errors };
}

export function validateGuestChallenge(value: unknown) {
  if (!isRecord(value)) return { errors: [{ field: "body", message: "Thông tin tra cứu không hợp lệ." }] };
  const orderNumber = typeof value.orderNumber === "string" ? value.orderNumber.trim().replace(/^#/u, "").toUpperCase() : "";
  const contact = typeof value.contact === "string" ? value.contact.trim() : "";
  const normalizedPhone = contact.replace(/[\s.-]/gu, "");
  const errors: Array<{ field: string; message: string }> = [];
  if (Object.keys(value).some((key) => !["orderNumber", "contact"].includes(key))) errors.push({ field: "body", message: "Thông tin chứa trường không được hỗ trợ." });
  if (!ORDER_NUMBER_PATTERN.test(orderNumber)) errors.push({ field: "orderNumber", message: "Mã Order không đúng định dạng." });
  if (!EMAIL_PATTERN.test(contact.toLowerCase()) && !PHONE_PATTERN.test(normalizedPhone)) errors.push({ field: "contact", message: "Nhập email hoặc số điện thoại đã dùng đặt hàng." });
  return errors.length ? { errors } : { data: { orderNumber, contact: EMAIL_PATTERN.test(contact.toLowerCase()) ? contact.toLowerCase() : normalizedPhone }, errors };
}

export function validateGuestOtp(value: unknown) {
  if (!isRecord(value)) return { errors: [{ field: "body", message: "Mã xác minh không hợp lệ." }] };
  const otp = typeof value.otp === "string" ? value.otp.trim() : "";
  if (Object.keys(value).some((key) => key !== "otp") || !/^\d{6}$/u.test(otp)) return { errors: [{ field: "otp", message: "Mã xác minh gồm đúng 6 chữ số." }] };
  return { data: { otp }, errors: [] };
}
