import { randomUUID } from "node:crypto";
import { emptyCart, mapCustomerCoreCart, withCartItemQuantity, type CustomerCoreCart } from "@/lib/cart/customer-core";
import { parseCartRoute, validateAddCartItemInput, validateUpdateCartItemInput } from "@/lib/cart/validation";
import { fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import { NextRequest, NextResponse } from "next/server";

const CART_COOKIE = "oma_cart_context";
const CART_CONTEXT_PATTERN = /^[0-9a-f-]{36}$/u;

function cookieHeader(contextId: string): string {
  const secure = process.env.NODE_ENV === "production" ? "; Secure" : "";
  return `${CART_COOKIE}=${contextId}; Path=/; HttpOnly; SameSite=Lax; Max-Age=2592000${secure}`;
}

function withContextCookie(response: NextResponse, contextId: string, shouldSet: boolean): NextResponse {
  if (shouldSet) response.headers.append("Set-Cookie", cookieHeader(contextId));
  return response;
}

function errorResponse(code: string, message: string, status: number, contextId: string, shouldSet: boolean, errors: Array<{ field: string; message: string }> = []) {
  return withContextCookie(NextResponse.json({ code, message, errors, requestId: `BFF-CART-${code}` }, { status }), contextId, shouldSet);
}

function getContext(request: NextRequest) {
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

async function callUpstream(request: NextRequest, path: string, method: string, contextId: string, scenario?: string, body?: unknown) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json", "X-Session-Id": contextId });
  const authorization = request.headers.get("authorization");
  const accessToken = request.cookies.get("oma_access_token")?.value;
  const cookie = request.headers.get("cookie");
  if (authorization) headers.set("Authorization", authorization);
  else if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (cookie) headers.set("Cookie", cookie);
  if (body !== undefined) headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  const response = await fetch(`${base}/${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

function upstreamError(payload: unknown, status: number, contextId: string, shouldSet: boolean) {
  const body = payload && typeof payload === "object"
    ? payload
    : { error_code: "ERR_CART", user_message: "Không thể cập nhật giỏ hàng lúc này." };
  return withContextCookie(NextResponse.json(body, { status }), contextId, shouldSet);
}

export async function handleCartRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const validation = parseCartRoute(request.method, path, request.nextUrl.searchParams);
  const { contextId, shouldSet } = getContext(request);
  if (validation.error || !validation.operation) {
    const notFound = validation.error?.field === "path";
    return errorResponse(notFound ? "NOT_FOUND" : "INVALID_REQUEST", validation.error?.message ?? "Yêu cầu không hợp lệ.", notFound ? 404 : 400, contextId, shouldSet, validation.error ? [validation.error] : []);
  }
  if (process.env.NODE_ENV === "production" && validation.mockScenario) {
    return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400, contextId, shouldSet);
  }
  try {
    const projection = await fetchCustomerCapabilities(request.cookies.get("oma_access_token")?.value);
    if (!projection.capabilities.includes("CART_MANAGE")) return errorResponse("CAPABILITY_FORBIDDEN", "Actor hiện tại không có quyền sử dụng giỏ hàng.", 403, contextId, shouldSet);
  } catch {
    return errorResponse("CAPABILITY_UNAVAILABLE", "Chưa thể kiểm tra quyền Customer lúc này.", 503, contextId, shouldSet);
  }

  try {
    if (validation.operation === "clear-cart") {
      const { response, payload } = await callUpstream(request, "cart", "DELETE", contextId, validation.mockScenario);
      if (!response.ok) return upstreamError(payload, response.status, contextId, shouldSet);
      return withContextCookie(NextResponse.json(emptyCart()), contextId, shouldSet);
    }

    let upstreamPath = "cart";
    let upstreamMethod = "GET";
    let upstreamBody: unknown;
    let requestedQuantity: number | undefined;

    if (validation.operation === "add-item") {
      const parsed = validateAddCartItemInput(await readJsonBody(request));
      if (!parsed.data) return errorResponse("INVALID_QUANTITY", "Dữ liệu thêm giỏ chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
      upstreamPath = "cart/items";
      upstreamMethod = "POST";
      upstreamBody = { sku_code: parsed.data.skuId, quantity: parsed.data.quantity };
    } else if (validation.operation === "update-item") {
      const parsed = validateUpdateCartItemInput(await readJsonBody(request));
      if (!parsed.data) return errorResponse("INVALID_QUANTITY", "Số lượng chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
      upstreamPath = `cart/items/${encodeURIComponent(validation.itemId!)}`;
      upstreamMethod = "PUT";
      upstreamBody = parsed.data;
      requestedQuantity = parsed.data.quantity;
    } else if (validation.operation === "remove-item") {
      upstreamPath = `cart/items/${encodeURIComponent(validation.itemId!)}`;
      upstreamMethod = "DELETE";
    }

    const { response, payload } = await callUpstream(request, upstreamPath, upstreamMethod, contextId, validation.mockScenario, upstreamBody);
    if (!response.ok) return upstreamError(payload, response.status, contextId, shouldSet);
    let cart = mapCustomerCoreCart(payload as CustomerCoreCart);
    if (process.env.NODE_ENV !== "production" && validation.operation === "update-item" && requestedQuantity !== undefined) {
      cart = withCartItemQuantity(cart, validation.itemId!, requestedQuantity);
    }
    return withContextCookie(NextResponse.json(cart, { status: response.status }), contextId, shouldSet);
  } catch {
    return errorResponse(
      "CART_UPSTREAM_UNAVAILABLE",
      "Dịch vụ giỏ hàng đang tạm gián đoạn. Hãy kiểm tra Mockoon customer API.",
      503,
      contextId,
      shouldSet
    );
  }
}
