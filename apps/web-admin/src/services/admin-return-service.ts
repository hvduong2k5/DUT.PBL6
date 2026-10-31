import type { ReturnCaseDetail, ReturnCaseList } from "@/lib/returns/types";
import { AdminReturnApiError } from "@/lib/returns/types";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`/api/admin/returns${path}`, { cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminReturnApiError(response.status, payload);
  return payload as T;
}

export const adminReturnService = {
  cases() { return request<ReturnCaseList>("/cases"); },
  case(caseId: string) { return request<ReturnCaseDetail>(`/cases/${encodeURIComponent(caseId)}`); }
};
