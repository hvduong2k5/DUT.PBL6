import type { CreateReturnCaseInput, CreateReturnCaseResult, ReturnCaseDetail, ReturnCaseMessage, ReturnEligibility } from "@/lib/returns/types";
import { ReturnApiError } from "@/lib/returns/types";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/returns${path}`, { ...init, cache: "no-store", credentials: "include" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new ReturnApiError(response.status, payload);
  return payload as T;
}

function scenarioQuery(mockScenario?: string) {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return params.size ? `?${params}` : "";
}

export const returnService = {
  eligibility(orderId: string, mockScenario?: string) {
    return request<ReturnEligibility>(`/orders/${encodeURIComponent(orderId)}/eligibility${scenarioQuery(mockScenario)}`);
  },
  create(input: CreateReturnCaseInput, mockScenario?: string) {
    return request<CreateReturnCaseResult>(scenarioQuery(mockScenario), { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) });
  },
  detail(caseId: string, mockScenario?: string) {
    return request<ReturnCaseDetail>(`/${encodeURIComponent(caseId)}${scenarioQuery(mockScenario)}`);
  },
  supplement(caseId: string, message: string, idempotencyKey: string, mockScenario?: string) {
    return request<ReturnCaseMessage>(`/${encodeURIComponent(caseId)}/supplements${scenarioQuery(mockScenario)}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ message, idempotencyKey }) });
  }
};
