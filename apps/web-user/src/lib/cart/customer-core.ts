import type { Cart, CartItem } from "./types";

interface Money {
  currency_code: string;
  units: number;
  nanos: number;
}

export interface CustomerCoreCartItem {
  item_id: string;
  sku_code: string;
  product_name: string;
  variant_name: string;
  thumbnail_url: string;
  unit_price: Money;
  quantity: number;
  subtotal: Money;
  in_stock: boolean;
}

export interface CustomerCoreCart {
  cart_id: string;
  items: CustomerCoreCartItem[];
  total_items: number;
  subtotal_amount: Money;
}

function productSlug(name: string) {
  return name
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/gu, "")
    .replace(/đ/giu, "d")
    .toLocaleLowerCase("vi")
    .replace(/[^a-z0-9]+/gu, "-")
    .replace(/^-|-$/gu, "");
}

function weightFromVariant(name: string) {
  const match = name.match(/(\d+)\s*g\b/iu);
  return match ? Number(match[1]) : 0;
}

function mapItem(item: CustomerCoreCartItem): CartItem {
  const lineSubtotalVnd = item.unit_price.units * item.quantity;
  return {
    itemId: item.item_id,
    productId: "",
    productSlug: productSlug(item.product_name),
    productName: item.product_name,
    imageUrl: item.thumbnail_url,
    imageAlt: item.product_name,
    skuId: item.sku_code,
    skuLabel: item.variant_name,
    weightGrams: weightFromVariant(item.variant_name),
    packageType: item.variant_name,
    unitPriceVnd: item.unit_price.units,
    previousUnitPriceVnd: null,
    priceChanged: false,
    quantity: item.quantity,
    lineSubtotalVnd: item.in_stock ? lineSubtotalVnd : null,
    isAvailable: item.in_stock,
    unavailableReason: item.in_stock ? null : "Sản phẩm hiện không còn hàng."
  };
}

export function mapCustomerCoreCart(cart: CustomerCoreCart): Cart {
  const items = cart.items.map(mapItem);
  return {
    items,
    itemCount: items.reduce((total, item) => total + item.quantity, 0),
    subtotalVnd: items.reduce((total, item) => total + (item.lineSubtotalVnd ?? 0), 0),
    hasBlockingIssues: items.some((item) => !item.isAvailable),
    notices: items.filter((item) => !item.isAvailable).map((item) => ({
      code: "SKU_UNAVAILABLE" as const,
      itemId: item.itemId,
      message: item.unavailableReason ?? "Sản phẩm hiện không còn hàng."
    })),
    updatedAt: new Date().toISOString()
  };
}

export function withCartItemQuantity(cart: Cart, itemId: string, quantity: number): Cart {
  const items = cart.items.map((item) => item.itemId === itemId ? {
    ...item,
    quantity,
    lineSubtotalVnd: item.isAvailable ? item.unitPriceVnd * quantity : null
  } : item);

  return {
    ...cart,
    items,
    itemCount: items.reduce((total, item) => total + item.quantity, 0),
    subtotalVnd: items.reduce((total, item) => total + (item.lineSubtotalVnd ?? 0), 0),
    updatedAt: new Date().toISOString()
  };
}

export function emptyCart(): Cart {
  return { items: [], itemCount: 0, subtotalVnd: 0, hasBlockingIssues: false, notices: [], updatedAt: new Date().toISOString() };
}
