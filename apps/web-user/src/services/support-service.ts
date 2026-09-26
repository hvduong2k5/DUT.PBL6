import type { CreateSupportTicketInput, CreateSupportTicketResult, SupportAttachmentMetadata, SupportContext, SupportMessage, SupportTicketDetail } from "@/lib/support/types";
import { SupportApiError } from "@/lib/support/types";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/support${path}`, { ...init, cache: "no-store", credentials: "include" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new SupportApiError(response.status, payload);
  return payload as T;
}

function scenarioQuery(mockScenario?: string) {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return params.size ? `?${params}` : "";
}

export const supportService = {
  context(mockScenario?: string) {
    return request<SupportContext>(`/context${scenarioQuery(mockScenario)}`);
  },
  create(input: CreateSupportTicketInput, mockScenario?: string) {
    return request<CreateSupportTicketResult>(`/tickets${scenarioQuery(mockScenario)}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) });
  },
  detail(ticketId: string, mockScenario?: string) {
    return request<SupportTicketDetail>(`/tickets/${encodeURIComponent(ticketId)}${scenarioQuery(mockScenario)}`);
  },
  message(ticketId: string, message: string, attachments: SupportAttachmentMetadata[], idempotencyKey: string, mockScenario?: string) {
    return request<SupportMessage>(`/tickets/${encodeURIComponent(ticketId)}/messages${scenarioQuery(mockScenario)}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ message, attachments, idempotencyKey }) });
  }
};
