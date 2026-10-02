import type { AdminProductDetail, AdminProductList } from "@/lib/catalog/types";
import { AdminCatalogApiError } from "@/lib/catalog/types";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`/api/admin/catalog${path}`, { cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminCatalogApiError(response.status, payload);
  return payload as T;
}

export const adminCatalogService = {
  products() { return request<AdminProductList>("/products"); },
  product(productId: string) { return request<AdminProductDetail>(`/products/${encodeURIComponent(productId)}`); }
};
