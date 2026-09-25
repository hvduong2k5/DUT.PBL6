import type { ProductDetail } from "@/lib/product-detail/types";
import { ProductDetailApiError } from "@/lib/product-detail/types";

export const productDetailService = {
  async getBySlug(slug: string, mockScenario?: string): Promise<ProductDetail> {
    const params = new URLSearchParams();
    if (mockScenario) params.set("mockScenario", mockScenario);
    const suffix = params.size ? `?${params}` : "";
    const response = await fetch(`/api/catalog/products/${encodeURIComponent(slug)}${suffix}`, { cache: "no-store" });
    const contentType = response.headers.get("content-type") ?? "";
    const payload = contentType.includes("application/json") ? await response.json() : {};
    if (!response.ok) throw new ProductDetailApiError(response.status, payload);
    return payload as ProductDetail;
  }
};
