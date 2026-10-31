export interface CartNotice {
  code: "PRICE_CHANGED" | "SKU_UNAVAILABLE";
  itemId: string;
  message: string;
}

export interface CartItem {
  itemId: string;
  productId: string;
  productSlug: string;
  productName: string;
  imageUrl: string;
  imageAlt: string;
  skuId: string;
  skuLabel: string;
  weightGrams: number;
  flavor?: string | null;
  packageType: string;
  unitPriceVnd: number;
  priceKnown?: boolean;
  previousUnitPriceVnd?: number | null;
  priceChanged: boolean;
  quantity: number;
  lineSubtotalVnd?: number | null;
  isAvailable: boolean;
  availabilityUnverified?: boolean;
  unavailableReason?: string | null;
}

export interface Cart {
  revision?: number;
  items: CartItem[];
  itemCount: number;
  subtotalVnd: number;
  pricingComplete?: boolean;
  hasBlockingIssues: boolean;
  notices: CartNotice[];
  updatedAt?: string;
}

export interface AddCartItemInput {
  skuId: string;
  quantity: number;
}

export interface UpdateCartItemInput {
  quantity: number;
}
