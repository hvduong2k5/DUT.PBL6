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
  wardCode: string;
  wardName: string;
  addressLine: string;
}

export interface CheckoutWard {
  wardCode: string;
  wardName: string;
}

export interface CheckoutProvince {
  provinceCode: string;
  provinceName: string;
  wards: CheckoutWard[];
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
  defaultRecipient?: CheckoutRecipient;
  savedAddresses: SavedCheckoutAddress[];
  locations: CheckoutProvince[];
  accountDataWarning?: string;
}

export interface ShippingOption {
  shippingOptionId: string;
  name: string;
  description: string;
  feeVnd: number;
  estimatedDelivery: string;
  calculatedAtCheckout?: boolean;
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
  voucherCode?: string;
}

export interface CheckoutConfirmation {
  customerMode: "GUEST" | "REGISTERED";
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
  paymentPath: string;
  message: string;
  vietQrUrl?: string;
  paymentExpiresAt?: string;
}

export type MvpCheckoutPaymentMethod = "VIETQR" | "COD";

export interface MvpCheckoutPreparation {
  cartRevision: number;
  items: CheckoutLine[];
  itemCount: number;
  subtotalVnd: number;
  pricingComplete: boolean;
  customerMode: "GUEST" | "REGISTERED";
  defaultRecipient?: CheckoutRecipient;
  savedAddresses: SavedCheckoutAddress[];
  locations: CheckoutProvince[];
  accountDataWarning?: string;
}

export interface MvpCheckoutQuote {
  quoteId: string;
  cartRevision: number;
  paymentMethod: MvpCheckoutPaymentMethod;
  subtotalVnd: number;
  shippingFeeVnd: number;
  totalVnd: number;
  expiresAt: string;
  shippingExpiresAt: string;
  items: Array<{ skuCode: string; productName: string; quantity: number; unitPriceVnd?: number }>;
}

export interface MvpCheckoutOperationView {
  operationId: string;
  status: "ACCEPTED" | "PROCESSING" | "WAITING_RETRY" | "COMPENSATING" | "SUCCEEDED" | "FAILED" | "MANUAL_REVIEW";
  stage: "VALIDATING" | "ACQUIRING" | "PLACING" | "COMPENSATING" | "DONE";
  orderId: string | null;
  orderStatus?: string;
  errorCode?: string;
  updatedAt: string;
  terminal: boolean;
  succeeded: boolean;
}
