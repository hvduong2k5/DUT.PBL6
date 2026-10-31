import type {
  AcceptCheckoutRequest,
  CreateCheckoutQuoteRequest,
  OrderAddress,
  OrderPaymentMethod
} from "@/lib/api-contract";
import type {
  ConfirmCheckoutInput,
  CheckoutPaymentMethod,
  CheckoutRecipient,
  PrepareCheckoutInput,
  QuoteShippingInput,
  ShippingAddress
} from "./types";

const ITEM_ID_PATTERN = /^[A-Za-z0-9-]{1,80}$/u;
const SESSION_ID_PATTERN = /^checkout-[0-9a-f-]{36}$/u;
const OPTION_ID_PATTERN = /^[A-Z0-9][A-Z0-9-]{1,63}$/u;
const IDEMPOTENCY_KEY_PATTERN = /^[A-Za-z0-9-]{16,80}$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const PHONE_PATTERN = /^(?:\+84|0)(?:\d[ .-]?){8,10}\d$/u;
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/u;
const PAYMENT_METHODS = new Set<CheckoutPaymentMethod>(["BANK_TRANSFER", "COD"]);
const VOUCHER_PATTERN = /^[A-Z0-9_-]{3,50}$/u;
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu;

export type CheckoutOperation = "prepare" | "quote-shipping" | "confirm";

export interface CheckoutRouteValidation {
  operation?: CheckoutOperation;
  mockScenario?: string;
  cartScenario?: string;
  error?: { field: string; message: string };
}

export type MvpCheckoutOperation = "create-quote" | "accept" | "get-operation";

export interface MvpCheckoutRouteValidation {
  operation?: MvpCheckoutOperation;
  operationId?: string;
  mockScenario?: string;
  error?: { field: string; message: string };
}

type FieldError = { field: string; message: string };

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasOnlyKeys(value: Record<string, unknown>, keys: string[]): boolean {
  return Object.keys(value).every((key) => keys.includes(key));
}

function isTrimmedText(value: unknown, min: number, max: number): value is string {
  return typeof value === "string" && value === value.trim() && value.length >= min && value.length <= max;
}

function validateItemIds(value: unknown): value is string[] {
  return Array.isArray(value)
    && value.length >= 1
    && value.length <= 20
    && value.every((item) => typeof item === "string" && ITEM_ID_PATTERN.test(item))
    && new Set(value).size === value.length;
}

export function parseCheckoutRoute(method: string, path: string[], params: URLSearchParams): CheckoutRouteValidation {
  const unexpectedQuery = [...params.keys()].find((key) => !["mockScenario", "cartScenario"].includes(key));
  if (unexpectedQuery) return { error: { field: unexpectedQuery, message: "Query không được hỗ trợ cho Checkout." } };

  const mockScenario = params.get("mockScenario") || undefined;
  const cartScenario = params.get("cartScenario") || undefined;
  if (mockScenario && !SCENARIO_PATTERN.test(mockScenario)) return { error: { field: "mockScenario", message: "Mock scenario không hợp lệ." } };
  if (cartScenario && !SCENARIO_PATTERN.test(cartScenario)) return { error: { field: "cartScenario", message: "Cart scenario không hợp lệ." } };
  if (method !== "POST" || path.length !== 1) return { error: { field: "path", message: "Checkout route is not available." } };
  if (path[0] === "prepare") return { operation: "prepare", mockScenario, cartScenario };
  if (path[0] === "shipping-quotes") return { operation: "quote-shipping", mockScenario, cartScenario };
  if (path[0] === "confirm") return { operation: "confirm", mockScenario, cartScenario };
  return { error: { field: "path", message: "Checkout route is not available." } };
}

export function parseMvpCheckoutRoute(method: string, path: string[], params: URLSearchParams): MvpCheckoutRouteValidation {
  const unexpectedQuery = [...params.keys()].find((key) => key !== "mockScenario");
  if (unexpectedQuery) return { error: { field: unexpectedQuery, message: "Query không được hỗ trợ cho Checkout." } };
  const mockScenario = params.get("mockScenario") || undefined;
  if (mockScenario && !SCENARIO_PATTERN.test(mockScenario)) return { error: { field: "mockScenario", message: "Mock scenario không hợp lệ." } };
  if (method === "POST" && path.length === 0) return { operation: "accept", mockScenario };
  if (method === "POST" && path.length === 1 && path[0] === "quote") return { operation: "create-quote", mockScenario };
  if (method === "GET" && path.length === 2 && path[0] === "operations") {
    if (!UUID_PATTERN.test(path[1])) return { error: { field: "operationId", message: "Mã Checkout operation không hợp lệ." } };
    return { operation: "get-operation", operationId: path[1], mockScenario };
  }
  return { error: { field: "path", message: "Checkout route is not available." } };
}

function validRevision(value: unknown): value is number {
  return Number.isSafeInteger(value) && Number(value) >= 0;
}

