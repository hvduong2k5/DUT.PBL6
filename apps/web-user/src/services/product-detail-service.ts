import { requestPublicJson } from "@/lib/api-contract";
import type { CatalogProductDetail } from "@/lib/api-contract";
import { mapCatalogProductDetail } from "@/lib/product-detail/mvp";
import type { ProductDetail } from "@/lib/product-detail/types";

export const productDetailService = {
  async getBySlug(slug: string, mockScenario?: string): Promise<ProductDetail> {
    const params = new URLSearchParams();
    if (mockScenario) params.set("mockScenario", mockScenario);
    const suffix = params.size ? `?${params}` : "";
    const response = await requestPublicJson<CatalogProductDetail>(`/api/catalog/products/${encodeURIComponent(slug)}${suffix}`);
    return mapCatalogProductDetail(response.data);
  }
};
