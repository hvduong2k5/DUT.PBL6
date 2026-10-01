import { NextRequest, NextResponse } from "next/server";
import type { CustomerCoreOrderDetail } from "@/lib/orders/customer-core";
import {
  mapCustomerCoreReturnCreated,
  mapCustomerCoreReturnDetail,
  mapOrderToReturnEligibility,
  toCustomerCoreReturnRequest,
  type CustomerCoreReturnTicket
} from "@/lib/returns/customer-core";
import { parseReturnRoute, validateCreateReturnCase } from "@/lib/returns/validation";

function errorResponse(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) {
  return NextResponse.json({ code, message, errors, requestId: `BFF-RETURN-${code}` }, { status });
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

async function upstream(request: NextRequest, path: string, method: string, scenario?: string, body?: unknown, idempotencyKey?: string) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json" });
  const cookie = request.headers.get("cookie");
  const authorization = request.headers.get("authorization");
  const accessToken = request.cookies.get("oma_access_token")?.value;
  if (cookie) headers.set("Cookie", cookie);
  if (authorization) headers.set("Authorization", authorization);
  else if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (body !== undefined) headers.set("Content-Type", "application/json");
  if (idempotencyKey) headers.set("X-Idempotency-Key", idempotencyKey);
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

function relayError(payload: unknown, status: number) {
  return NextResponse.json(payload && typeof payload === "object" ? payload : {
    error_code: "ERR_RETURN",
    user_message: "Không thể xử lý yêu cầu đổi trả lúc này."
  }, { status });
}

export async function handleReturnRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parseReturnRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation) return errorResponse(route.error?.field === "path" ? "NOT_FOUND" : "INVALID_REQUEST", route.error?.message ?? "Yêu cầu hậu mãi không hợp lệ.", route.error?.field === "path" ? 404 : 400, route.error ? [route.error] : []);
  if (process.env.NODE_ENV === "production" && route.mockScenario) return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400);
  if (request.method !== "GET" && !hasTrustedOrigin(request)) return errorResponse("ORIGIN_NOT_ALLOWED", "Request origin không được phép.", 403);

  try {
    if (route.operation === "eligibility" && route.orderId) {
      const result = await upstream(request, `orders/${encodeURIComponent(route.orderId)}`, "GET", route.mockScenario);
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      return NextResponse.json(mapOrderToReturnEligibility(result.payload as CustomerCoreOrderDetail));
    }

    if (route.operation === "create") {
      const parsed = validateCreateReturnCase(await readJson(request));
      if (!parsed.data) return errorResponse("VALIDATION_ERROR", "Thông tin yêu cầu đổi trả chưa hợp lệ.", 422, parsed.errors);
      const result = await upstream(request, "returns", "POST", route.mockScenario, toCustomerCoreReturnRequest(parsed.data), parsed.data.idempotencyKey);
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      return NextResponse.json(mapCustomerCoreReturnCreated(result.payload as CustomerCoreReturnTicket), { status: 201 });
    }

    if (route.operation === "detail" && route.caseId) {
      const result = await upstream(request, `returns/${encodeURIComponent(route.caseId)}`, "GET", route.mockScenario);
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      return NextResponse.json(mapCustomerCoreReturnDetail(result.payload as CustomerCoreReturnTicket));
    }

    return errorResponse("RETURN_SUPPLEMENT_NOT_SUPPORTED", "API customer hiện chưa hỗ trợ bổ sung nội dung vào ticket đổi trả.", 501);
  } catch {
    return errorResponse("RETURN_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ đổi trả. Hãy kiểm tra Mockoon customer API.", 503);
  }
}
