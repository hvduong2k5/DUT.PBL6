import {
  mapCustomerCoreCancellation,
  mapCustomerCoreOrderDetail,
  mapCustomerCoreOrderListItem,
  type CustomerCoreOrderDetail,
  type CustomerCoreOrderListItem,
  type CustomerCoreTracking
} from "@/lib/orders/customer-core";
import type { GuestChallenge, GuestVerification, OrderListResponse } from "@/lib/orders/types";
import { parseOrderRoute, validateCancelOrder, validateGuestChallenge, validateGuestOtp } from "@/lib/orders/validation";
import { fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import { GUEST_ORDER_ACCESS_COOKIE, decodeGuestOrderAccess, encodeGuestOrderAccess } from "@/lib/orders/guest-access";
import { NextRequest, NextResponse } from "next/server";

function errorResponse(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) {
  return NextResponse.json({ code, message, errors, requestId: `BFF-ORDER-${code}` }, { status });
}

function hasTrustedOrigin(request: NextRequest) {
  const origin = request.headers.get("origin");
  if (!origin) return true;
  try {
    const source = new URL(origin);
    return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host;
  } catch { return false; }
}

async function readJson(request: NextRequest): Promise<unknown> {
  try { return await request.json(); } catch { return undefined; }
}

function upstreamHeaders(request: NextRequest, scenario?: string, hasBody = false, idempotencyKey?: string) {
  const headers = new Headers({ Accept: "application/json" });
  const cookie = request.headers.get("cookie");
  const authorization = request.headers.get("authorization");
  const accessToken = request.cookies.get("oma_access_token")?.value;
  if (cookie) headers.set("Cookie", cookie);
  if (authorization) headers.set("Authorization", authorization);
  else if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (hasBody) headers.set("Content-Type", "application/json");
  if (idempotencyKey) headers.set("X-Idempotency-Key", idempotencyKey);
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

async function upstream(request: NextRequest, path: string, method: string, scenario?: string, body?: unknown, idempotencyKey?: string) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/${path}`, {
    method,
    headers: upstreamHeaders(request, scenario, body !== undefined, idempotencyKey),
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
  const payload = contentType.includes("application/json") ? await response.json() as Record<string, unknown> : {};
  return { response, payload };
}

function text(value: unknown, fallback = "") { return typeof value === "string" ? value : fallback; }
function dateTime(value: unknown) { return typeof value === "string" && !Number.isNaN(Date.parse(value)) ? value : new Date().toISOString(); }
function positiveInteger(value: unknown, fallback: number) { return Number.isInteger(value) && Number(value) > 0 ? Number(value) : fallback; }

function relayError(payload: unknown, status: number) {
  return NextResponse.json(payload && typeof payload === "object" ? payload : { error_code: "ERR_ORDER", user_message: "Không thể xử lý đơn hàng lúc này." }, { status });
}

export async function handleOrderRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parseOrderRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation) return errorResponse(route.error?.field === "path" ? "NOT_FOUND" : "INVALID_REQUEST", route.error?.message ?? "Yêu cầu Order không hợp lệ.", route.error?.field === "path" ? 404 : 400, route.error ? [route.error] : []);
  if (process.env.NODE_ENV === "production" && route.mockScenario) return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400);
  if (request.method !== "GET" && !hasTrustedOrigin(request)) return errorResponse("ORIGIN_NOT_ALLOWED", "Request origin không được phép.", 403);

  if (route.operation === "list" || route.operation === "cancel" || route.operation === "detail") {
    const accessToken = request.cookies.get("oma_access_token")?.value;
    const guestAccess = decodeGuestOrderAccess(request.cookies.get(GUEST_ORDER_ACCESS_COOKIE)?.value);
    if (route.operation === "detail" && !accessToken && guestAccess) {
      if (guestAccess.orderId !== route.orderId) return errorResponse("GUEST_ORDER_FORBIDDEN", "Quyền Guest chỉ hợp lệ cho đúng Order đã được xác minh.", 403);
    } else {
      if (!accessToken) return errorResponse("AUTH_REQUIRED", "Vui lòng đăng nhập hoặc xác minh quyền sở hữu Order.", 401);
      try {
        const projection = await fetchCustomerCapabilities(accessToken);
        if (!projection.capabilities.includes("ORDER_HISTORY_VIEW")) return errorResponse("CAPABILITY_FORBIDDEN", "Tài khoản không có quyền xem hoặc quản lý Order.", 403);
      } catch {
        return errorResponse("CAPABILITY_UNAVAILABLE", "Chưa thể kiểm tra quyền Customer lúc này.", 503);
      }
    }
  }

  try {
    if (route.operation === "guest-challenge") {
      const parsed = validateGuestChallenge(await readJson(request));
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Thông tin tra cứu chưa hợp lệ.", 422, parsed.errors);
      const result = await extensionUpstream(request, "orders/guest-access/challenges", route.mockScenario, parsed.data);
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      const challenge: GuestChallenge = {
        challengeId: text(result.payload.challengeId),
        maskedDestination: text(result.payload.maskedDestination),
        expiresAt: dateTime(result.payload.expiresAt),
        resendAfterSeconds: Math.min(300, positiveInteger(result.payload.resendAfterSeconds, 60)),
        message: text(result.payload.message)
      };
      return NextResponse.json(challenge, { status: 202 });
    }

    if (route.operation === "guest-verify" && route.challengeId) {
      const parsed = validateGuestOtp(await readJson(request));
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Mã xác minh chưa hợp lệ.", 422, parsed.errors);
      const result = await extensionUpstream(request, `orders/guest-access/challenges/${encodeURIComponent(route.challengeId)}/verify`, route.mockScenario, parsed.data);
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      const verification: GuestVerification = {
        orderId: text(result.payload.orderId),
        orderNumber: text(result.payload.orderNumber),
        orderPath: text(result.payload.orderPath),
        accessExpiresAt: dateTime(result.payload.accessExpiresAt)
      };
      const upstreamAccessExpiresAt = Date.parse(verification.accessExpiresAt);
      if (!verification.orderId || !verification.orderNumber || !Number.isFinite(upstreamAccessExpiresAt) || upstreamAccessExpiresAt <= Date.now()) {
        return errorResponse("GUEST_ACCESS_INVALID", "Phản hồi xác minh Guest Order không hợp lệ.", 502);
      }
      const accessExpiresAt = Math.min(upstreamAccessExpiresAt, Date.now() + 30 * 60 * 1000);
      verification.accessExpiresAt = new Date(accessExpiresAt).toISOString();
      const response = NextResponse.json(verification);
      response.cookies.set(GUEST_ORDER_ACCESS_COOKIE, encodeGuestOrderAccess({
        orderId: verification.orderId,
        orderNumber: verification.orderNumber,
        accessExpiresAt
      }), {
        path: "/api/orders",
        httpOnly: true,
        sameSite: "lax",
        secure: process.env.NODE_ENV === "production",
        maxAge: Math.max(1, Math.floor((accessExpiresAt - Date.now()) / 1000))
      });
      return response;
    }

    if (route.operation === "list" && route.filters) {
      const params = new URLSearchParams({ page: String(route.filters.page), page_size: String(route.filters.pageSize) });
      if (route.filters.status !== "ALL") params.set("status", route.filters.status === "SHIPPED" ? "SHIPPING" : route.filters.status);
      const { response, payload } = await upstream(request, `orders?${params}`, "GET", route.mockScenario);
      if (!response.ok) return relayError(payload, response.status);
      const source = payload as { orders?: CustomerCoreOrderListItem[]; total?: number; page?: number; page_size?: number };
      const query = route.filters.q.toLocaleLowerCase("vi");
      const items = (source.orders ?? []).map(mapCustomerCoreOrderListItem).filter((item) => (
        (!query || item.orderNumber.toLocaleLowerCase("vi").includes(query))
        && (!route.filters!.year || new Date(item.placedAt).getFullYear() === route.filters!.year)
      ));
      const hasLocalFilter = Boolean(query || route.filters.year);
      const totalItems = hasLocalFilter ? items.length : source.total ?? items.length;
      const result: OrderListResponse = {
        items,
        page: source.page ?? route.filters.page,
        pageSize: source.page_size ?? route.filters.pageSize,
        totalItems,
        totalPages: Math.max(1, Math.ceil(totalItems / (source.page_size ?? route.filters.pageSize))),
        customer: { displayName: "Thành viên Tri Kỷ", email: "" }
      };
      return NextResponse.json(result);
    }

    if (route.operation === "detail" && route.orderId) {
      const detailResult = await upstream(request, `orders/${encodeURIComponent(route.orderId)}`, "GET", route.mockScenario);
      if (!detailResult.response.ok) return relayError(detailResult.payload, detailResult.response.status);
      const trackingResult = await upstream(request, `orders/${encodeURIComponent(route.orderId)}/tracking`, "GET", route.mockScenario);
      const tracking = trackingResult.response.ok ? trackingResult.payload as CustomerCoreTracking : undefined;
      const detail = mapCustomerCoreOrderDetail(detailResult.payload as CustomerCoreOrderDetail, tracking);
      if (detail.orderId !== route.orderId) return errorResponse("ORDER_ID_MISMATCH", "Dữ liệu Order không khớp với Order đã được yêu cầu.", 502);
      return NextResponse.json(detail);
    }

    if (route.operation === "cancel" && route.orderId) {
      const parsed = validateCancelOrder(await readJson(request));
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Thông tin hủy Order chưa hợp lệ.", 422, parsed.errors);
      const reasonLabels = { CHANGED_MIND: "Thay đổi nhu cầu", WRONG_INFORMATION: "Thông tin đặt hàng chưa đúng", OTHER: "Lý do khác" };
      const { response, payload } = await upstream(request, `orders/${encodeURIComponent(route.orderId)}/cancel`, "POST", route.mockScenario, { cancel_reason: parsed.data.note || reasonLabels[parsed.data.reasonCode] }, parsed.data.idempotencyKey);
      if (!response.ok) return relayError(payload, response.status);
      return NextResponse.json(mapCustomerCoreCancellation(route.orderId));
    }

    return errorResponse("NOT_FOUND", "Order route is not available.", 404);
  } catch {
    return errorResponse("ORDER_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ Order. Hãy kiểm tra Mockoon customer API.", 503);
  }
}
