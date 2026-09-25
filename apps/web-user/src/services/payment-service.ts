import type { PaymentRetryResult, PaymentStatusResult, PaymentSummary } from "@/lib/payment/types";
import { PaymentApiError } from "@/lib/payment/types";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/payments/${path}`, { ...init, cache: "no-store" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new PaymentApiError(response.status, payload);
  return payload as T;
}

function pathFor(orderId: string, suffix: string, mockScenario?: string) {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return `orders/${encodeURIComponent(orderId)}${suffix}${params.size ? `?${params}` : ""}`;
}

export const paymentService = {
  getSummary(orderId: string, mockScenario?: string) {
    return request<PaymentSummary>(pathFor(orderId, "", mockScenario));
  },
  getStatus(orderId: string, mockScenario?: string) {
    return request<PaymentStatusResult>(pathFor(orderId, "/status", mockScenario));
  },
  retry(orderId: string, idempotencyKey: string, mockScenario?: string) {
    return request<PaymentRetryResult>(pathFor(orderId, "/attempts", mockScenario), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ idempotencyKey })
    });
  }
};
