import { randomUUID } from "node:crypto";
import type { Cart, CartItem, CartNotice } from "@/lib/cart/types";
import { parseCartRoute, validateAddCartItemInput, validateUpdateCartItemInput } from "@/lib/cart/validation";
import { NextRequest, NextResponse } from "next/server";

const CART_COOKIE = "oma_cart_context";
const CART_CONTEXT_PATTERN = /^[0-9a-f-]{36}$/u;
const MOCK_CONTEXTS = new Set(["cart-prefilled", "cart-empty", "price-changed", "cart-unavailable"]);

interface CartLineRecord {
  id: string;
  contextId: string;
  skuId: string;
  quantity: number;
  unitPriceVndAtAddition: number;
  productId: string;
  productSlug: string;
  productName: string;
  imageUrl: string;
  imageAlt: string;
  skuLabel: string;
  weightGrams: number;
  flavor?: string | null;
  packageType: string;
  createdAt: string;
}

interface SkuProjection {
  skuId: string;
  productId: string;
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

class UpstreamHttpError extends Error {
  constructor(readonly status: number) {
    super(`Cart upstream returned ${status}`);
  }
}

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

function getContext(request: NextRequest, scenario?: string) {
  const existing = request.cookies.get(CART_COOKIE)?.value;
  const validExisting = existing && CART_CONTEXT_PATTERN.test(existing) ? existing : undefined;
  const contextId = validExisting ?? randomUUID();
  const mockContextId = process.env.NODE_ENV !== "production" && scenario && MOCK_CONTEXTS.has(scenario)
    ? `mock-${scenario}`
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

async function upstreamJson<T>(path: string, scenario?: string, init: RequestInit = {}): Promise<T> {
  const base = (process.env.CART_UPSTREAM_URL ?? "http://127.0.0.1:4014/api/v1").replace(/\/$/u, "");
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body) headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  const response = await fetch(`${base}/${path}`, { ...init, headers, cache: "no-store", redirect: "manual" });
  if (!response.ok) throw new UpstreamHttpError(response.status);
  if (response.status === 204) return undefined as T;
  return await response.json() as T;
}

async function getLines(contextId: string, scenario?: string): Promise<CartLineRecord[]> {
  const params = new URLSearchParams({ contextId_eq: contextId, sort: "createdAt", order: "asc" });
  return upstreamJson<CartLineRecord[]>(`internal/cart-lines?${params}`, scenario);
}

async function getProjections(scenario?: string): Promise<SkuProjection[]> {
  return upstreamJson<SkuProjection[]>("internal/cart-sku-projections", scenario);
}

function composeCart(lines: CartLineRecord[], projections: SkuProjection[]): Cart {
  const projectionBySku = new Map(projections.map((projection) => [projection.skuId, projection]));
  const notices: CartNotice[] = [];
  const items: CartItem[] = lines.map((line) => {
    const current = projectionBySku.get(line.skuId);
    const isAvailable = current?.isAvailable ?? false;
    const currentPrice = current?.unitPriceVnd ?? line.unitPriceVndAtAddition;
    const priceChanged = Boolean(current && currentPrice !== line.unitPriceVndAtAddition);
    if (priceChanged) {
      notices.push({ code: "PRICE_CHANGED", itemId: line.id, message: `Giá ${line.skuLabel} đã được cập nhật theo giá bán hiện tại.` });
    }
    if (!isAvailable) {
      notices.push({ code: "SKU_UNAVAILABLE", itemId: line.id, message: current?.unavailableReason || "Quy cách này hiện không còn khả dụng." });
    }
    return {
      itemId: line.id,
      productId: current?.productId ?? line.productId,
      productSlug: current?.productSlug ?? line.productSlug,
      productName: current?.productName ?? line.productName,
      imageUrl: current?.imageUrl ?? line.imageUrl,
      imageAlt: current?.imageAlt ?? line.imageAlt,
      skuId: line.skuId,
      skuLabel: current?.skuLabel ?? line.skuLabel,
      weightGrams: current?.weightGrams ?? line.weightGrams,
      flavor: current?.flavor ?? line.flavor,
      packageType: current?.packageType ?? line.packageType,
      unitPriceVnd: currentPrice,
      previousUnitPriceVnd: priceChanged ? line.unitPriceVndAtAddition : null,
      priceChanged,
      quantity: line.quantity,
      lineSubtotalVnd: isAvailable ? currentPrice * line.quantity : null,
      isAvailable,
      unavailableReason: isAvailable ? null : current?.unavailableReason || "Quy cách này hiện không còn khả dụng."
    };
  });
  return {
    items,
    itemCount: items.reduce((total, item) => total + item.quantity, 0),
    subtotalVnd: items.reduce((total, item) => total + (item.lineSubtotalVnd ?? 0), 0),
    hasBlockingIssues: items.some((item) => !item.isAvailable),
    notices,
    updatedAt: new Date().toISOString()
  };
}

async function currentCart(contextId: string, scenario?: string): Promise<Cart> {
  const [lines, projections] = await Promise.all([getLines(contextId, scenario), getProjections(scenario)]);
  return composeCart(lines, projections);
}

function assertPurchasable(projection: SkuProjection | undefined, quantity: number): { code: string; message: string; status: number } | undefined {
  if (!projection) return { code: "SKU_NOT_FOUND", message: "Không tìm thấy quy cách sản phẩm có thể mua.", status: 404 };
  if (!projection.isAvailable) return { code: "SKU_UNAVAILABLE", message: projection.unavailableReason || "Quy cách này hiện không khả dụng.", status: 409 };
  if (quantity > projection.maxPurchasableQuantity) return { code: "QUANTITY_UNAVAILABLE", message: "Số lượng yêu cầu hiện không thể đáp ứng. Vui lòng giảm số lượng và thử lại.", status: 409 };
  return undefined;
}

export async function handleCartRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const validation = parseCartRoute(request.method, path, request.nextUrl.searchParams);
  const scenario = validation.mockScenario;
  const { contextId, mockContextId, shouldSet } = getContext(request, scenario);

