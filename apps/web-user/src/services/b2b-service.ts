import type { AcceptB2BQuoteResult, B2BContext, B2BQuoteDetail, CreateB2BQuoteRequestInput, CreateB2BQuoteRequestResult } from "@/lib/b2b/types";
import { B2BApiError } from "@/lib/b2b/types";

async function request<T>(path: string, init: RequestInit = {}, mockScenario?: string): Promise<T> {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  const response = await fetch(`/api/b2b${path}${params.size ? `?${params}` : ""}`, { ...init, cache: "no-store", credentials: "include" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new B2BApiError(response.status, payload);
  return payload as T;
}

export const b2bService = {
  context(mockScenario?: string) { return request<B2BContext>("/context", {}, mockScenario); },
  createQuoteRequest(input: CreateB2BQuoteRequestInput, mockScenario?: string) {
    return request<CreateB2BQuoteRequestResult>("/quote-requests", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }, mockScenario);
  },
  quote(quoteId: string, mockScenario?: string) { return request<B2BQuoteDetail>(`/quotes/${encodeURIComponent(quoteId)}`, {}, mockScenario); },
  acceptQuote(quoteId: string, idempotencyKey: string, mockScenario?: string) {
    return request<AcceptB2BQuoteResult>(`/quotes/${encodeURIComponent(quoteId)}/accept`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ idempotencyKey, confirmation: true }) }, mockScenario);
  }
};
