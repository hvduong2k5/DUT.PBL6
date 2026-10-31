import { ContractDataError, parseVndAmount } from "@/lib/api-contract";
import type {
  AcceptCheckoutRequest,
  CheckoutOperation,
  CreateCheckoutQuoteRequest,
  OrderQuote
} from "@/lib/api-contract";
import type { Cart } from "@/lib/cart/types";
import type { CustomerAddress, CustomerProfile } from "@/lib/customer/types";
import type {
  CheckoutProvince,
  CheckoutRecipient,
  MvpCheckoutOperationView,
  MvpCheckoutPaymentMethod,
  MvpCheckoutPreparation,
  MvpCheckoutQuote,
  ShippingAddress
} from "./types";

const OPERATION_STATUSES = new Set<CheckoutOperation["status"]>([
  "ACCEPTED", "PROCESSING", "WAITING_RETRY", "COMPENSATING", "SUCCEEDED", "FAILED", "MANUAL_REVIEW"
]);
const OPERATION_STAGES = new Set<CheckoutOperation["stage"]>([
  "VALIDATING", "ACQUIRING", "PLACING", "COMPENSATING", "DONE"
]);

function requiredText(value: unknown, field: string): string {
  if (typeof value !== "string" || !value.trim()) throw new ContractDataError(`${field} is required.`);
  return value;
}

function checkedTotal(value: number, field: string): number {
  const amount = parseVndAmount(value);
  if (!Number.isSafeInteger(amount)) throw new ContractDataError(`${field} exceeds the safe integer range.`);
  return amount;
}

function savedAddress(address: CustomerAddress, email: string): MvpCheckoutPreparation["savedAddresses"][number] {
  return {
    addressId: address.id,
    label: address.label,
    recipient: { fullName: address.recipientName, phone: address.recipientPhone, email },
    address: {
      provinceCode: address.province.code,
      provinceName: address.province.name,
      wardCode: address.ward.code,
      wardName: address.ward.name,
      addressLine: address.addressLine
    },
    isDefault: address.isDefault
  };
}

export function mapMvpCheckoutPreparation(
  cart: Cart,
  locations: CheckoutProvince[],
  profile?: CustomerProfile,
  addresses: CustomerAddress[] = [],
  accountDataWarning?: string
): MvpCheckoutPreparation {
  if (!Number.isSafeInteger(cart.revision) || Number(cart.revision) < 0) {
    throw new ContractDataError("Cart revision is required before Checkout.");
  }
  const items = cart.items.map((item) => ({
    itemId: item.itemId,
    productSlug: item.productSlug,
    productName: item.productName,
    imageUrl: item.imageUrl,
    imageAlt: item.imageAlt,
    skuId: item.skuId,
    skuLabel: item.skuLabel,
    weightGrams: item.weightGrams,
    flavor: item.flavor,
    packageType: item.packageType,
    unitPriceVnd: item.unitPriceVnd,
    quantity: item.quantity,
    lineSubtotalVnd: item.lineSubtotalVnd ?? 0,
    priceChanged: false,
    previousUnitPriceVnd: null
  }));
  const subtotalVnd = items.reduce((total, item) => total + item.lineSubtotalVnd, 0);
  if (!Number.isSafeInteger(subtotalVnd)) throw new ContractDataError("Checkout subtotal exceeds the safe integer range.");
  return {
    cartRevision: cart.revision as number,
    items,
    itemCount: items.reduce((total, item) => total + item.quantity, 0),
    subtotalVnd,
    pricingComplete: cart.items.every((item) => item.priceKnown !== false),
    customerMode: profile ? "REGISTERED" : "GUEST",
    ...(profile?.fullName && profile.phone ? {
      defaultRecipient: { fullName: profile.fullName, phone: profile.phone, email: profile.email }
    } : {}),
    savedAddresses: addresses.map((address) => savedAddress(address, profile?.email ?? "")),
    locations,
    ...(accountDataWarning ? { accountDataWarning } : {})
  };
}

export function buildMvpQuoteRequest(input: {
  cartRevision: number;
  paymentMethod: MvpCheckoutPaymentMethod;
  selectedAddressId?: string;
  recipient: CheckoutRecipient;
  address: ShippingAddress;
  voucherCode?: string;
}): CreateCheckoutQuoteRequest {
  return {
    cart_revision: input.cartRevision,
    payment_method: input.paymentMethod,
    ...(input.selectedAddressId
      ? { shipping_address_id: input.selectedAddressId }
      : {
          address: {
            recipient_name: input.recipient.fullName.trim(),
            phone: input.recipient.phone.trim(),
            street: input.address.addressLine.trim(),
            ward_code: input.address.wardCode,
            province_code: input.address.provinceCode
          }
        }),
    ...(input.voucherCode ? { voucher_code: input.voucherCode } : {})
  };
}

export function mapMvpCheckoutQuote(quote: OrderQuote): MvpCheckoutQuote {
  return {
    quoteId: requiredText(quote.quote_id, "Quote.quote_id"),
    cartRevision: checkedTotal(quote.cart_revision, "Quote.cart_revision"),
    paymentMethod: quote.payment_method,
    subtotalVnd: checkedTotal(quote.subtotal, "Quote.subtotal"),
    shippingFeeVnd: checkedTotal(quote.shipping_fee, "Quote.shipping_fee"),
    totalVnd: checkedTotal(quote.final_amount, "Quote.final_amount"),
    expiresAt: requiredText(quote.expires_at, "Quote.expires_at"),
    shippingExpiresAt: requiredText(quote.shipping_expires_at, "Quote.shipping_expires_at"),
    items: quote.items.map((item) => ({
      skuCode: requiredText(item.sku_code, "QuoteItem.sku_code"),
      productName: typeof item.product_name === "string" && item.product_name ? item.product_name : item.sku_code,
      quantity: checkedTotal(item.quantity, "QuoteItem.quantity"),
      ...(item.unit_price !== undefined ? { unitPriceVnd: checkedTotal(item.unit_price, "QuoteItem.unit_price") } : {})
    }))
  };
}

export function buildMvpAcceptRequest(quote: MvpCheckoutQuote): AcceptCheckoutRequest {
  return {
    quote_id: quote.quoteId,
    cart_revision: quote.cartRevision,
    payment_method: quote.paymentMethod
  };
}

export function mapMvpCheckoutOperation(operation: CheckoutOperation): MvpCheckoutOperationView {
  if (!OPERATION_STATUSES.has(operation.status) || !OPERATION_STAGES.has(operation.stage)) {
    throw new ContractDataError("Checkout operation status or stage is invalid.");
  }
  const terminal = operation.status === "SUCCEEDED" || operation.status === "FAILED" || operation.status === "MANUAL_REVIEW";
  return {
    operationId: requiredText(operation.operation_id, "CheckoutOperation.operation_id"),
    status: operation.status,
    stage: operation.stage,
    orderId: operation.order_id,
    orderStatus: operation.order_status,
    errorCode: operation.error_code,
    updatedAt: requiredText(operation.updated_at, "CheckoutOperation.updated_at"),
    terminal,
    succeeded: operation.status === "SUCCEEDED" && Boolean(operation.order_id)
  };
}
