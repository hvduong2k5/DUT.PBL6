import { randomUUID } from "node:crypto";
import { mapCustomerCoreCart, type CustomerCoreCart } from "@/lib/cart/customer-core";
import {
  buildCustomerCoreCheckoutRequest,
  mapCustomerCoreCheckoutPreferences,
  mapCustomerCoreCheckoutConfirmation,
  type CustomerCoreCheckoutResponse
} from "@/lib/checkout/customer-core";
import type { CheckoutLine, CheckoutPreparation, ShippingQuote } from "@/lib/checkout/types";
import { parseCheckoutLocations } from "@/lib/checkout/locations";
import type { CustomerCoreAddress, CustomerCoreProfile } from "@/lib/customer/customer-core";
import {
  parseCheckoutRoute,
  validateConfirmCheckoutInput,
  validatePrepareCheckoutInput,
  validateQuoteShippingInput
} from "@/lib/checkout/validation";
import { PAYMENT_ACCESS_COOKIE, encodePaymentAccess, type PaymentAccessClaim } from "@/lib/payment/access";
import { fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import { GUEST_ORDER_ACCESS_COOKIE, encodeGuestOrderAccess, type GuestOrderAccessClaim } from "@/lib/orders/guest-access";
import { NextRequest, NextResponse } from "next/server";

const CART_COOKIE = "oma_cart_context";
const CART_CONTEXT_PATTERN = /^[0-9a-f-]{36}$/u;

function withCartCookie(response: NextResponse, contextId: string, shouldSet: boolean): NextResponse {
  if (shouldSet) response.cookies.set(CART_COOKIE, contextId, {
    path: "/",
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    maxAge: 2_592_000
  });
  return response;
}

function errorResponse(code: string, message: string, status: number, contextId: string, shouldSet: boolean, errors: Array<{ field: string; message: string }> = []) {
  return withCartCookie(NextResponse.json({ code, message, errors, requestId: `BFF-CHECKOUT-${code}` }, { status }), contextId, shouldSet);
}

function getCartContext(request: NextRequest) {
  const existing = request.cookies.get(CART_COOKIE)?.value;
  const validExisting = existing && CART_CONTEXT_PATTERN.test(existing) ? existing : undefined;
  return { contextId: validExisting ?? randomUUID(), shouldSet: !validExisting };
}

async function readJsonBody(request: NextRequest): Promise<unknown> {
  try {
    return await request.json();
  } catch {
    return undefined;
  }
}

function upstreamHeaders(request: NextRequest, contextId: string, scenario?: string, hasBody = false) {
  const headers = new Headers({ Accept: "application/json", "X-Session-Id": contextId });
  const authorization = request.headers.get("authorization");
  const accessToken = request.cookies.get("oma_access_token")?.value;
  const cookie = request.headers.get("cookie");
  if (authorization) headers.set("Authorization", authorization);
  else if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (cookie) headers.set("Cookie", cookie);
  if (hasBody) headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

async function loadSelectedLines(request: NextRequest, contextId: string, itemIds: string[], scenario?: string): Promise<CheckoutLine[]> {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/cart`, { headers: upstreamHeaders(request, contextId, scenario), cache: "no-store", redirect: "manual" });
  const payload = await response.json() as CustomerCoreCart & { error_code?: string; user_message?: string };
  if (!response.ok) throw { status: response.status, payload };
  const cart = mapCustomerCoreCart(payload);
  const requested = new Set(itemIds);
  const selected = cart.items.filter((item) => requested.has(item.itemId));
  if (selected.length !== requested.size) {
    throw { status: 409, payload: { error_code: "CART_CHANGED", user_message: "Giỏ hàng đã thay đổi. Vui lòng quay lại giỏ và chọn lại sản phẩm." } };
  }
  if (selected.some((item) => !item.isAvailable)) {
    throw { status: 409, payload: { error_code: "CHECKOUT_ITEMS_UNAVAILABLE", user_message: "Một số sản phẩm đã hết hàng. Vui lòng kiểm tra lại giỏ." } };
  }
  return selected.map((item) => ({
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
    lineSubtotalVnd: item.lineSubtotalVnd ?? item.unitPriceVnd * item.quantity,
    priceChanged: false,
    previousUnitPriceVnd: null
  }));
}

async function loadRegisteredCheckoutPreferences(request: NextRequest, contextId: string, canReadProfile: boolean, canReadAddresses: boolean) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const load = async (path: string) => {
    const response = await fetch(`${base}/${path}`, { headers: upstreamHeaders(request, contextId), cache: "no-store", redirect: "manual" });
    if (!response.ok) throw new Error(`Registered checkout source ${path} returned ${response.status}`);
    return await response.json() as unknown;
  };
  const [profileResult, addressesResult] = await Promise.allSettled([
    canReadProfile ? load("profile") : Promise.resolve(undefined),
    canReadAddresses ? load("profile/addresses") : Promise.resolve(undefined)
  ]);
  const profile = profileResult.status === "fulfilled" ? profileResult.value as CustomerCoreProfile | undefined : undefined;
  const addressPayload = addressesResult.status === "fulfilled" ? addressesResult.value as { addresses?: CustomerCoreAddress[] } | undefined : undefined;
  const preferences = mapCustomerCoreCheckoutPreferences(profile, addressPayload?.addresses ?? []);
  const unavailable = (canReadProfile && profileResult.status === "rejected") || (canReadAddresses && addressesResult.status === "rejected");
  return {
    ...preferences,
    ...(unavailable ? { accountDataWarning: "Chưa thể tải đầy đủ hồ sơ hoặc sổ địa chỉ. Bạn vẫn có thể nhập thông tin nhận hàng cho đơn này." } : {})
  };
}

async function loadCheckoutLocations(request: NextRequest, contextId: string, scenario?: string) {
  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/locations/checkout`, { headers: upstreamHeaders(request, contextId, scenario), cache: "no-store", redirect: "manual" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  if (!response.ok) throw { status: response.status, payload };
  const locations = parseCheckoutLocations(payload);
  if (!locations) throw { status: 502, payload: { code: "CHECKOUT_LOCATIONS_INVALID", message: "Danh mục Tỉnh/Thành phố và Phường/Xã không hợp lệ." } };
  return locations;
}

function preparation(
  lines: CheckoutLine[],
  customerMode: "GUEST" | "REGISTERED",
  locations: CheckoutPreparation["locations"],
  preferences: Pick<CheckoutPreparation, "defaultRecipient" | "savedAddresses" | "accountDataWarning"> = { savedAddresses: [] }
): CheckoutPreparation {
  return {
    checkoutSessionId: `checkout-${randomUUID()}`,
    items: lines,
    itemCount: lines.reduce((total, line) => total + line.quantity, 0),
    subtotalVnd: lines.reduce((total, line) => total + line.lineSubtotalVnd, 0),
    priceRevalidatedAt: new Date().toISOString(),
    requiresPriceAcknowledgement: false,
    notices: [],
    customerMode,
    defaultRecipient: preferences.defaultRecipient,
    savedAddresses: preferences.savedAddresses,
    locations,
    accountDataWarning: preferences.accountDataWarning
  };
}

function calculatedShippingQuote(): ShippingQuote {
  const now = Date.now();
  return {
    options: [{
      shippingOptionId: "CALCULATED-AT-CHECKOUT",
      name: "Giao hàng tiêu chuẩn",
      description: "Phí được hệ thống tính khi tạo đơn",
      feeVnd: 0,
      estimatedDelivery: "Thời gian giao dự kiến sẽ hiển thị trong chi tiết đơn",
      calculatedAtCheckout: true
    }],
    quotedAt: new Date(now).toISOString(),
    expiresAt: new Date(now + 15 * 60 * 1000).toISOString()
  };
}

function isUpstreamFailure(value: unknown): value is { status: number; payload: unknown } {
  return Boolean(value && typeof value === "object" && "status" in value && "payload" in value);
}

async function grantGuestOrderAccess(
  request: NextRequest,
  claim: Omit<GuestOrderAccessClaim, "accessExpiresAt">,
  contact: { phone: string; email?: string },
  idempotencyKey: string,
  scenario?: string
): Promise<GuestOrderAccessClaim> {
  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const headers = upstreamHeaders(request, randomUUID(), scenario, true);
  headers.set("X-Idempotency-Key", idempotencyKey);
  const response = await fetch(`${base}/orders/guest-access/grants`, {
    method: "POST",
    headers,
    body: JSON.stringify({ ...claim, contact }),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as Record<string, unknown> : {};
  if (!response.ok) throw { status: response.status, payload };
  const expiresAt = typeof payload.accessExpiresAt === "string" ? Date.parse(payload.accessExpiresAt) : Number.NaN;
  if (payload.granted !== true || !Number.isFinite(expiresAt) || expiresAt <= Date.now() || expiresAt > Date.now() + 24 * 60 * 60 * 1000) {
    throw {
      status: 502,
      payload: { code: "GUEST_ACCESS_GRANT_INVALID", message: "Order đã được tạo nhưng chưa thể cấp quyền xem đơn cho Guest." }
    };
  }
  return { ...claim, accessExpiresAt: expiresAt };
}

export async function handleCheckoutRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parseCheckoutRoute(request.method, path, request.nextUrl.searchParams);
  const { contextId, shouldSet } = getCartContext(request);
  if (route.error || !route.operation) {
    const notFound = route.error?.field === "path";
    return errorResponse(notFound ? "NOT_FOUND" : "INVALID_REQUEST", route.error?.message ?? "Yêu cầu không hợp lệ.", notFound ? 404 : 400, contextId, shouldSet, route.error ? [route.error] : []);
  }
  if (process.env.NODE_ENV === "production" && (route.mockScenario || route.cartScenario)) {
    return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400, contextId, shouldSet);
  }
  let customerMode: "GUEST" | "REGISTERED" = "GUEST";
  let canReadProfile = false;
  let canReadAddresses = false;
  try {
    const projection = await fetchCustomerCapabilities(request.cookies.get("oma_access_token")?.value);
    if (!projection.capabilities.includes("CHECKOUT_CREATE")) return errorResponse("CAPABILITY_FORBIDDEN", "Actor hiện tại không có quyền tạo Checkout.", 403, contextId, shouldSet);
    customerMode = projection.actor === "GUEST" ? "GUEST" : "REGISTERED";
    canReadProfile = projection.capabilities.includes("PROFILE_MANAGE");
    canReadAddresses = projection.capabilities.includes("ADDRESS_MANAGE");
  } catch {
    return errorResponse("CAPABILITY_UNAVAILABLE", "Chưa thể kiểm tra quyền Customer lúc này.", 503, contextId, shouldSet);
  }

  try {
    const body = await readJsonBody(request);
    if (route.operation === "prepare") {
      const parsed = validatePrepareCheckoutInput(body);
      if (!parsed.data) return errorResponse("INVALID_REQUEST", "Sản phẩm Checkout chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
      const [lines, locations, preferences] = await Promise.all([
        loadSelectedLines(request, contextId, parsed.data.itemIds, route.cartScenario),
        loadCheckoutLocations(request, contextId, route.mockScenario),
        customerMode === "REGISTERED"
          ? loadRegisteredCheckoutPreferences(request, contextId, canReadProfile, canReadAddresses)
          : Promise.resolve({ savedAddresses: [] })
      ]);
      return withCartCookie(NextResponse.json(preparation(lines, customerMode, locations, preferences)), contextId, shouldSet);
    }

    if (route.operation === "quote-shipping") {
      const parsed = validateQuoteShippingInput(body);
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Địa chỉ giao hàng chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
      await loadSelectedLines(request, contextId, parsed.data.itemIds, route.cartScenario);
      return withCartCookie(NextResponse.json(calculatedShippingQuote()), contextId, shouldSet);
    }

    const parsed = validateConfirmCheckoutInput(body);
    if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Thông tin xác nhận đơn chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
    const lines = await loadSelectedLines(request, contextId, parsed.data.itemIds, route.cartScenario);
    const checkoutBody = buildCustomerCoreCheckoutRequest(lines, parsed.data.recipient, parsed.data.address, parsed.data.paymentMethod, parsed.data.voucherCode);
    const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
    const headers = upstreamHeaders(request, contextId, route.mockScenario, true);
    headers.set("X-Idempotency-Key", parsed.data.idempotencyKey);
    const upstream = await fetch(`${base}/checkout`, {
      method: "POST",
      headers,
      body: JSON.stringify(checkoutBody),
      cache: "no-store",
      redirect: "manual"
    });
    const upstreamPayload = await upstream.json() as CustomerCoreCheckoutResponse & { error_code?: string; user_message?: string };
    if (!upstream.ok) throw { status: upstream.status, payload: upstreamPayload };
    const confirmation = mapCustomerCoreCheckoutConfirmation(upstreamPayload, parsed.data.paymentMethod, customerMode);
    const guestClaim = customerMode === "GUEST" ? await grantGuestOrderAccess(
      request,
      { orderId: confirmation.orderId, orderNumber: confirmation.orderNumber },
      { phone: parsed.data.recipient.phone, ...(parsed.data.recipient.email ? { email: parsed.data.recipient.email } : {}) },
      parsed.data.idempotencyKey,
      route.mockScenario
    ) : undefined;
    const response = withCartCookie(NextResponse.json(confirmation, { status: 201 }), contextId, shouldSet);
    if (guestClaim) response.cookies.set(GUEST_ORDER_ACCESS_COOKIE, encodeGuestOrderAccess(guestClaim), {
      path: "/api/orders",
      httpOnly: true,
      sameSite: "lax",
      secure: process.env.NODE_ENV === "production",
      maxAge: Math.max(1, Math.floor((guestClaim.accessExpiresAt - Date.now()) / 1000))
    });
    const paymentClaim: PaymentAccessClaim = {
      orderId: confirmation.orderId,
      orderNumber: confirmation.orderNumber,
      method: confirmation.paymentMethod,
      amountVnd: confirmation.totalVnd,
      paymentExpiresAt: confirmation.paymentExpiresAt ?? confirmation.reservationExpiresAt,
      accessExpiresAt: Date.now() + 30 * 60 * 1000,
      vietQrUrl: confirmation.vietQrUrl
    };
    response.cookies.set(PAYMENT_ACCESS_COOKIE, encodePaymentAccess(paymentClaim), {
      path: "/api/payments",
      httpOnly: true,
      sameSite: "lax",
      secure: process.env.NODE_ENV === "production",
      maxAge: Math.max(1, Math.floor((paymentClaim.accessExpiresAt - Date.now()) / 1000))
    });
    return response;
  } catch (cause) {
    if (isUpstreamFailure(cause)) {
      return withCartCookie(NextResponse.json(cause.payload, { status: cause.status }), contextId, shouldSet);
    }
    return errorResponse("CHECKOUT_UPSTREAM_UNAVAILABLE", "Dịch vụ Checkout đang tạm gián đoạn. Hãy kiểm tra Mockoon customer API.", 503, contextId, shouldSet);
  }
}
