const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;

export type PaymentOperation = "instructions" | "status";

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
  if (method !== "GET" || path[0] !== "orders" || !path[1] || !UUID_PATTERN.test(path[1])) {
    return { error: { field: "path", message: "Payment route is not available." } };
  }
  if (path.length === 3 && path[2] === "instructions") return { operation: "instructions", orderId: path[1], mockScenario };
  if (path.length === 3 && path[2] === "status") return { operation: "status", orderId: path[1], mockScenario };
  return { error: { field: "path", message: "Payment route is not available." } };
}
