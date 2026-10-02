export function parseAdminAccessPath(path: string[]) {
  if (path.length === 1 && path[0] === "employees") return { kind: "list" as const };
  if (path.length === 2 && path[0] === "employees" && path[1]) return { kind: "detail" as const, employeeId: path[1] };
  if (path.length === 1 && path[0] === "roles") return { kind: "role-list" as const };
  if (path.length === 2 && path[0] === "roles" && path[1]) return { kind: "role-detail" as const, roleCode: path[1] };
  if (path.length === 1 && path[0] === "permissions") return { kind: "permission-list" as const };
  if (path.length === 2 && path[0] === "permissions" && path[1]) return { kind: "permission-detail" as const, permissionCode: path[1] };
  return { kind: "invalid" as const };
}
