export function parseAdminOrderPath(path: string[]) {
  if (path.length === 0) return { kind: "order-list" as const };
  if (path.length === 1 && path[0]) return { kind: "order-detail" as const, orderId: path[0] };
  return { kind: "invalid" as const };
}
