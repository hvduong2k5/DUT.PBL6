import type { AdminB2BActionInput, AdminB2BActionResult, AdminB2BFileHistory, AdminB2BList, AdminB2BQuoteVersionHistory, AdminB2BRequestDetail } from "@/lib/b2b/types";
import { AdminB2BApiError } from "@/lib/b2b/types";

async function request<T>(path: string, init: RequestInit = {}, mockScenario?: string): Promise<T> {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  const response = await fetch(`/api/admin/b2b${path}${params.size ? `${path.includes("?") ? "&" : "?"}${params}` : ""}`, { ...init, cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminB2BApiError(response.status, payload);
  return payload as T;
}

export const adminB2BService = {
  list(mockScenario?: string) { return request<AdminB2BList>("/quote-requests", {}, mockScenario); },
  detail(requestId: string, mockScenario?: string) { return request<AdminB2BRequestDetail>(`/quote-requests/${encodeURIComponent(requestId)}`, {}, mockScenario); },
  files(requestId: string, mockScenario?: string) { return request<AdminB2BFileHistory>(`/quote-requests/${encodeURIComponent(requestId)}/files`, {}, mockScenario); },
  versions(requestId: string, currentVersion: number, currentStatus: string, mockScenario?: string) { return request<AdminB2BQuoteVersionHistory>(`/quote-requests/${encodeURIComponent(requestId)}/versions?currentVersion=${currentVersion}&currentStatus=${encodeURIComponent(currentStatus)}`, {}, mockScenario); },
  act(requestId: string, input: AdminB2BActionInput, mockScenario?: string) { return request<AdminB2BActionResult>(`/quote-requests/${encodeURIComponent(requestId)}/actions`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }, mockScenario); }
};
