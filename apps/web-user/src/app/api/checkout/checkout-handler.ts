import { randomUUID } from "node:crypto";
import type { CheckoutLine, CheckoutPreparation, SavedCheckoutAddress, ShippingOption, ShippingQuote } from "@/lib/checkout/types";
import {
  parseCheckoutRoute,
  validateConfirmCheckoutInput,
  validatePrepareCheckoutInput,
  validateQuoteShippingInput
} from "@/lib/checkout/validation";
import { NextRequest, NextResponse } from "next/server";

const CART_COOKIE = "oma_cart_context";
const CART_CONTEXT_PATTERN = /^[0-9a-f-]{36}$/u;
const CART_MOCK_CONTEXTS = new Set(["cart-prefilled", "cart-empty", "price-changed", "cart-unavailable"]);

interface CartLineRecord {
  id: string;
  contextId: string;
  skuId: string;
  quantity: number;
  unitPriceVndAtAddition: number;
  productSlug: string;
  productName: string;
  imageUrl: string;
  imageAlt: string;
  skuLabel: string;
  weightGrams: number;
  flavor?: string | null;
  packageType: string;
}

interface SkuProjection {
  skuId: string;
  productSlug: string;
  productName: string;
  imageUrl: string;
  imageAlt: string;
  skuLabel: string;
  weightGrams: number;
  flavor?: string | null;
  packageType: string;
  unitPriceVnd: number;
  isAvailable: boolean;
  unavailableReason?: string | null;
  maxPurchasableQuantity: number;
}

interface CheckoutBootstrap {
  customerMode: "GUEST" | "REGISTERED";
  savedAddresses: SavedCheckoutAddress[];
}

interface UpstreamConfirmation {
  orderId: string;
  orderNumber: string;
  status: "PENDING_PAYMENT" | "PLACED";
  paymentStatus: "UNPAID";
  reservationExpiresAt: string;
  nextStep: "PAYMENT_REQUIRED" | "ORDER_PLACED";
  message: string;
}

interface UpstreamErrorBody {
  code?: string;
  message?: string;
  errors?: Array<{ field: string; message: string }>;
  itemIssues?: Array<{ itemId: string; code: string; message: string }>;
}

class UpstreamHttpError extends Error {
  constructor(readonly status: number, readonly body: UpstreamErrorBody = {}) {
    super(body.message || `Checkout upstream returned ${status}`);
  }
}

function cartCookieHeader(contextId: string): string {
  const secure = process.env.NODE_ENV === "production" ? "; Secure" : "";
  return `${CART_COOKIE}=${contextId}; Path=/; HttpOnly; SameSite=Lax; Max-Age=2592000${secure}`;
}

function withCartCookie(response: NextResponse, contextId: string, shouldSet: boolean): NextResponse {
  if (shouldSet) response.headers.append("Set-Cookie", cartCookieHeader(contextId));
  return response;
}

function errorResponse(
  code: string,
  message: string,
  status: number,
  contextId: string,
  shouldSet: boolean,
  errors: Array<{ field: string; message: string }> = [],
  itemIssues: Array<{ itemId: string; code: string; message: string }> = []
) {
  return withCartCookie(NextResponse.json({ code, message, errors, itemIssues, requestId: `BFF-CHECKOUT-${code}` }, { status }), contextId, shouldSet);
}

function getCartContext(request: NextRequest, cartScenario?: string) {
  const existing = request.cookies.get(CART_COOKIE)?.value;
  const validExisting = existing && CART_CONTEXT_PATTERN.test(existing) ? existing : undefined;
  const contextId = validExisting ?? randomUUID();
  const mockContextId = process.env.NODE_ENV !== "production" && cartScenario && CART_MOCK_CONTEXTS.has(cartScenario)
    ? `mock-${cartScenario}`
    : contextId;
  return { contextId, mockContextId, shouldSet: !validExisting };
}

async function readJsonBody(request: NextRequest): Promise<unknown> {
  try {
    return await request.json();
  } catch {
    return undefined;
  }
}

