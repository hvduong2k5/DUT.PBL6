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
  previousUnitPriceVnd?: number | null;
  priceChanged: boolean;
  quantity: number;
  lineSubtotalVnd?: number | null;
  isAvailable: boolean;
  unavailableReason?: string | null;
}

export interface Cart {
  items: CartItem[];
  itemCount: number;
  subtotalVnd: number;
  hasBlockingIssues: boolean;
  notices: CartNotice[];
  updatedAt: string;
}

export interface AddCartItemInput {
  skuId: string;
  quantity: number;
}

export interface UpdateCartItemInput {
  quantity: number;
}

interface CartErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  errors?: Array<{ field: string; message: string }>;
}

export class CartApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly errors: Array<{ field: string; message: string }>;

  constructor(status: number, body: CartErrorBody) {
    super(body.message || "Không thể cập nhật giỏ hàng lúc này.");
    this.name = "CartApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.requestId = body.requestId;
    this.errors = body.errors ?? [];
  }
}