  if (validation.error || !validation.operation) {
    const notFound = validation.error?.field === "path";
    return errorResponse(notFound ? "NOT_FOUND" : "INVALID_REQUEST", validation.error?.message ?? "Yêu cầu không hợp lệ.", notFound ? 404 : 400, contextId, shouldSet, validation.error ? [validation.error] : []);
  }
  if (process.env.NODE_ENV === "production" && scenario) {
    return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400, contextId, shouldSet);
  }

  try {
    if (validation.operation === "get-cart") {
      return withContextCookie(NextResponse.json(await currentCart(mockContextId, scenario)), contextId, shouldSet);
    }

    if (validation.operation === "add-item") {
      const parsed = validateAddCartItemInput(await readJsonBody(request));
      if (!parsed.data) return errorResponse("INVALID_QUANTITY", "Dữ liệu thêm giỏ chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
      const [lines, projections] = await Promise.all([getLines(mockContextId, scenario), getProjections(scenario)]);
      const projection = projections.find((item) => item.skuId === parsed.data!.skuId);
      const existing = lines.find((line) => line.skuId === parsed.data!.skuId);
      const nextQuantity = (existing?.quantity ?? 0) + parsed.data.quantity;
      const blocked = assertPurchasable(projection, nextQuantity);
      if (blocked) return errorResponse(blocked.code, blocked.message, blocked.status, contextId, shouldSet);
      if (existing) {
        await upstreamJson(`internal/cart-lines/${encodeURIComponent(existing.id)}`, scenario, { method: "PATCH", body: JSON.stringify({ quantity: nextQuantity }) });
      } else {
        const now = new Date().toISOString();
        const line: CartLineRecord = {
          id: randomUUID(), contextId: mockContextId, skuId: projection!.skuId, quantity: parsed.data.quantity,
          unitPriceVndAtAddition: projection!.unitPriceVnd, productId: projection!.productId,
          productSlug: projection!.productSlug, productName: projection!.productName,
          imageUrl: projection!.imageUrl, imageAlt: projection!.imageAlt, skuLabel: projection!.skuLabel,
          weightGrams: projection!.weightGrams, flavor: projection!.flavor, packageType: projection!.packageType,
          createdAt: now
        };
        await upstreamJson("internal/cart-lines", scenario, { method: "POST", body: JSON.stringify(line) });
      }
      return withContextCookie(NextResponse.json(await currentCart(mockContextId, scenario), { status: existing ? 200 : 201 }), contextId, shouldSet);
    }

    const lines = await getLines(mockContextId, scenario);
    const line = lines.find((item) => item.id === validation.itemId && item.contextId === mockContextId);
    if (!line) return errorResponse("CART_ITEM_NOT_FOUND", "Dòng giỏ hàng không còn tồn tại. Vui lòng tải lại giỏ.", 404, contextId, shouldSet);

    if (validation.operation === "update-item") {
      const parsed = validateUpdateCartItemInput(await readJsonBody(request));
      if (!parsed.data) return errorResponse("INVALID_QUANTITY", "Số lượng chưa hợp lệ.", 422, contextId, shouldSet, parsed.errors);
      const projections = await getProjections(scenario);
      const blocked = assertPurchasable(projections.find((item) => item.skuId === line.skuId), parsed.data.quantity);
      if (blocked) return errorResponse(blocked.code, blocked.message, blocked.status, contextId, shouldSet);
      await upstreamJson(`internal/cart-lines/${encodeURIComponent(line.id)}`, scenario, { method: "PATCH", body: JSON.stringify({ quantity: parsed.data.quantity }) });
    } else {
      await upstreamJson(`internal/cart-lines/${encodeURIComponent(line.id)}`, scenario, { method: "DELETE" });
    }
    return withContextCookie(NextResponse.json(await currentCart(mockContextId, scenario)), contextId, shouldSet);
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) {
      const status = cause.status >= 500 ? 500 : cause.status;
      return errorResponse("CART_ERROR", "Không thể cập nhật giỏ hàng lúc này. Vui lòng thử lại.", status, contextId, shouldSet);
    }
    const message = process.env.NODE_ENV === "production"
      ? "Dịch vụ giỏ hàng đang tạm gián đoạn. Vui lòng thử lại sau."
      : "Cart Mockoon chưa chạy trên port 4014. Hãy chạy npm run dev:cart từ apps/web-user.";
    return errorResponse("CART_UPSTREAM_UNAVAILABLE", message, 503, contextId, shouldSet);
  }
}