async function upstreamJson<T>(baseUrl: string, path: string, scenario?: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body) headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  const response = await fetch(`${baseUrl.replace(/\/$/u, "")}/${path}`, { ...init, headers, cache: "no-store", redirect: "manual" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : undefined;
  if (!response.ok) throw new UpstreamHttpError(response.status, (payload ?? {}) as UpstreamErrorBody);
  return payload as T;
}

async function loadSelectedLines(contextId: string, itemIds: string[], cartScenario?: string): Promise<CheckoutLine[]> {
  const cartBase = process.env.CART_UPSTREAM_URL ?? "http://127.0.0.1:4014/api/v1";
  const params = new URLSearchParams({ contextId_eq: contextId, sort: "createdAt", order: "asc" });
  const [lines, projections] = await Promise.all([
    upstreamJson<CartLineRecord[]>(cartBase, `internal/cart-lines?${params}`, cartScenario),
    upstreamJson<SkuProjection[]>(cartBase, "internal/cart-sku-projections", cartScenario)
  ]);
  const requested = new Set(itemIds);
  const selected = lines.filter((line) => requested.has(line.id));
  if (selected.length !== requested.size) {
    const found = new Set(selected.map((line) => line.id));
    throw new UpstreamHttpError(409, {
      code: "CART_CHANGED",
      message: "Giỏ hàng đã thay đổi. Vui lòng quay lại giỏ và chọn lại sản phẩm.",
      itemIssues: itemIds.filter((itemId) => !found.has(itemId)).map((itemId) => ({ itemId, code: "ITEM_NOT_FOUND", message: "Dòng sản phẩm không còn trong giỏ hiện tại." }))
    });
  }
  const projectionBySku = new Map(projections.map((projection) => [projection.skuId, projection]));
  const issues: Array<{ itemId: string; code: string; message: string }> = [];
  const result = selected.map((line) => {
    const current = projectionBySku.get(line.skuId);
    if (!current?.isAvailable) issues.push({ itemId: line.id, code: "SKU_UNAVAILABLE", message: current?.unavailableReason || "Quy cách này không còn khả dụng." });
    else if (line.quantity > current.maxPurchasableQuantity) issues.push({ itemId: line.id, code: "QUANTITY_UNAVAILABLE", message: "Số lượng hiện không thể đáp ứng. Vui lòng điều chỉnh giỏ hàng." });
    const unitPriceVnd = current?.unitPriceVnd ?? line.unitPriceVndAtAddition;
    return {
      itemId: line.id,
      productSlug: current?.productSlug ?? line.productSlug,
      productName: current?.productName ?? line.productName,
      imageUrl: current?.imageUrl ?? line.imageUrl,
      imageAlt: current?.imageAlt ?? line.imageAlt,
      skuId: line.skuId,
      skuLabel: current?.skuLabel ?? line.skuLabel,
      weightGrams: current?.weightGrams ?? line.weightGrams,
      flavor: current?.flavor ?? line.flavor,
      packageType: current?.packageType ?? line.packageType,
      unitPriceVnd,
      quantity: line.quantity,
      lineSubtotalVnd: unitPriceVnd * line.quantity,
      priceChanged: Boolean(current && unitPriceVnd !== line.unitPriceVndAtAddition),
      previousUnitPriceVnd: current && unitPriceVnd !== line.unitPriceVndAtAddition ? line.unitPriceVndAtAddition : null
    } satisfies CheckoutLine;
  });
  if (issues.length) throw new UpstreamHttpError(409, { code: "CHECKOUT_ITEMS_UNAVAILABLE", message: "Một số sản phẩm cần được điều chỉnh trước khi Checkout.", itemIssues: issues });
  return result;
}

function publicShippingQuote(value: ShippingQuote): ShippingQuote {
  const options = Array.isArray(value.options) ? value.options.filter((option): option is ShippingOption => (
    typeof option?.shippingOptionId === "string"
    && typeof option.name === "string"
    && typeof option.description === "string"
    && Number.isInteger(option.feeVnd) && option.feeVnd >= 0
    && typeof option.estimatedDelivery === "string"
  )).map((option) => ({
    shippingOptionId: option.shippingOptionId,
    name: option.name,
    description: option.description,
    feeVnd: option.feeVnd,
    estimatedDelivery: option.estimatedDelivery
  })) : [];
  return { options, quotedAt: value.quotedAt, expiresAt: value.expiresAt };
}

function publicBootstrap(value: CheckoutBootstrap): CheckoutBootstrap {
  const savedAddresses = Array.isArray(value.savedAddresses) ? value.savedAddresses.map((saved) => ({
    addressId: saved.addressId,
    label: saved.label,
    recipient: { fullName: saved.recipient.fullName, phone: saved.recipient.phone, email: saved.recipient.email },
    address: {
      provinceCode: saved.address.provinceCode,
      provinceName: saved.address.provinceName,
      districtCode: saved.address.districtCode,
      districtName: saved.address.districtName,
      addressLine: saved.address.addressLine
    },
    isDefault: Boolean(saved.isDefault)
  })) : [];
  return { customerMode: value.customerMode === "REGISTERED" ? "REGISTERED" : "GUEST", savedAddresses };
}

function preparation(lines: CheckoutLine[], bootstrap: CheckoutBootstrap): CheckoutPreparation {
  const priceRevalidatedAt = new Date().toISOString();
  const changed = lines.filter((line) => line.priceChanged);
  return {
    checkoutSessionId: `checkout-${randomUUID()}`,
    items: lines,
    itemCount: lines.reduce((total, line) => total + line.quantity, 0),
    subtotalVnd: lines.reduce((total, line) => total + line.lineSubtotalVnd, 0),
    priceRevalidatedAt,
    requiresPriceAcknowledgement: changed.length > 0,
    notices: changed.map((line) => ({ code: "PRICE_CHANGED", itemId: line.itemId, message: `Giá ${line.skuLabel} đã được cập nhật theo giá bán hiện tại.` })),
    customerMode: bootstrap.customerMode,
    savedAddresses: bootstrap.savedAddresses
  };
}

export async function handleCheckoutRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parseCheckoutRoute(request.method, path, request.nextUrl.searchParams);
  const { contextId, mockContextId, shouldSet } = getCartContext(request, route.cartScenario);
  if (route.error || !route.operation) {
    const notFound = route.error?.field === "path";
    return errorResponse(notFound ? "NOT_FOUND" : "INVALID_REQUEST", route.error?.message ?? "Yêu cầu không hợp lệ.", notFound ? 404 : 400, contextId, shouldSet, route.error ? [route.error] : []);
  }
  if (process.env.NODE_ENV === "production" && (route.mockScenario || route.cartScenario)) {
    return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400, contextId, shouldSet);
  }

  const checkoutBase = process.env.CHECKOUT_UPSTREAM_URL ?? "http://127.0.0.1:4015/api/v1";
  try {
    const body = await readJsonBody(request);
    if (route.operation === "prepare") {
      const parsed = validatePrepareCheckoutInput(body);
      if (!parsed.data) return errorResponse("INVALID_REQUEST", "Sản phẩm Checkout chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
      const [lines, bootstrap] = await Promise.all([
        loadSelectedLines(mockContextId, parsed.data.itemIds, route.cartScenario),
        upstreamJson<CheckoutBootstrap>(checkoutBase, "checkout/bootstrap", route.mockScenario)
      ]);
      return withCartCookie(NextResponse.json(preparation(lines, publicBootstrap(bootstrap))), contextId, shouldSet);
    }

    if (route.operation === "quote-shipping") {
      const parsed = validateQuoteShippingInput(body);
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Địa chỉ giao hàng chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
      const lines = await loadSelectedLines(mockContextId, parsed.data.itemIds, route.cartScenario);
      const quote = await upstreamJson<ShippingQuote>(checkoutBase, "checkout/shipping-quotes", route.mockScenario, {
        method: "POST",
        body: JSON.stringify({
          checkoutSessionId: parsed.data.checkoutSessionId,
          address: parsed.data.address,
          parcel: {
            itemCount: lines.reduce((total, line) => total + line.quantity, 0),
            totalWeightGrams: lines.reduce((total, line) => total + line.weightGrams * line.quantity, 0)
          }
        })
      });
      return withCartCookie(NextResponse.json(publicShippingQuote(quote)), contextId, shouldSet);
    }

    const parsed = validateConfirmCheckoutInput(body);
    if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Thông tin xác nhận đơn chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
    const lines = await loadSelectedLines(mockContextId, parsed.data.itemIds, route.cartScenario);
    if (lines.some((line) => line.priceChanged) && !parsed.data.priceChangesAcknowledged) {
      return errorResponse("PRICE_ACKNOWLEDGEMENT_REQUIRED", "Vui lòng xác nhận giá hiện tại trước khi đặt hàng.", 409, contextId, shouldSet);
    }
    const quote = publicShippingQuote(await upstreamJson<ShippingQuote>(checkoutBase, "checkout/shipping-quotes", undefined, {
      method: "POST",
      body: JSON.stringify({
        checkoutSessionId: parsed.data.checkoutSessionId,
        address: parsed.data.address,
        parcel: {
          itemCount: lines.reduce((total, line) => total + line.quantity, 0),
          totalWeightGrams: lines.reduce((total, line) => total + line.weightGrams * line.quantity, 0)
        }
      })
    }));
    if (!Number.isFinite(Date.parse(quote.expiresAt)) || Date.parse(quote.expiresAt) <= Date.now()) {
      return errorResponse("SHIPPING_QUOTE_EXPIRED", "Báo phí vận chuyển đã hết hiệu lực. Vui lòng tính lại phí.", 409, contextId, shouldSet);
    }
    const shipping = quote.options.find((option) => option.shippingOptionId === parsed.data!.shippingOptionId);
    if (!shipping) return errorResponse("SHIPPING_OPTION_UNAVAILABLE", "Phương thức giao hàng đã chọn không còn khả dụng. Vui lòng tính lại phí.", 409, contextId, shouldSet);
    const subtotalVnd = lines.reduce((total, line) => total + line.lineSubtotalVnd, 0);
    const upstreamConfirmation = await upstreamJson<UpstreamConfirmation>(checkoutBase, "checkout/confirm", route.mockScenario, {
      method: "POST",
      headers: { "Idempotency-Key": parsed.data.idempotencyKey },
      body: JSON.stringify({
        checkoutSessionId: parsed.data.checkoutSessionId,
        idempotencyKey: parsed.data.idempotencyKey,
        recipient: parsed.data.recipient,
        shippingAddress: parsed.data.address,
        shipping,
        paymentMethod: parsed.data.paymentMethod,
        commercialSnapshot: { lines, subtotalVnd, shippingFeeVnd: shipping.feeVnd, discountVnd: 0, totalVnd: subtotalVnd + shipping.feeVnd }
      })
    });
    const confirmation = {
      orderId: upstreamConfirmation.orderId,
      orderNumber: upstreamConfirmation.orderNumber,
      status: upstreamConfirmation.status,
      paymentStatus: "UNPAID" as const,
      paymentMethod: parsed.data.paymentMethod,
      reservationExpiresAt: upstreamConfirmation.reservationExpiresAt,
      subtotalVnd,
      shippingFeeVnd: shipping.feeVnd,
      discountVnd: 0,
      totalVnd: subtotalVnd + shipping.feeVnd,
      nextStep: upstreamConfirmation.nextStep,
      message: upstreamConfirmation.message
    };
    return withCartCookie(NextResponse.json(confirmation, { status: 201 }), contextId, shouldSet);
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) {
      const status = cause.status >= 400 && cause.status <= 599 ? cause.status : 500;
      return errorResponse(cause.body.code || "CHECKOUT_ERROR", cause.body.message || "Không thể xử lý Checkout lúc này.", status, contextId, shouldSet, cause.body.errors ?? [], cause.body.itemIssues ?? []);
    }
    const message = process.env.NODE_ENV === "production"
      ? "Dịch vụ Checkout đang tạm gián đoạn. Vui lòng thử lại sau."
      : "Checkout Mockoon chưa sẵn sàng trên port 4015 hoặc Cart Mockoon chưa chạy trên port 4014.";
    return errorResponse("CHECKOUT_UPSTREAM_UNAVAILABLE", message, 503, contextId, shouldSet);
  }
}
