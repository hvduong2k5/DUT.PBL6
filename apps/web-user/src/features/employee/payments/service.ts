import type { PaymentCaseDecision, PaymentCaseDetail, PaymentCaseList } from "@/features/employee/payments/types";
import { AdminPaymentApiError } from "@/features/employee/payments/types";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/admin/payments${path}`, {
    cache: "no-store",
    credentials: "include",
    ...init,
    headers: { ...(init?.body ? { "Content-Type": "application/json" } : {}), ...(init?.headers ?? {}) }
  });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminPaymentApiError(response.status, payload);
  return payload as T;
}

export interface AssignCaseInput { expectedVersion: number; reason: string; assigneeId: string; idempotencyKey: string }
export interface ResolveCaseInput { expectedVersion: number; reason: string; resolution: string; decisionReference: string; evidenceIds: string[]; idempotencyKey: string }
export interface CreateDecisionInput { resolution: string; reason: string; evidenceIds: string[] }

export const adminPaymentService = {
  cases() { return request<PaymentCaseList>("/cases"); },
  case(caseId: string) { return request<PaymentCaseDetail>(`/cases/${encodeURIComponent(caseId)}`); },
  assign(caseId: string, input: AssignCaseInput) { return request<PaymentCaseDetail>(`/cases/${encodeURIComponent(caseId)}/assign`, { method: "POST", body: JSON.stringify(input) }); },
  resolve(caseId: string, input: ResolveCaseInput) { return request<PaymentCaseDetail>(`/cases/${encodeURIComponent(caseId)}/resolve`, { method: "POST", body: JSON.stringify(input) }); },
  decisions(caseId: string) { return request<{ items: PaymentCaseDecision[] }>(`/cases/${encodeURIComponent(caseId)}/decisions`); },
  createDecision(caseId: string, input: CreateDecisionInput) { return request<PaymentCaseDecision>(`/cases/${encodeURIComponent(caseId)}/decisions`, { method: "POST", body: JSON.stringify(input) }); }
};
