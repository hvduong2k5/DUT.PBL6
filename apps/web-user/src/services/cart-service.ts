import { requestPublicJson } from "@/lib/api-contract";
import type { AddCartItemRequest, OrderCart, UpdateCartItemRequest } from "@/lib/api-contract";
import { mapMvpCart } from "@/lib/cart/mvp";
import type { AddCartItemInput, Cart, UpdateCartItemInput } from "@/lib/cart/types";

function pathWithScenario(path: string, mockScenario?: string): string {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return `/api/cart${path}${params.size ? `?${params}` : ""}`;
}

async function getCart(mockScenario?: string): Promise<Cart> {
  const response = await requestPublicJson<OrderCart>(pathWithScenario("", mockScenario), { credentials: "include" });
  return mapMvpCart(response.data);
}

async function removeCartItem(itemId: string, mockScenario?: string): Promise<Cart> {
  const response = await requestPublicJson<OrderCart>(pathWithScenario(`/items/${encodeURIComponent(itemId)}`, mockScenario), {
    method: "DELETE",
    credentials: "include"
  });
  return mapMvpCart(response.data);
}

export const cartService = {
  get: getCart,

  async addItem(input: AddCartItemInput, mockScenario?: string): Promise<Cart> {
    const current = await getCart(mockScenario);
    const body: AddCartItemRequest = {
      sku_code: input.skuId,
      quantity: input.quantity,
      revision: current.revision ?? 0
    };
    const response = await requestPublicJson<OrderCart>(pathWithScenario("/items", mockScenario), {
      method: "POST",
      credentials: "include",
      body
    });
    return mapMvpCart(response.data);
  },

  async updateItem(itemId: string, input: UpdateCartItemInput, revision: number, mockScenario?: string): Promise<Cart> {
    const body: UpdateCartItemRequest = { quantity: input.quantity, revision };
    const response = await requestPublicJson<OrderCart>(pathWithScenario(`/items/${encodeURIComponent(itemId)}`, mockScenario), {
      method: "PUT",
      credentials: "include",
      body
    });
    return mapMvpCart(response.data);
  },

  removeItem: removeCartItem,

  async clear(mockScenario?: string): Promise<Cart> {
    let current = await getCart(mockScenario);
    for (const item of current.items) current = await removeCartItem(item.itemId, mockScenario);
    return process.env.NODE_ENV === "production" ? current : {
      ...current,
      items: [],
      itemCount: 0,
      subtotalVnd: 0,
      pricingComplete: true,
      hasBlockingIssues: false,
      notices: []
    };
  }
};
