import type { CatalogResult, DiscoveryConfig, ProductSummary } from "@/lib/catalog/types";
import { CatalogApiError } from "@/lib/catalog/types";

async function request<T>(path: string): Promise<{ payload: T; headers: Headers }> {
  const response = await fetch(`/api/catalog/${path}`, { cache: "no-store" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new CatalogApiError(response.status, payload);
  return { payload: payload as T, headers: response.headers };
}

export const catalogService = {
  async getConfig(mockScenario?: string) {
    const params = new URLSearchParams();
    if (mockScenario) params.set("mockScenario", mockScenario);
    const suffix = params.size ? `?${params}` : "";
    return (await request<DiscoveryConfig>(`discovery-config${suffix}`)).payload;
  },

  async getProducts(params: URLSearchParams): Promise<CatalogResult> {
    const { payload, headers } = await request<ProductSummary[]>(`products?${params}`);
    return {
      items: payload,
      filteredCount: Number(headers.get("x-filtered-count") ?? payload.length),
      totalCount: Number(headers.get("x-total-count") ?? payload.length),
      page: Number(params.get("page") ?? 1),
      pageSize: Number(params.get("pageSize") ?? 9)
    };
  }
};
