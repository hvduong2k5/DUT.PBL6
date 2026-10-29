import {
  isValidIdempotencyKey,
  normalizePublicError,
  parseDecimalVersion,
  type PublicErrorBody
} from "@/lib/api-contract";
import { authorizeCustomerCapability, customerAccessFailure } from "@/lib/auth/customer-access-server";
import { applyTrustedAuthorization, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import { GUEST_ORDER_ACCESS_COOKIE, decodeGuestOrderAccess, encodeGuestOrderAccess } from "@/lib/orders/guest-access";
import { isOrderCancellation, isOwnedOrder, isOwnedOrdersResponse } from "@/lib/orders/mvp";
import type { GuestChallenge, GuestVerification } from "@/lib/orders/types";
import { parseOrderRoute, validateGuestChallenge, validateGuestOtp, validateMvpCancelRequest } from "@/lib/orders/validation";
import { NextRequest, NextResponse } from "next/server";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function publicErrorResponse(
  status: number,
  code: string,
  message: string,
  requestId: string,
  details?: Record<string, unknown>
) {
  const body: PublicErrorBody = { code, message, requestId, ...(details ? { details } : {}) };
  return NextResponse.json(body, { status });
}

function hasTrustedOrigin(request: NextRequest): boolean {
  const origin = request.headers.get("origin");
  if (!origin) return true;
  try {
    const source = new URL(origin);
    return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host;
  } catch {
    return false;
  }
}

async function readJson(request: NextRequest): Promise<unknown> {
  try {
    return await request.json() as unknown;
  } catch {
    return undefined;
  }
}

function upstreamHeaders(
  request: NextRequest,
  scenario?: string,
  hasBody = false,
  idempotencyKey?: string,
  ifMatch?: string
) {
  const headers = new Headers({ Accept: "application/json" });
  applyTrustedAuthorization(headers, request);
  if (hasBody) headers.set("Content-Type", "application/json");
  if (idempotencyKey) headers.set("Idempotency-Key", idempotencyKey);
  if (ifMatch) headers.set("If-Match", ifMatch);
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

async function upstream(
  request: NextRequest,
  path: string,
  method: string,
  scenario?: string,
  body?: unknown,
  idempotencyKey?: string,
  ifMatch?: string
) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/${path}`, {
    method,
    headers: upstreamHeaders(request, scenario, body !== undefined, idempotencyKey, ifMatch),
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

async function extensionUpstream(request: NextRequest, path: string, scenario?: string, body?: unknown) {
  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/${path}`, {
    method: "POST",
    headers: upstreamHeaders(request, scenario, body !== undefined),
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

function responseHeaders(upstreamResponse: Response): Headers {
  const headers = new Headers();
  for (const name of ["cache-control", "location", "retry-after", "x-request-id"]) {
    const value = upstreamResponse.headers.get(name);
    if (value) headers.set(name, value);
  }
  return headers;
}

function relayError(payload: unknown, upstreamResponse: Response) {
  const error = normalizePublicError(payload, {
    code: "ORDER_UPSTREAM_ERROR",
    message: "Không thể xử lý Order lúc này.",
    requestId: upstreamResponse.headers.get("x-request-id") ?? "BFF-ORDER-UPSTREAM"
  });
  return NextResponse.json(error, { status: upstreamResponse.status, headers: responseHeaders(upstreamResponse) });
}

function guestExtensionEnabled(): boolean {
  return MVP_FEATURE_GATES.guestOrder;
}

function text(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function dateTime(value: unknown): string | undefined {
  return typeof value === "string" && !Number.isNaN(Date.parse(value)) ? value : undefined;
}

function positiveInteger(value: unknown): number | undefined {
  return Number.isSafeInteger(value) && Number(value) > 0 ? Number(value) : undefined;
}

export async function handleOrderRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parseOrderRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation) {
    const notFound = route.error?.field === "path";
    return publicErrorResponse(
      notFound ? 404 : 400,
      notFound ? "NOT_FOUND" : "INVALID_REQUEST",
      route.error?.message ?? "Yêu cầu Order không hợp lệ.",
      "BFF-ORDER-ROUTE",
      route.error ? { errors: [route.error] } : undefined
    );
  }
  if (process.env.NODE_ENV === "production" && route.mockScenario) {
    return publicErrorResponse(400, "INVALID_REQUEST", "Mock scenario không khả dụng trong production.", "BFF-ORDER-SCENARIO");
  }
  if (request.method !== "GET" && !hasTrustedOrigin(request)) {
    return publicErrorResponse(403, "ORIGIN_NOT_ALLOWED", "Request origin không được phép.", "BFF-ORDER-ORIGIN");
  }

  const isGuestExtension = route.operation === "guest-challenge" || route.operation === "guest-verify";
  if (isGuestExtension && !guestExtensionEnabled()) {
    return publicErrorResponse(403, "FEATURE_NOT_ENABLED", "Guest Order chưa được bật trong MVP contract hiện tại.", "BFF-ORDER-GUEST-GATE");
  }

  const accessToken = getTrustedAccessToken(request);
  const guestAccess = guestExtensionEnabled()
    ? decodeGuestOrderAccess(request.cookies.get(GUEST_ORDER_ACCESS_COOKIE)?.value)
    : undefined;
  const isGatedGuestDetail = route.operation === "detail"
    && !accessToken
    && guestAccess?.orderId === route.orderId;

  if (!isGuestExtension && !isGatedGuestDetail) {
    try {
      await authorizeCustomerCapability(accessToken, "ORDER_HISTORY_VIEW");
    } catch (cause) {
      const failure = customerAccessFailure(cause);
      return publicErrorResponse(failure.status, failure.code, failure.message, "BFF-ORDER-ACCESS");
    }
  }

  try {
    if (route.operation === "guest-challenge") {
      const parsed = validateGuestChallenge(await readJson(request));
      if (!parsed.data) return publicErrorResponse(422, "VALIDATION_ERROR", "Thông tin tra cứu chưa hợp lệ.", "BFF-ORDER-GUEST", { errors: parsed.errors });
      const result = await extensionUpstream(request, "orders/guest-access/challenges", route.mockScenario, parsed.data);
      if (!result.response.ok) return relayError(result.payload, result.response);
      if (!isRecord(result.payload)) return publicErrorResponse(502, "INVALID_GUEST_CHALLENGE", "Guest challenge không đúng contract mở rộng.", "BFF-ORDER-GUEST");
      const challengeId = text(result.payload.challengeId);
      const maskedDestination = text(result.payload.maskedDestination);
      const expiresAt = dateTime(result.payload.expiresAt);
      const resendAfterSeconds = positiveInteger(result.payload.resendAfterSeconds);
      if (!challengeId || !maskedDestination || !expiresAt || !resendAfterSeconds) {
        return publicErrorResponse(502, "INVALID_GUEST_CHALLENGE", "Guest challenge không đúng contract mở rộng.", "BFF-ORDER-GUEST");
      }
      const challenge: GuestChallenge = {
        challengeId,
        maskedDestination,
        expiresAt,
        resendAfterSeconds: Math.min(300, resendAfterSeconds),
        message: text(result.payload.message) ?? "Mã xác minh đã được gửi."
      };
      return NextResponse.json(challenge, { status: 202 });
    }

    if (route.operation === "guest-verify" && route.challengeId) {
      const parsed = validateGuestOtp(await readJson(request));
      if (!parsed.data) return publicErrorResponse(422, "VALIDATION_ERROR", "Mã xác minh chưa hợp lệ.", "BFF-ORDER-GUEST", { errors: parsed.errors });
      const result = await extensionUpstream(request, `orders/guest-access/challenges/${encodeURIComponent(route.challengeId)}/verify`, route.mockScenario, parsed.data);
      if (!result.response.ok) return relayError(result.payload, result.response);
      if (!isRecord(result.payload)) return publicErrorResponse(502, "GUEST_ACCESS_INVALID", "Phản hồi xác minh Guest Order không hợp lệ.", "BFF-ORDER-GUEST");
      const orderId = text(result.payload.orderId);
      const orderNumber = text(result.payload.orderNumber);
      const orderPath = text(result.payload.orderPath);
      const expiresAt = dateTime(result.payload.accessExpiresAt);
      if (!orderId || !orderNumber || !orderPath || !expiresAt || Date.parse(expiresAt) <= Date.now()) {
        return publicErrorResponse(502, "GUEST_ACCESS_INVALID", "Phản hồi xác minh Guest Order không hợp lệ.", "BFF-ORDER-GUEST");
      }
      const accessExpiresAt = Math.min(Date.parse(expiresAt), Date.now() + 30 * 60 * 1000);
      const verification: GuestVerification = { orderId, orderNumber, orderPath, accessExpiresAt: new Date(accessExpiresAt).toISOString() };
      const response = NextResponse.json(verification);
      response.cookies.set(GUEST_ORDER_ACCESS_COOKIE, encodeGuestOrderAccess({ orderId, orderNumber, accessExpiresAt }), {
        path: "/api/orders",
        httpOnly: true,
        sameSite: "lax",
        secure: process.env.NODE_ENV === "production",
        maxAge: Math.max(1, Math.floor((accessExpiresAt - Date.now()) / 1000))
      });
      return response;
    }

    if (route.operation === "list") {
      const result = await upstream(request, "orders", "GET", route.mockScenario);
      if (!result.response.ok) return relayError(result.payload, result.response);
      if (result.response.status !== 200 || !isOwnedOrdersResponse(result.payload)) {
        return publicErrorResponse(502, "INVALID_ORDERS_RESPONSE", "Danh sách Order không đúng public contract.", "BFF-ORDER-LIST");
      }
      return NextResponse.json(result.payload, { headers: responseHeaders(result.response) });
    }

    if (route.operation === "detail" && route.orderId) {
      const result = await upstream(request, `orders/${encodeURIComponent(route.orderId)}`, "GET", route.mockScenario);
      if (!result.response.ok) return relayError(result.payload, result.response);
      if (result.response.status !== 200 || !isOwnedOrder(result.payload) || result.payload.order_id !== route.orderId) {
        return publicErrorResponse(502, "INVALID_ORDER_RESPONSE", "Order detail không đúng public contract.", "BFF-ORDER-DETAIL");
      }
      return NextResponse.json(result.payload, { headers: responseHeaders(result.response) });
    }

    if (route.operation === "tracking" && route.orderId) {
      const result = await upstream(request, `orders/${encodeURIComponent(route.orderId)}/tracking`, "GET", route.mockScenario);
      if (!result.response.ok) return relayError(result.payload, result.response);
      if (result.response.status !== 200 || !isRecord(result.payload)) {
        return publicErrorResponse(502, "INVALID_TRACKING_RESPONSE", "Order tracking không đúng public contract.", "BFF-ORDER-TRACKING");
      }
      return NextResponse.json(result.payload, { headers: responseHeaders(result.response) });
    }

    if (route.operation === "cancel" && route.orderId) {
      const idempotencyKey = request.headers.get("idempotency-key");
      const expectedVersion = parseDecimalVersion(request.headers.get("if-match"));
      if (!idempotencyKey || !isValidIdempotencyKey(idempotencyKey)) {
        return publicErrorResponse(400, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key không hợp lệ.", "BFF-ORDER-CANCEL");
      }
      if (!expectedVersion) {
        return publicErrorResponse(400, "INVALID_ORDER_VERSION", "If-Match phải là Order version dương.", "BFF-ORDER-CANCEL");
      }
      const parsed = validateMvpCancelRequest(await readJson(request));
      if (!parsed.data) return publicErrorResponse(400, "INVALID_REQUEST", "Yêu cầu hủy Order chưa hợp lệ.", "BFF-ORDER-CANCEL", { errors: parsed.errors });
      const result = await upstream(
        request,
        `orders/${encodeURIComponent(route.orderId)}/cancel`,
        "POST",
        route.mockScenario,
        parsed.data,
        idempotencyKey,
        String(expectedVersion)
      );
      if (!result.response.ok) return relayError(result.payload, result.response);
      if (![200, 202].includes(result.response.status) || !isOrderCancellation(result.payload) || result.payload.order_id !== route.orderId) {
        return publicErrorResponse(502, "INVALID_CANCELLATION_RESPONSE", "Order cancellation không đúng public contract.", "BFF-ORDER-CANCEL");
      }
      return NextResponse.json(result.payload, { status: result.response.status, headers: responseHeaders(result.response) });
    }

    return publicErrorResponse(404, "NOT_FOUND", "Order route is not available.", "BFF-ORDER-ROUTE");
  } catch {
    return publicErrorResponse(503, "ORDER_UPSTREAM_UNAVAILABLE", "Dịch vụ Order đang tạm gián đoạn. Hãy kiểm tra API Gateway.", "BFF-ORDER-UPSTREAM");
  }
}
