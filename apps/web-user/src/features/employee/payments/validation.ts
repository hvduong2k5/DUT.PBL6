export function parseAdminPaymentPath(path: string[]) {
  if (path.length === 1 && path[0] === "cases") return { kind: "case-list" as const };
  if (path.length === 2 && path[0] === "cases" && path[1]) return { kind: "case-detail" as const, caseId: path[1] };
  if (path.length === 3 && path[0] === "cases" && path[1] && path[2] === "assign") return { kind: "case-assign" as const, caseId: path[1] };
  if (path.length === 3 && path[0] === "cases" && path[1] && path[2] === "resolve") return { kind: "case-resolve" as const, caseId: path[1] };
  if (path.length === 3 && path[0] === "cases" && path[1] && path[2] === "decisions") return { kind: "case-decisions" as const, caseId: path[1] };
  return { kind: "invalid" as const };
}
