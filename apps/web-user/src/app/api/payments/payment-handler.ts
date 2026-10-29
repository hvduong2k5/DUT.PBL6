import { normalizePublicError, type PublicErrorBody } from "@/lib/api-contract";
import { fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import { isPaymentInstructions, isPaymentStatus } from "@/lib/payment/mvp";
import { parsePaymentRoute } from "@/lib/payment/validation";
import { NextRequest, NextResponse } from "next/server";

function publicErrorResponse(status: number, code: string, message: string, requestId: string) {
  const body: PublicErrorBody = { code, message, requestId };
  return NextResponse.json(body, { status });
}

function bearerToken(request: NextRequest): string | undefined {
  const cookieToken = request.cookies.get("oma_access_token")?.value;
  if (cookieToken) return cookieToken;
  const authorization = request.headers.get("authorization");
  const match = authorization?.match(/^Bearer\s+(.+)$/iu);
  return match?.[1];
}

function upstreamHeaders(request: NextRequest, scenario?: string) {
  const headers = new Headers({ Accept: "application/json" });
  const authorization = request.headers.get("authorization");
  const accessToken = request.cookies.get("oma_access_token")?.value;
  const cookie = request.headers.get("cookie");
  if (authorization) headers.set("Authorization", authorization);
  else if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (cookie) headers.set("Cookie", cookie);
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

async function upstream(request: NextRequest, path: string, scenario?: string) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/${path}`, {
    method: "GET",
    headers: upstreamHeaders(request, scenario),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

function responseHeaders(upstreamResponse: Response): Headers {
  const headers = new Headers({ "Cache-Control": "no-store" });
  for (const name of ["retry-after", "x-request-id"]) {
    const value = upstreamResponse.headers.get(name);
    if (value) headers.set(name, value);
  }
  return headers;
}

function relayError(payload: unknown, upstreamResponse: Response) {
  const error = normalizePublicError(payload, {
    code: "PAYMENT_UPSTREAM_ERROR",
    message: "Không thể đọc Payment lúc này.",
    requestId: upstreamResponse.headers.get("x-request-id") ?? "BFF-PAYMENT-UPSTREAM"
  });
  return NextResponse.json(error, { status: upstreamResponse.status, headers: responseHeaders(upstreamResponse) });
}

export async function handlePaymentRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parsePaymentRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation || !route.orderId) {
    const notFound = route.error?.field === "path";
    return publicErrorResponse(
      notFound ? 404 : 400,
      notFound ? "NOT_FOUND" : "INVALID_REQUEST",
      route.error?.message ?? "Yêu cầu Payment không hợp lệ.",
      "BFF-PAYMENT-ROUTE"
    );
  }
  if (process.env.NODE_ENV === "production" && route.mockScenario) {
    return publicErrorResponse(400, "INVALID_REQUEST", "Mock scenario không khả dụng trong production.", "BFF-PAYMENT-SCENARIO");
  }

  const accessToken = bearerToken(request);
  if (!accessToken) {
    return publicErrorResponse(401, "AUTH_REQUIRED", "Vui lòng đăng nhập để xem Payment của Order.", "BFF-PAYMENT-AUTH");
  }
  try {
    const projection = await fetchCustomerCapabilities(accessToken);
    if (!projection.capabilities.includes("ORDER_HISTORY_VIEW")) {
      return publicErrorResponse(403, "CAPABILITY_FORBIDDEN", "Tài khoản không có quyền xem Payment của Order.", "BFF-PAYMENT-CAPABILITY");
    }
  } catch {
    return publicErrorResponse(503, "CAPABILITY_UNAVAILABLE", "Chưa thể kiểm tra quyền Customer lúc này.", "BFF-PAYMENT-CAPABILITY");
  }

  try {
    const upstreamPath = route.operation === "instructions"
      ? `orders/${encodeURIComponent(route.orderId)}/payment-instructions`
      : `orders/${encodeURIComponent(route.orderId)}/payment`;
    const result = await upstream(request, upstreamPath, route.mockScenario);
    if (!result.response.ok) return relayError(result.payload, result.response);
    if (result.response.status !== 200) {
      return publicErrorResponse(502, "INVALID_PAYMENT_STATUS", "Payment trả HTTP status ngoài contract.", "BFF-PAYMENT-CONTRACT");
    }
    if (route.operation === "instructions") {
      if (!isPaymentInstructions(result.payload) || result.payload.order_id !== route.orderId) {
        return publicErrorResponse(502, "INVALID_PAYMENT_INSTRUCTIONS", "Payment instructions không đúng public contract.", "BFF-PAYMENT-INSTRUCTIONS");
      }
    } else if (!isPaymentStatus(result.payload) || result.payload.order_id !== route.orderId) {
      return publicErrorResponse(502, "INVALID_PAYMENT_RESPONSE", "Payment status không đúng public contract.", "BFF-PAYMENT-STATUS");
    }
    return NextResponse.json(result.payload, { headers: responseHeaders(result.response) });
  } catch {
    return publicErrorResponse(503, "PAYMENT_UPSTREAM_UNAVAILABLE", "Dịch vụ Payment đang tạm gián đoạn. Hãy kiểm tra API Gateway.", "BFF-PAYMENT-UPSTREAM");
  }
}