function validPaymentMethod(value: unknown): value is OrderPaymentMethod {
  return value === "VIETQR" || value === "COD";
}

function validateOrderAddress(value: unknown): value is OrderAddress {
  return isRecord(value)
    && hasOnlyKeys(value, ["recipient_name", "phone", "street", "ward_code", "province_code"])
    && [value.recipient_name, value.phone, value.street, value.ward_code, value.province_code]
      .every((field) => typeof field === "string" && field.trim().length >= 1 && field.length <= 255);
}

export function validateMvpQuoteRequest(value: unknown): { data?: CreateCheckoutQuoteRequest; errors: FieldError[] } {
  if (!isRecord(value) || !hasOnlyKeys(value, ["cart_revision", "payment_method", "shipping_address_id", "address", "voucher_code"])) {
    return { errors: [{ field: "body", message: "Yêu cầu tạo quote không đúng contract." }] };
  }
  const errors: FieldError[] = [];
  if (!validRevision(value.cart_revision)) errors.push({ field: "cart_revision", message: "Cart revision không hợp lệ." });
  if (!validPaymentMethod(value.payment_method)) errors.push({ field: "payment_method", message: "Phương thức thanh toán không hợp lệ." });
  const hasAddressId = value.shipping_address_id !== undefined;
  const hasAddress = value.address !== undefined;
  if (hasAddressId === hasAddress) errors.push({ field: "address", message: "Chỉ cung cấp shipping_address_id hoặc address." });
  if (hasAddressId && (typeof value.shipping_address_id !== "string" || !UUID_PATTERN.test(value.shipping_address_id))) {
    errors.push({ field: "shipping_address_id", message: "ID địa chỉ giao hàng không hợp lệ." });
  }
  if (hasAddress && !validateOrderAddress(value.address)) errors.push({ field: "address", message: "Địa chỉ giao hàng không hợp lệ." });
  if (value.voucher_code !== undefined && (typeof value.voucher_code !== "string" || !VOUCHER_PATTERN.test(value.voucher_code))) {
    errors.push({ field: "voucher_code", message: "Mã ưu đãi không hợp lệ." });
  }
  return errors.length ? { errors } : { data: value as unknown as CreateCheckoutQuoteRequest, errors };
}

export function validateMvpAcceptRequest(value: unknown): { data?: AcceptCheckoutRequest; errors: FieldError[] } {
  if (!isRecord(value) || !hasOnlyKeys(value, ["quote_id", "cart_revision", "payment_method"])) {
    return { errors: [{ field: "body", message: "Yêu cầu accept Checkout không đúng contract." }] };
  }
  const errors: FieldError[] = [];
  if (typeof value.quote_id !== "string" || !UUID_PATTERN.test(value.quote_id)) errors.push({ field: "quote_id", message: "Quote ID không hợp lệ." });
  if (!validRevision(value.cart_revision)) errors.push({ field: "cart_revision", message: "Cart revision không hợp lệ." });
  if (!validPaymentMethod(value.payment_method)) errors.push({ field: "payment_method", message: "Phương thức thanh toán không hợp lệ." });
  return errors.length ? { errors } : { data: value as unknown as AcceptCheckoutRequest, errors };
}

export function validateRecipient(value: unknown): FieldError[] {
  if (!isRecord(value) || !hasOnlyKeys(value, ["fullName", "phone", "email"])) return [{ field: "recipient", message: "Thông tin người nhận không hợp lệ." }];
  const errors: FieldError[] = [];
  if (!isTrimmedText(value.fullName, 2, 100)) errors.push({ field: "fullName", message: "Họ tên người nhận cần từ 2 đến 100 ký tự." });
  if (typeof value.phone !== "string" || !PHONE_PATTERN.test(value.phone)) errors.push({ field: "phone", message: "Số điện thoại người nhận không hợp lệ." });
  if (value.email !== undefined && value.email !== "" && (typeof value.email !== "string" || value.email.length > 254 || !EMAIL_PATTERN.test(value.email))) errors.push({ field: "email", message: "Email nhận hóa đơn không hợp lệ." });
  return errors;
}

export function validateShippingAddress(value: unknown): FieldError[] {
  if (!isRecord(value) || !hasOnlyKeys(value, ["provinceCode", "provinceName", "wardCode", "wardName", "addressLine"])) return [{ field: "address", message: "Địa chỉ giao hàng không hợp lệ." }];
  const errors: FieldError[] = [];
  if (!isTrimmedText(value.provinceCode, 2, 32) || !OPTION_ID_PATTERN.test(value.provinceCode)) errors.push({ field: "provinceCode", message: "Vui lòng chọn Tỉnh/Thành phố." });
  if (!isTrimmedText(value.provinceName, 2, 100)) errors.push({ field: "provinceName", message: "Tên Tỉnh/Thành phố không hợp lệ." });
  if (!isTrimmedText(value.wardCode, 2, 32) || !OPTION_ID_PATTERN.test(value.wardCode)) errors.push({ field: "wardCode", message: "Vui lòng chọn Phường/Xã giao hàng." });
  if (!isTrimmedText(value.wardName, 2, 100)) errors.push({ field: "wardName", message: "Tên Phường/Xã không hợp lệ." });
  if (!isTrimmedText(value.addressLine, 8, 250)) errors.push({ field: "addressLine", message: "Địa chỉ cụ thể cần từ 8 đến 250 ký tự." });
  return errors;
}

