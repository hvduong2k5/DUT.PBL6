import type { PaymentCaseDecision, PaymentCaseDetail, PaymentCaseList, PaymentRefundList, PaymentRefundSummary, RefundApprovalItem, RefundCertificateItem, RefundStartInstructionView } from "@/features/employee/payments/types";
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
export interface CreateApprovalInput { receiptId: string; amount: number; beneficiaryAccount: string; beneficiaryBank: string; beneficiaryHolder: string; reason: string }
export interface RefundActionInput { expectedVersion: number; reason: string; idempotencyKey: string; careDecisionReference?: string }
export interface IssueCertificateInput { refundId: string; attemptId: string; decision: string; definitive: boolean; sha256: string; amount: number; beneficiaryAccount: string; bank: string; holder: string; bankIdentity: string; bankReference: string; occurredAt: string }

export const adminPaymentService = {
  cases() { return request<PaymentCaseList>("/cases"); },
  case(caseId: string) { return request<PaymentCaseDetail>(`/cases/${encodeURIComponent(caseId)}`); },
  assign(caseId: string, input: AssignCaseInput) { return request<PaymentCaseDetail>(`/cases/${encodeURIComponent(caseId)}/assign`, { method: "POST", body: JSON.stringify(input) }); },
  resolve(caseId: string, input: ResolveCaseInput) { return request<PaymentCaseDetail>(`/cases/${encodeURIComponent(caseId)}/resolve`, { method: "POST", body: JSON.stringify(input) }); },
  decisions(caseId: string) { return request<{ items: PaymentCaseDecision[] }>(`/cases/${encodeURIComponent(caseId)}/decisions`); },
  createDecision(caseId: string, input: CreateDecisionInput) { return request<PaymentCaseDecision>(`/cases/${encodeURIComponent(caseId)}/decisions`, { method: "POST", body: JSON.stringify(input) }); },
  refunds() { return request<PaymentRefundList>("/refunds"); },
  refund(refundId: string) { return request<PaymentRefundSummary>(`/refunds/${encodeURIComponent(refundId)}`); },
  refundAction(refundId: string, action: string, input: RefundActionInput) { return request<PaymentRefundSummary | RefundStartInstructionView>(`/refunds/${encodeURIComponent(refundId)}/${action}`, { method: "POST", body: JSON.stringify(input) }); },
  attachEvidence(refundId: string, input: { expectedVersion: number; reason: string; attemptId: string; evidenceIds: string[]; idempotencyKey: string }) { return request<PaymentRefundSummary>(`/refunds/${encodeURIComponent(refundId)}/evidence`, { method: "POST", body: JSON.stringify(input) }); },
  verifyRefund(refundId: string, input: { expectedVersion: number; reason: string; attemptId: string; outcome: string; evidenceIds: string[]; idempotencyKey: string }) { return request<PaymentRefundSummary>(`/refunds/${encodeURIComponent(refundId)}/verify`, { method: "POST", body: JSON.stringify(input) }); },
  approvals(receiptId?: string) { return request<{ items: RefundApprovalItem[] }>(receiptId ? `/refund-approvals?receiptId=${encodeURIComponent(receiptId)}` : "/refund-approvals"); },
  createApproval(input: CreateApprovalInput) { return request<RefundApprovalItem>("/refund-approvals", { method: "POST", body: JSON.stringify(input) }); },
  withdrawApproval(approvalId: string, input: { withdrawalReference: string; reason: string }) { return request<RefundApprovalItem>(`/refund-approvals/${encodeURIComponent(approvalId)}/withdraw`, { method: "POST", body: JSON.stringify(input) }); },
  acceptApproval(approvalId: string, idempotencyKey: string) { return request<PaymentRefundSummary>(`/refund-approvals/${encodeURIComponent(approvalId)}/accept`, { method: "POST", body: JSON.stringify({ idempotencyKey }) }); },
  certificates(refundId: string) { return request<{ items: RefundCertificateItem[] }>(`/refund-evidence?refundId=${encodeURIComponent(refundId)}`); },
  issueCertificate(input: IssueCertificateInput) { return request<RefundCertificateItem>("/refund-evidence", { method: "POST", body: JSON.stringify(input) }); }
};
