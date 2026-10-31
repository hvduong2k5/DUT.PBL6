export function parseAdminSupportPath(path: string[]) {
  if (path.length === 1 && path[0] === "tickets") return { kind: "ticket-list" as const };
  if (path.length === 2 && path[0] === "tickets" && path[1]) return { kind: "ticket-detail" as const, ticketId: path[1] };
  return { kind: "invalid" as const };
}
