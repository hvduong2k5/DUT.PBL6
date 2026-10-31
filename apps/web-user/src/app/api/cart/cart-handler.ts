import { hasTrustedOrigin } from "@/lib/auth/request-origin";
import {
  normalizePublicError,
  type OrderCart,
  type OrderItem,
  type PublicErrorBody
} from "@/lib/api-contract";
import { authorizeCustomerCapability, customerAccessFailure } from "@/lib/auth/customer-access-server";
import { applyTrustedAuthorization, getTrustedAccessToken } from "@/lib/auth/server-credential";
import {
  parseMvpCartRoute,
  validateMvpAddCartItemInput,
  validateMvpUpdateCartItemInput
} from "@/lib/cart/validation";
import { NextRequest } from "next/server";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}



async function readJsonBody(request: NextRequest): Promise<unknown> {
  try {
    return await request.json() as unknown;
  } catch {
    return undefined;
  }
}

function publicErrorResponse(
  status: number,
  code: string,
  message: string,
  requestId: string,
  details?: Record<string, unknown>
): Response {
  const body: PublicErrorBody = { code, message, requestId, ...(details ? { details } : {}) };
  return Response.json(body, { status });
}

function upstreamHeaders(request: NextRequest, hasBody: boolean, scenario?: string): Headers {
  const headers = new Headers({ Accept: "application/json" });
  applyTrustedAuthorization(headers, request);
  if (hasBody) headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

async function callUpstream(request: NextRequest, path: string, method: string, scenario?: string, body?: unknown) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/${path}`, {
    method,
    headers: upstreamHeaders(request, body !== undefined, scenario),
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

function responseHeaders(upstream: Response): Headers {
  const headers = new Headers();
  for (const name of ["cache-control", "retry-after", "x-request-id"]) {
    const value = upstream.headers.get(name);
    if (value) headers.set(name, value);
  }
  return headers;
}

function relayUpstreamError(payload: unknown, upstream: Response): Response {
  const error = normalizePublicError(payload, {
    code: "CART_UPSTREAM_ERROR",
    message: "Không thể cập nhật giỏ hàng lúc này.",
    requestId: upstream.headers.get("x-request-id") ?? "BFF-CART-UPSTREAM"
  });
  return Response.json(error, { status: upstream.status, headers: responseHeaders(upstream) });
}

function isOrderItem(value: unknown): value is OrderItem {
  return isRecord(value)
    && typeof value.sku_code === "string"
    && Number.isSafeInteger(value.quantity)
    && Number(value.quantity) >= 1;
}

function isOrderCart(value: unknown): value is OrderCart {
  return isRecord(value)
    && Number.isSafeInteger(value.revision)
    && Number(value.revision) >= 0
    && Array.isArray(value.items)
    && value.items.every(isOrderItem);
}

function legacyMoneyUnits(value: unknown): number | undefined {
  if (!isRecord(value) || value.currency_code !== "VND" || value.nanos !== 0) return undefined;
  return Number.isSafeInteger(value.units) && Number(value.units) >= 0 ? Number(value.units) : undefined;
}

function legacyWeight(value: unknown): number | undefined {
  if (typeof value !== "string") return undefined;
  const match = value.match(/(\d+)\s*g\b/iu);
  return match ? Number(match[1]) : undefined;
}

function normalizeCart(payload: unknown, fallbackRevision: number, updatedItem?: { itemId: string; quantity: number }): OrderCart | undefined {
  if (isOrderCart(payload)) return payload;
  if (process.env.NODE_ENV === "production" || !isRecord(payload) || !Array.isArray(payload.items)) return undefined;
  const items: OrderItem[] = [];
  for (const value of payload.items) {
    if (!isRecord(value) || typeof value.item_id !== "string" || typeof value.sku_code !== "string" || !Number.isSafeInteger(value.quantity)) return undefined;
    const unitPrice = legacyMoneyUnits(value.unit_price);
    items.push({
      item_id: value.item_id,
      sku_code: value.sku_code,
      quantity: updatedItem?.itemId === value.item_id ? updatedItem.quantity : Number(value.quantity),
      ...(unitPrice !== undefined ? { unit_price: unitPrice } : {}),
      ...(typeof value.product_name === "string" ? { product_name: value.product_name } : {}),
      ...(legacyWeight(value.variant_name) !== undefined ? { weight_grams: legacyWeight(value.variant_name) } : {})
    });
  }
  return { revision: fallbackRevision, items };
}

export async function handleCartRequest(request: NextRequest, path: string[]): Promise<Response> {
  const validation = parseMvpCartRoute(request.method, path, request.nextUrl.searchParams);
  if (validation.error || !validation.operation) {
    const notFound = validation.error?.field === "path";
    return publicErrorResponse(
      notFound ? 404 : 400,
      notFound ? "NOT_FOUND" : "INVALID_REQUEST",
      validation.error?.message ?? "Yêu cầu không hợp lệ.",
      "BFF-CART-ROUTE",
      validation.error ? { errors: [validation.error] } : undefined
    );
  }
  if (request.method !== "GET" && !hasTrustedOrigin(request)) {
    return publicErrorResponse(403, "ORIGIN_NOT_ALLOWED", "Request origin is not allowed.", "BFF-CART-ORIGIN");
  }
  if (process.env.NODE_ENV === "production" && validation.mockScenario) {
    return publicErrorResponse(400, "INVALID_REQUEST", "Mock scenario không khả dụng trong production.", "BFF-CART-SCENARIO");
  }

  const accessToken = getTrustedAccessToken(request);
  try {
    await authorizeCustomerCapability(accessToken, "CART_MANAGE");
  } catch (cause) {
    const failure = customerAccessFailure(cause);
    return publicErrorResponse(failure.status, failure.code, failure.message, "BFF-CART-ACCESS");
  }

  try {
    if (validation.operation === "get-cart") {
      const result = await callUpstream(request, "cart", "GET", validation.mockScenario);
      if (!result.response.ok) return relayUpstreamError(result.payload, result.response);
      const cart = normalizeCart(result.payload, 0);
      return cart
        ? Response.json(cart, { headers: responseHeaders(result.response) })
        : publicErrorResponse(502, "INVALID_CART_RESPONSE", "Dữ liệu giỏ hàng không đúng contract.", "BFF-CART-RESPONSE");
    }

    if (validation.operation === "add-item") {
      const parsed = validateMvpAddCartItemInput(await readJsonBody(request));
      if (!parsed.data) return publicErrorResponse(400, "INVALID_REQUEST", "Dữ liệu thêm giỏ chưa hợp lệ.", "BFF-CART-ADD", { errors: parsed.errors });
      const result = await callUpstream(request, "cart/items", "POST", validation.mockScenario, parsed.data);
      if (!result.response.ok) return relayUpstreamError(result.payload, result.response);
      const cart = normalizeCart(result.payload, parsed.data.revision + 1);
      return cart
        ? Response.json(cart, { headers: responseHeaders(result.response) })
        : publicErrorResponse(502, "INVALID_CART_RESPONSE", "Dữ liệu giỏ hàng không đúng contract.", "BFF-CART-ADD");
    }

    if (validation.operation === "update-item") {
      const parsed = validateMvpUpdateCartItemInput(await readJsonBody(request));
      if (!parsed.data) return publicErrorResponse(400, "INVALID_REQUEST", "Dữ liệu cập nhật giỏ chưa hợp lệ.", "BFF-CART-UPDATE", { errors: parsed.errors });
      const result = await callUpstream(
        request,
        `cart/items/${encodeURIComponent(validation.itemId!)}`,
        "PUT",
        validation.mockScenario,
        parsed.data
      );
      if (!result.response.ok) return relayUpstreamError(result.payload, result.response);
      const cart = normalizeCart(result.payload, parsed.data.revision + 1, { itemId: validation.itemId!, quantity: parsed.data.quantity });
      return cart
        ? Response.json(cart, { headers: responseHeaders(result.response) })
        : publicErrorResponse(502, "INVALID_CART_RESPONSE", "Dữ liệu giỏ hàng không đúng contract.", "BFF-CART-UPDATE");
    }

    const deletion = await callUpstream(
      request,
      `cart/items/${encodeURIComponent(validation.itemId!)}`,
      "DELETE",
      validation.mockScenario
    );
    if (!deletion.response.ok) return relayUpstreamError(deletion.payload, deletion.response);
    const current = await callUpstream(request, "cart", "GET", validation.mockScenario);
    if (!current.response.ok) return relayUpstreamError(current.payload, current.response);
    const cart = normalizeCart(current.payload, 1);
    if (!cart) return publicErrorResponse(502, "INVALID_CART_RESPONSE", "Dữ liệu giỏ hàng không đúng contract.", "BFF-CART-DELETE");
    if (process.env.NODE_ENV !== "production") cart.items = cart.items.filter((item) => item.item_id !== validation.itemId);
    return Response.json(cart, { headers: responseHeaders(current.response) });
  } catch {
    return publicErrorResponse(503, "CART_UPSTREAM_UNAVAILABLE", "Dịch vụ giỏ hàng đang tạm gián đoạn. Hãy kiểm tra API Gateway hoặc Mockoon.", "BFF-CART-UPSTREAM");
  }
}
