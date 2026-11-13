import type { SupportTicketDetail, SupportTicketList } from "@/features/employee/support/types";
import { AdminSupportApiError } from "@/features/employee/support/types";
async function request<T>(path: string): Promise<T> { const response = await fetch(`/api/admin/support${path}`, { cache: "no-store", credentials: "include" }); const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {}; if (!response.ok) throw new AdminSupportApiError(response.status, payload); return payload as T; }
export const adminSupportService = { tickets() { return request<SupportTicketList>("/tickets"); }, ticket(ticketId: string) { return request<SupportTicketDetail>(`/tickets/${encodeURIComponent(ticketId)}`); } };
