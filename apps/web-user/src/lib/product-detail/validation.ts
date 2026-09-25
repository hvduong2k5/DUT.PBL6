const SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;

export interface ProductDetailRequestValidation {
  slug?: string;
  mockScenario?: string;
  error?: { field: string; message: string };
}

export function parseProductDetailRequest(path: string[], params: URLSearchParams): ProductDetailRequestValidation {
  if (path.length !== 2 || path[0] !== "products") {
    return { error: { field: "path", message: "Catalog route is not available." } };
  }

  const slug = path[1];
  if (slug.length > 120 || !SLUG_PATTERN.test(slug)) {
    return { error: { field: "slug", message: "Slug sản phẩm không hợp lệ." } };
  }

  const unexpectedQuery = [...params.keys()].find((key) => key !== "mockScenario");
  if (unexpectedQuery) {
    return { error: { field: unexpectedQuery, message: "Query không được hỗ trợ cho chi tiết sản phẩm." } };
  }

  const mockScenario = params.get("mockScenario") || undefined;
  if (mockScenario && !SCENARIO_PATTERN.test(mockScenario)) {
    return { error: { field: "mockScenario", message: "Mock scenario không hợp lệ." } };
  }

  return { slug, mockScenario };
}

export function toProductDetailUpstreamPath(slug: string): string {
  return `catalog/products/${encodeURIComponent(slug)}`;
}
