import { ContractDataError, parseVndAmount } from "@/lib/api-contract";
import type { OrderCart, OrderItem } from "@/lib/api-contract";
import type { Cart, CartItem } from "./types";

function requiredText(value: unknown, field: string): string {
  if (typeof value !== "string" || !value.trim()) throw new ContractDataError(`${field} is required.`);
  return value;
}

function productSlug(name: string): string {
  return name
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/gu, "")
    .replace(/đ/giu, "d")
    .toLocaleLowerCase("vi")
    .replace(/[^a-z0-9]+/gu, "-")
    .replace(/^-|-$/gu, "");
}

function mapItem(item: OrderItem): CartItem {
  const skuCode = requiredText(item.sku_code, "CartItem.sku_code");
  const itemId = requiredText(item.item_id, "CartItem.item_id");
  const quantity = parseVndAmount(item.quantity);
  if (quantity < 1) throw new ContractDataError("CartItem.quantity must be positive.");
  const productName = typeof item.product_name === "string" && item.product_name.trim() ? item.product_name : skuCode;
  const priceKnown = item.unit_price !== undefined;
  const unitPriceVnd = priceKnown ? parseVndAmount(item.unit_price) : 0;
  const weightGrams = item.weight_grams === undefined ? 0 : parseVndAmount(item.weight_grams);
  const lineSubtotalVnd = priceKnown ? unitPriceVnd * quantity : null;
  if (lineSubtotalVnd !== null && !Number.isSafeInteger(lineSubtotalVnd)) {
    throw new ContractDataError("CartItem subtotal exceeds the safe integer range.");
  }

  return {
    itemId,
    productId: "",
    productSlug: productSlug(productName),
    productName,
    imageUrl: "",
    imageAlt: productName,
    skuId: skuCode,
    skuLabel: weightGrams ? `${weightGrams}g` : skuCode,
    weightGrams,
    packageType: weightGrams ? `${weightGrams}g` : skuCode,
    unitPriceVnd,
    priceKnown,
    previousUnitPriceVnd: null,
    priceChanged: false,
    quantity,
    lineSubtotalVnd,
    isAvailable: true,
    availabilityUnverified: true,
    unavailableReason: null
  };
}

export function mapMvpCart(cart: OrderCart): Cart {
  if (!Number.isSafeInteger(cart.revision) || cart.revision < 0) {
    throw new ContractDataError("Cart.revision must be a non-negative safe integer.");
  }
  if (!Array.isArray(cart.items)) throw new ContractDataError("Cart.items must be an array.");
  const items = cart.items.map(mapItem);
  const subtotalVnd = items.reduce((total, item) => total + (item.lineSubtotalVnd ?? 0), 0);
  if (!Number.isSafeInteger(subtotalVnd)) throw new ContractDataError("Cart subtotal exceeds the safe integer range.");
  return {
    revision: cart.revision,
    items,
    itemCount: items.reduce((total, item) => total + item.quantity, 0),
    subtotalVnd,
    pricingComplete: items.every((item) => item.priceKnown !== false),
    hasBlockingIssues: false,
    notices: []
  };
}
