export type AdminCatalogRoute =
  | { kind: "product-list" }
  | { kind: "product-detail"; productId: string }
  | { kind: "product-subresource"; productId: string; segments: string[] }
  | { kind: "invalid" };

const SUBRESOURCE_PATTERNS: string[][] = [
  ["transition"],
  ["skus"],
  ["skus", ":skuId"],
  ["skus", ":skuId", "prices"],
  ["images"],
  ["images", ":imageId"]
];

function matchesSubresource(segments: string[]): boolean {
  return SUBRESOURCE_PATTERNS.some(
    (pattern) => pattern.length === segments.length && pattern.every((part, index) => part.startsWith(":") ? Boolean(segments[index]) : part === segments[index])
  );
}

export function parseAdminCatalogPath(path: string[]): AdminCatalogRoute {
  if (path.length === 1 && path[0] === "products") return { kind: "product-list" };
  if (path.length >= 2 && path[0] === "products" && path[1]) {
    if (path.length === 2) return { kind: "product-detail", productId: path[1] };
    const segments = path.slice(2);
    if (matchesSubresource(segments)) return { kind: "product-subresource", productId: path[1], segments };
  }
  return { kind: "invalid" };
}
