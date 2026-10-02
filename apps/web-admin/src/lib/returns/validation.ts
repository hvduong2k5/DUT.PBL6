export function parseAdminReturnPath(path: string[]) {
  if (path.length === 1 && path[0] === "cases") return { kind: "case-list" as const };
  if (path.length === 2 && path[0] === "cases" && path[1]) return { kind: "case-detail" as const, caseId: path[1] };
  return { kind: "invalid" as const };
}
