export function parseAdminCatalogPath(path: string[]) {
  if (path.length === 1 && path[0] === "products") return { kind: "product-list" as const };
  if (path.length === 2 && path[0] === "products" && path[1]) return { kind: "product-detail" as const, productId: path[1] };
  return { kind: "invalid" as const };
}
