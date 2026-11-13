export function parseAdminAccessPath(path: string[]) {
  if (path.length === 1 && path[0] === "employees") return { kind: "list" as const };
  if (path.length === 2 && path[0] === "employees" && path[1]) return { kind: "detail" as const, employeeId: path[1] };
  if (path.length === 1 && path[0] === "roles") return { kind: "role-list" as const };
  if (path.length === 2 && path[0] === "roles" && path[1]) return { kind: "role-detail" as const, roleCode: path[1] };
  if (path.length === 1 && path[0] === "permissions") return { kind: "permission-list" as const };
  if (path.length === 2 && path[0] === "permissions" && path[1]) return { kind: "permission-detail" as const, permissionCode: path[1] };
  return { kind: "invalid" as const };
}

export type AdminAccessWriteRoute =
  | { kind: "create-role" }
  | { kind: "update-role"; roleCode: string }
  | { kind: "delete-role"; roleCode: string }
  | { kind: "replace-role-permissions"; roleCode: string }
  | { kind: "assign-role"; employeeId: string }
  | { kind: "revoke-assignment"; employeeId: string; assignmentId: string }
  | { kind: "reset-password"; employeeId: string }
  | { kind: "revoke-sessions"; employeeId: string }
  | { kind: "create-invitation" }
  | { kind: "revoke-invitation"; invitationId: string }
  | { kind: "invalid" };

/** Write routes for the administration workspaces (F-47). Kept separate from
 * the read parser so the read contract and its tests stay untouched. */
export function parseAdminAccessWritePath(path: string[], method: string): AdminAccessWriteRoute {
  if (method === "POST" && path.length === 1 && path[0] === "roles") return { kind: "create-role" };
  if (method === "PATCH" && path.length === 2 && path[0] === "roles" && path[1]) return { kind: "update-role", roleCode: path[1] };
  if (method === "DELETE" && path.length === 2 && path[0] === "roles" && path[1]) return { kind: "delete-role", roleCode: path[1] };
  if (method === "PUT" && path.length === 3 && path[0] === "roles" && path[1] && path[2] === "permissions") return { kind: "replace-role-permissions", roleCode: path[1] };
  if (method === "POST" && path.length === 3 && path[0] === "employees" && path[1] && path[2] === "assignments") return { kind: "assign-role", employeeId: path[1] };
  if (method === "DELETE" && path.length === 4 && path[0] === "employees" && path[1] && path[2] === "assignments" && path[3]) return { kind: "revoke-assignment", employeeId: path[1], assignmentId: path[3] };
  if (method === "POST" && path.length === 3 && path[0] === "employees" && path[1] && path[2] === "password-reset") return { kind: "reset-password", employeeId: path[1] };
  if (method === "POST" && path.length === 3 && path[0] === "employees" && path[1] && path[2] === "sessions-revoke") return { kind: "revoke-sessions", employeeId: path[1] };
  if (method === "POST" && path.length === 1 && path[0] === "invitations") return { kind: "create-invitation" };
  if (method === "DELETE" && path.length === 2 && path[0] === "invitations" && path[1]) return { kind: "revoke-invitation", invitationId: path[1] };
  return { kind: "invalid" };
}
