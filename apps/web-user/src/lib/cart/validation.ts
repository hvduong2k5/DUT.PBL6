import type { AddCartItemInput, UpdateCartItemInput } from "./types";

export const MAX_CART_ITEM_QUANTITY = 20;

const SKU_ID_PATTERN = /^[A-Z0-9][A-Z0-9-]{1,63}$/u;
const ITEM_ID_PATTERN = /^[A-Za-z0-9-]{1,80}$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;

export type CartOperation = "get-cart" | "add-item" | "update-item" | "remove-item";

export interface CartRouteValidation {
  operation?: CartOperation;
  itemId?: string;
  mockScenario?: string;
  error?: { field: string; message: string };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasOnlyKeys(value: Record<string, unknown>, keys: string[]): boolean {
  return Object.keys(value).every((key) => keys.includes(key));
}

export function parseCartRoute(method: string, path: string[], params: URLSearchParams): CartRouteValidation {
  const unexpectedQuery = [...params.keys()].find((key) => key !== "mockScenario");
  if (unexpectedQuery) {
    return { error: { field: unexpectedQuery, message: "Query không được hỗ trợ cho giỏ hàng." } };
  }

  const mockScenario = params.get("mockScenario") || undefined;
  if (mockScenario && !SCENARIO_PATTERN.test(mockScenario)) {
    return { error: { field: "mockScenario", message: "Mock scenario không hợp lệ." } };
  }

  if (method === "GET" && path.length === 0) return { operation: "get-cart", mockScenario };
  if (method === "POST" && path.length === 1 && path[0] === "items") return { operation: "add-item", mockScenario };

  if ((method === "PATCH" || method === "DELETE") && path.length === 2 && path[0] === "items") {
    if (!ITEM_ID_PATTERN.test(path[1])) {
      return { error: { field: "itemId", message: "Mã dòng giỏ hàng không hợp lệ." } };
    }
    return {
      operation: method === "PATCH" ? "update-item" : "remove-item",
      itemId: path[1],
      mockScenario
    };
  }

  return { error: { field: "path", message: "Cart route is not available." } };
}

export function validateAddCartItemInput(value: unknown): { data?: AddCartItemInput; errors: Array<{ field: string; message: string }> } {
  if (!isRecord(value) || !hasOnlyKeys(value, ["skuId", "quantity"])) {
    return { errors: [{ field: "body", message: "Dữ liệu thêm giỏ không hợp lệ." }] };
  }
  const errors: Array<{ field: string; message: string }> = [];
  if (typeof value.skuId !== "string" || !SKU_ID_PATTERN.test(value.skuId)) {
    errors.push({ field: "skuId", message: "SKU không hợp lệ." });
  }
  if (!Number.isInteger(value.quantity) || Number(value.quantity) < 1 || Number(value.quantity) > MAX_CART_ITEM_QUANTITY) {
    errors.push({ field: "quantity", message: `Số lượng phải là số nguyên từ 1 đến ${MAX_CART_ITEM_QUANTITY}.` });
  }
  return errors.length ? { errors } : { data: { skuId: value.skuId as string, quantity: value.quantity as number }, errors };
}

export function validateUpdateCartItemInput(value: unknown): { data?: UpdateCartItemInput; errors: Array<{ field: string; message: string }> } {
  if (!isRecord(value) || !hasOnlyKeys(value, ["quantity"])) {
    return { errors: [{ field: "body", message: "Dữ liệu cập nhật số lượng không hợp lệ." }] };
  }
  if (!Number.isInteger(value.quantity) || Number(value.quantity) < 1 || Number(value.quantity) > MAX_CART_ITEM_QUANTITY) {
    return { errors: [{ field: "quantity", message: `Số lượng phải là số nguyên từ 1 đến ${MAX_CART_ITEM_QUANTITY}.` }] };
  }
  return { data: { quantity: value.quantity as number }, errors: [] };
}
