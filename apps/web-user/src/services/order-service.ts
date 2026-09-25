import type { CancelOrderInput, CancelOrderResult, GuestChallenge, GuestVerification, OrderDetail, OrderListResponse, OrderStatus } from "@/lib/orders/types";
import { OrderApiError } from "@/lib/orders/types";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/orders${path}`, { ...init, cache: "no-store", credentials: "include" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new OrderApiError(response.status, payload);
  return payload as T;
}

function scenarioQuery(mockScenario?: string) {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return params;
}

export const orderService = {
  list(input: { q?: string; status?: OrderStatus | "ALL"; year?: number; page?: number; pageSize?: number; mockScenario?: string }) {
    const params = scenarioQuery(input.mockScenario);
    if (input.q) params.set("q", input.q);
    if (input.status && input.status !== "ALL") params.set("status", input.status);
    if (input.year) params.set("year", String(input.year));
    if (input.page) params.set("page", String(input.page));
    if (input.pageSize) params.set("pageSize", String(input.pageSize));
    return request<OrderListResponse>(`?${params}`);
  },
  detail(orderId: string, mockScenario?: string) {
    const params = scenarioQuery(mockScenario);
    return request<OrderDetail>(`/${encodeURIComponent(orderId)}${params.size ? `?${params}` : ""}`);
  },
  cancel(orderId: string, input: CancelOrderInput, mockScenario?: string) {
    const params = scenarioQuery(mockScenario);
    return request<CancelOrderResult>(`/${encodeURIComponent(orderId)}/cancellations${params.size ? `?${params}` : ""}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) });
  },
  createGuestChallenge(input: { orderNumber: string; contact: string }, mockScenario?: string) {
    const params = scenarioQuery(mockScenario);
    return request<GuestChallenge>(`/guest-access/challenges${params.size ? `?${params}` : ""}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) });
  },
  verifyGuestChallenge(challengeId: string, otp: string, mockScenario?: string) {
    const params = scenarioQuery(mockScenario);
    return request<GuestVerification>(`/guest-access/challenges/${encodeURIComponent(challengeId)}/verify${params.size ? `?${params}` : ""}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ otp }) });
  }
};
