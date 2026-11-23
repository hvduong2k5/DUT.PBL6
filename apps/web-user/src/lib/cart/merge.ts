import type { OrderCart } from "@/lib/api-contract";

export interface CartMergeOutcome { sku_code: string; status: "merged" | "adjusted" | "dropped"; requested_quantity: number; quantity: number; reason?: string }
export interface CartMergeResult { merge_key: string; cart: OrderCart; items: CartMergeOutcome[]; merged_at: string }

export function isCartMergeResult(value: unknown): value is CartMergeResult {
  if (!value || typeof value !== "object") return false;
  const body = value as CartMergeResult;
  return typeof body.merge_key === "string" && body.merge_key.length >= 16 && body.merge_key.length <= 128
    && typeof body.merged_at === "string" && Number.isFinite(Date.parse(body.merged_at))
    && Boolean(body.cart) && Number.isSafeInteger(body.cart.revision) && body.cart.revision >= 0 && Array.isArray(body.cart.items)
    && body.cart.items.length <= 100 && body.cart.items.every((item) => typeof item.sku_code === "string" && Number.isSafeInteger(item.quantity) && item.quantity >= 1 && item.quantity <= 999)
    && Array.isArray(body.items) && body.items.every((item) => typeof item.sku_code === "string" && ["merged", "adjusted", "dropped"].includes(item.status) && Number.isSafeInteger(item.quantity) && item.quantity >= 0 && Number.isSafeInteger(item.requested_quantity) && item.requested_quantity >= 1 && (item.reason === undefined || typeof item.reason === "string"));
}
