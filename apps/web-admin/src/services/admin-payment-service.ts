import type { PaymentCaseDetail, PaymentCaseList } from "@/lib/payments/types";
import { AdminPaymentApiError } from "@/lib/payments/types";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`/api/admin/payments${path}`, { cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminPaymentApiError(response.status, payload);
  return payload as T;
}

export const adminPaymentService = {
  cases() { return request<PaymentCaseList>("/cases"); },
  case(caseId: string) { return request<PaymentCaseDetail>(`/cases/${encodeURIComponent(caseId)}`); }
};
