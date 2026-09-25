export interface CheckoutLine {
  itemId: string;
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
  quantity: number;
  lineSubtotalVnd: number;
  priceChanged: boolean;
  previousUnitPriceVnd?: number | null;
}

export interface CheckoutRecipient {
  fullName: string;
  phone: string;
  email?: string;
}

export interface ShippingAddress {
  provinceCode: string;
  provinceName: string;
  districtCode: string;
  districtName: string;
  addressLine: string;
}

export interface SavedCheckoutAddress {
  addressId: string;
  label: string;
  recipient: CheckoutRecipient;
  address: ShippingAddress;
  isDefault: boolean;
}

export interface CheckoutPreparation {
  checkoutSessionId: string;
  items: CheckoutLine[];
  itemCount: number;
  subtotalVnd: number;
  priceRevalidatedAt: string;
  requiresPriceAcknowledgement: boolean;
  notices: Array<{ code: "PRICE_CHANGED"; itemId: string; message: string }>;
  customerMode: "GUEST" | "REGISTERED";
  savedAddresses: SavedCheckoutAddress[];
}

export interface ShippingOption {
  shippingOptionId: string;
  name: string;
  description: string;
  feeVnd: number;
  estimatedDelivery: string;
}

export interface ShippingQuote {
  options: ShippingOption[];
  quotedAt: string;
  expiresAt: string;
}

export type CheckoutPaymentMethod = "BANK_TRANSFER" | "COD";

export interface PrepareCheckoutInput {
  itemIds: string[];
}

export interface QuoteShippingInput {
  checkoutSessionId: string;
  itemIds: string[];
  address: ShippingAddress;
}

export interface ConfirmCheckoutInput {
  checkoutSessionId: string;
  itemIds: string[];
  recipient: CheckoutRecipient;
  address: ShippingAddress;
  shippingOptionId: string;
  paymentMethod: CheckoutPaymentMethod;
  idempotencyKey: string;
  priceRevalidatedAt: string;
  priceChangesAcknowledged: boolean;
}

export interface CheckoutConfirmation {
  orderId: string;
  orderNumber: string;
  status: "PENDING_PAYMENT" | "PLACED";
  paymentStatus: "UNPAID";
  paymentMethod: CheckoutPaymentMethod;
  reservationExpiresAt: string;
  subtotalVnd: number;
  shippingFeeVnd: number;
  discountVnd: number;
  totalVnd: number;
  nextStep: "PAYMENT_REQUIRED" | "ORDER_PLACED";
  message: string;
}

interface CheckoutErrorBody {
  code?: string;
  message?: string;
  requestId?: string;
  errors?: Array<{ field: string; message: string }>;
  itemIssues?: Array<{ itemId: string; code: string; message: string }>;
}

export class CheckoutApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly errors: Array<{ field: string; message: string }>;
  readonly itemIssues: Array<{ itemId: string; code: string; message: string }>;

  constructor(status: number, body: CheckoutErrorBody) {
    super(body.message || "Không thể tiếp tục thanh toán lúc này.");
    this.name = "CheckoutApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.requestId = body.requestId;
    this.errors = body.errors ?? [];
    this.itemIssues = body.itemIssues ?? [];
  }
}
