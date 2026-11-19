import { requestPublicJson } from "@/lib/api-contract";
import type { CatalogProductsResponse, CatalogSearchResponse } from "@/lib/api-contract";
import { mapCatalogProducts, mapCatalogSearch } from "@/lib/catalog/mvp";
import type { CatalogResult, DiscoveryConfig } from "@/lib/catalog/types";

async function request<T>(path: string): Promise<T> {
  return (await requestPublicJson<T>(`/api/catalog/${path}`)).data;
}

export const catalogService = {
  async getConfig(mockScenario?: string) {
    const params = new URLSearchParams();
    if (mockScenario) params.set("mockScenario", mockScenario);
    const suffix = params.size ? `?${params}` : "";
    return request<DiscoveryConfig>(`discovery-config${suffix}`);
  },

  async getProducts(params: URLSearchParams): Promise<CatalogResult> {
    const page = Number(params.get("page") ?? 1);
    const pageSize = Number(params.get("pageSize") ?? 9);
    if (params.get("q")) {
      return mapCatalogSearch(await request<CatalogSearchResponse>(`products?${params}`), page, pageSize);
    }
    return mapCatalogProducts(await request<CatalogProductsResponse>(`products?${params}`));
  }
};