export function validatePrepareCheckoutInput(value: unknown): { data?: PrepareCheckoutInput; errors: FieldError[] } {
  if (!isRecord(value) || !hasOnlyKeys(value, ["itemIds"]) || !validateItemIds(value.itemIds)) {
    return { errors: [{ field: "itemIds", message: "Hãy chọn từ 1 đến 20 dòng giỏ hàng hợp lệ." }] };
  }
  return { data: { itemIds: value.itemIds }, errors: [] };
}

export function validateQuoteShippingInput(value: unknown): { data?: QuoteShippingInput; errors: FieldError[] } {
  if (!isRecord(value) || !hasOnlyKeys(value, ["checkoutSessionId", "itemIds", "address"])) return { errors: [{ field: "body", message: "Yêu cầu báo phí không hợp lệ." }] };
  const errors: FieldError[] = [];
  if (typeof value.checkoutSessionId !== "string" || !SESSION_ID_PATTERN.test(value.checkoutSessionId)) errors.push({ field: "checkoutSessionId", message: "Phiên Checkout không hợp lệ." });
  if (!validateItemIds(value.itemIds)) errors.push({ field: "itemIds", message: "Danh sách sản phẩm Checkout không hợp lệ." });
  errors.push(...validateShippingAddress(value.address));
  if (errors.length) return { errors };
  return { data: value as unknown as QuoteShippingInput, errors: [] };
}

export function validateConfirmCheckoutInput(value: unknown): { data?: ConfirmCheckoutInput; errors: FieldError[] } {
  if (!isRecord(value) || !hasOnlyKeys(value, ["checkoutSessionId", "itemIds", "recipient", "address", "shippingOptionId", "paymentMethod", "idempotencyKey", "priceRevalidatedAt", "priceChangesAcknowledged", "voucherCode"])) {
    return { errors: [{ field: "body", message: "Yêu cầu xác nhận Checkout không hợp lệ." }] };
  }
  const errors: FieldError[] = [];
  if (typeof value.checkoutSessionId !== "string" || !SESSION_ID_PATTERN.test(value.checkoutSessionId)) errors.push({ field: "checkoutSessionId", message: "Phiên Checkout không hợp lệ." });
  if (!validateItemIds(value.itemIds)) errors.push({ field: "itemIds", message: "Danh sách sản phẩm Checkout không hợp lệ." });
  errors.push(...validateRecipient(value.recipient), ...validateShippingAddress(value.address));
  if (typeof value.shippingOptionId !== "string" || !OPTION_ID_PATTERN.test(value.shippingOptionId)) errors.push({ field: "shippingOptionId", message: "Vui lòng chọn phương thức vận chuyển." });
  if (typeof value.paymentMethod !== "string" || !PAYMENT_METHODS.has(value.paymentMethod as CheckoutPaymentMethod)) errors.push({ field: "paymentMethod", message: "Vui lòng chọn phương thức thanh toán." });
  if (typeof value.idempotencyKey !== "string" || !IDEMPOTENCY_KEY_PATTERN.test(value.idempotencyKey)) errors.push({ field: "idempotencyKey", message: "Khóa chống gửi lặp không hợp lệ." });
  if (typeof value.priceRevalidatedAt !== "string" || Number.isNaN(Date.parse(value.priceRevalidatedAt))) errors.push({ field: "priceRevalidatedAt", message: "Mốc kiểm tra giá không hợp lệ." });
  if (typeof value.priceChangesAcknowledged !== "boolean") errors.push({ field: "priceChangesAcknowledged", message: "Xác nhận thay đổi giá không hợp lệ." });
  if (value.voucherCode !== undefined && (typeof value.voucherCode !== "string" || !VOUCHER_PATTERN.test(value.voucherCode))) errors.push({ field: "voucherCode", message: "Mã ưu đãi không hợp lệ." });
  if (errors.length) return { errors };
  return { data: value as unknown as ConfirmCheckoutInput, errors: [] };
}

export function getCheckoutFormErrors(recipient: CheckoutRecipient, address: ShippingAddress): Record<string, string> {
  return [...validateRecipient(recipient), ...validateShippingAddress(address)].reduce<Record<string, string>>((result, error) => {
    result[error.field] = error.message;
    return result;
  }, {});
}
