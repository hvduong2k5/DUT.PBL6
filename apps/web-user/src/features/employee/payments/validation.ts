export function parseAdminPaymentPath(path: string[]) {
  if (path.length === 1 && path[0] === "cases") return { kind: "case-list" as const };
  if (path.length === 2 && path[0] === "cases" && path[1]) return { kind: "case-detail" as const, caseId: path[1] };
  if (path.length === 3 && path[0] === "cases" && path[1] && path[2] === "assign") return { kind: "case-assign" as const, caseId: path[1] };
  if (path.length === 3 && path[0] === "cases" && path[1] && path[2] === "resolve") return { kind: "case-resolve" as const, caseId: path[1] };
  if (path.length === 3 && path[0] === "cases" && path[1] && path[2] === "decisions") return { kind: "case-decisions" as const, caseId: path[1] };
  if (path.length === 1 && path[0] === "refunds") return { kind: "refund-list" as const };
  if (path.length === 2 && path[0] === "refunds" && path[1]) return { kind: "refund-detail" as const, refundId: path[1] };
  if (path.length === 3 && path[0] === "refunds" && path[1] && path[2] === "evidence") return { kind: "refund-evidence" as const, refundId: path[1] };
  if (path.length === 3 && path[0] === "refunds" && path[1] && path[2] === "verify") return { kind: "refund-verify" as const, refundId: path[1] };
  if (path.length === 3 && path[0] === "refunds" && path[1] && ["claim", "start", "report-uncertain", "retry-attempt", "cancel-obligation"].includes(path[2])) return { kind: "refund-action" as const, refundId: path[1], action: path[2] };
  if (path.length === 1 && path[0] === "refund-approvals") return { kind: "refund-approvals" as const };
  if (path.length === 3 && path[0] === "refund-approvals" && path[1] && path[2] === "withdraw") return { kind: "refund-approval-withdraw" as const, approvalId: path[1] };
  if (path.length === 3 && path[0] === "refund-approvals" && path[1] && path[2] === "accept") return { kind: "refund-approval-accept" as const, approvalId: path[1] };
  if (path.length === 1 && path[0] === "refund-evidence") return { kind: "refund-evidence-records" as const };
  return { kind: "invalid" as const };
}
