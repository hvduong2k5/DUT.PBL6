import type { AddCartItemInput, Cart, UpdateCartItemInput } from "@/lib/cart/types";
import { CartApiError } from "@/lib/cart/types";

async function requestCart(path: string, init: RequestInit = {}, mockScenario?: string): Promise<Cart> {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  const suffix = params.size ? `?${params}` : "";
  const response = await fetch(`/api/cart${path}${suffix}`, {
    ...init,
    cache: "no-store",
    headers: init.body ? { "Content-Type": "application/json", ...init.headers } : init.headers
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new CartApiError(response.status, payload);
  return payload as Cart;
}

export const cartService = {
  get(mockScenario?: string): Promise<Cart> {
    return requestCart("", {}, mockScenario);
  },
  addItem(input: AddCartItemInput, mockScenario?: string): Promise<Cart> {
    return requestCart("/items", { method: "POST", body: JSON.stringify(input) }, mockScenario);
  },
  updateItem(itemId: string, input: UpdateCartItemInput, mockScenario?: string): Promise<Cart> {
    return requestCart(`/items/${encodeURIComponent(itemId)}`, { method: "PATCH", body: JSON.stringify(input) }, mockScenario);
  },
  removeItem(itemId: string, mockScenario?: string): Promise<Cart> {
    return requestCart(`/items/${encodeURIComponent(itemId)}`, { method: "DELETE" }, mockScenario);
  }
};
