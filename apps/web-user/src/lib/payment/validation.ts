const ORDER_ID_PATTERN = /^[A-Za-z0-9-]{8,100}$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const IDEMPOTENCY_KEY_PATTERN = /^payment-retry-[0-9a-f-]{36}$/u;

export type PaymentOperation = "summary" | "status" | "retry";

export interface PaymentRouteValidation {
  operation?: PaymentOperation;
  orderId?: string;
  mockScenario?: string;
  error?: { field: string; message: string };
}

export function parsePaymentRoute(method: string, path: string[], params: URLSearchParams): PaymentRouteValidation {
  const unexpected = [...params.keys()].find((key) => key !== "mockScenario");
  if (unexpected) return { error: { field: unexpected, message: "Query không được hỗ trợ cho Payment." } };
  const mockScenario = params.get("mockScenario") || undefined;
  if (mockScenario && !SCENARIO_PATTERN.test(mockScenario)) return { error: { field: "mockScenario", message: "Mock scenario không hợp lệ." } };
  if (path[0] !== "orders" || !path[1] || !ORDER_ID_PATTERN.test(path[1])) return { error: { field: "path", message: "Payment route is not available." } };
  if (method === "GET" && path.length === 2) return { operation: "summary", orderId: path[1], mockScenario };
  if (method === "GET" && path.length === 3 && path[2] === "status") return { operation: "status", orderId: path[1], mockScenario };
  if (method === "POST" && path.length === 3 && path[2] === "attempts") return { operation: "retry", orderId: path[1], mockScenario };
  return { error: { field: "path", message: "Payment route is not available." } };
}

export function validateRetryInput(value: unknown): { idempotencyKey?: string; error?: { field: string; message: string } } {
  if (!value || typeof value !== "object" || Array.isArray(value)) return { error: { field: "body", message: "Yêu cầu tạo lại Payment không hợp lệ." } };
  const record = value as Record<string, unknown>;
  if (Object.keys(record).some((key) => key !== "idempotencyKey") || typeof record.idempotencyKey !== "string" || !IDEMPOTENCY_KEY_PATTERN.test(record.idempotencyKey)) {
    return { error: { field: "idempotencyKey", message: "Khóa chống gửi lặp không hợp lệ." } };
  }
  return { idempotencyKey: record.idempotencyKey };
}
