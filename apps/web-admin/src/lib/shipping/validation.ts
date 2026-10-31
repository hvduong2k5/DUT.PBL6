export function parseAdminShippingPath(path: string[]) {
  if (path.length === 1 && path[0] === "shipments") return { kind: "shipment-list" as const };
  if (path.length === 2 && path[0] === "shipments" && path[1]) return { kind: "shipment-detail" as const, shipmentId: path[1] };
  if (path.length === 2 && path[0] === "workbench" && path[1] === "shipments") return { kind: "workbench-list" as const };
  if (path.length === 3 && path[0] === "workbench" && path[1] === "shipments" && path[2]) return { kind: "workbench-detail" as const, shipmentId: path[2] };
  return { kind: "invalid" as const };
}
