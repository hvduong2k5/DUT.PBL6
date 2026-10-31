import { hasTrustedOrigin } from "@/lib/auth/request-origin";
import {
  isValidIdempotencyKey,
  normalizePublicError,
  type CheckoutOperation,
  type OrderQuote,
  type PublicErrorBody
} from "@/lib/api-contract";
import { authorizeCustomerCapability, customerAccessFailure } from "@/lib/auth/customer-access-server";
import { applyTrustedAuthorization, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import {
  parseMvpCheckoutRoute,
  validateMvpAcceptRequest,
  validateMvpQuoteRequest
} from "@/lib/checkout/validation";
import { NextRequest } from "next/server";

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu;

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

function upstreamHeaders(request: NextRequest, scenario?: string, hasBody = false, idempotencyKey?: string): Headers {
  const headers = new Headers({ Accept: "application/json" });
  applyTrustedAuthorization(headers, request);
  if (hasBody) headers.set("Content-Type", "application/json");
  if (idempotencyKey) headers.set("Idempotency-Key", idempotencyKey);
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

async function callUpstream(
  request: NextRequest,
  path: string,
  method: string,
  scenario?: string,
  body?: unknown,
  idempotencyKey?: string
) {
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

function responseHeaders(upstream: Response): Headers {
  const headers = new Headers();
  for (const name of ["cache-control", "link", "location", "retry-after", "x-request-id"]) {
    const value = upstream.headers.get(name);
    if (value) headers.set(name, value);
  }
  return headers;
}

function relayUpstreamError(payload: unknown, upstream: Response): Response {
  const error = normalizePublicError(payload, {
    code: "CHECKOUT_UPSTREAM_ERROR",
    message: "Không thể tiếp tục Checkout lúc này.",
    requestId: upstream.headers.get("x-request-id") ?? "BFF-CHECKOUT-UPSTREAM"
  });
  return Response.json(error, { status: upstream.status, headers: responseHeaders(upstream) });
}

function isIsoDateTime(value: unknown): value is string {
  return typeof value === "string" && value.length > 0 && !Number.isNaN(Date.parse(value));
}

function isOrderAddress(value: unknown): boolean {
  return isRecord(value)
    && [value.recipient_name, value.phone, value.street, value.ward_code, value.province_code]
      .every((field) => typeof field === "string" && field.length > 0);
}

function isOrderItem(value: unknown): boolean {
  return isRecord(value)
    && typeof value.sku_code === "string"
    && value.sku_code.length > 0
    && Number.isSafeInteger(value.quantity)
    && Number(value.quantity) >= 1;
}

function isOrderQuote(value: unknown): value is OrderQuote {
  return isRecord(value)
    && typeof value.catalog_snapshot_id === "string"
    && value.catalog_snapshot_id.length > 0
    && isIsoDateTime(value.catalog_expires_at)
    && typeof value.shipping_quote_reference === "string"
    && value.shipping_quote_reference.length > 0
    && typeof value.quote_id === "string"
    && Number.isSafeInteger(value.cart_revision)
    && Number(value.cart_revision) >= 0
    && (value.payment_method === "VIETQR" || value.payment_method === "COD")
    && Array.isArray(value.items)
    && value.items.every(isOrderItem)
    && isOrderAddress(value.address)
    && Number.isSafeInteger(value.subtotal)
    && Number(value.subtotal) >= 0
    && Number.isSafeInteger(value.shipping_fee)
    && Number(value.shipping_fee) >= 0
    && Number.isSafeInteger(value.final_amount)
    && Number(value.final_amount) >= 0
    && isIsoDateTime(value.expires_at)
    && isIsoDateTime(value.shipping_expires_at);
}

function isCheckoutOperation(value: unknown): value is CheckoutOperation {
  return isRecord(value)
    && typeof value.operation_id === "string"
    && UUID_PATTERN.test(value.operation_id)
    && ["ACCEPTED", "PROCESSING", "WAITING_RETRY", "COMPENSATING", "SUCCEEDED", "FAILED", "MANUAL_REVIEW"].includes(String(value.status))
    && ["VALIDATING", "ACQUIRING", "PLACING", "COMPENSATING", "DONE"].includes(String(value.stage))
    && ((typeof value.order_id === "string" && UUID_PATTERN.test(value.order_id)) || value.order_id === null)
    && isIsoDateTime(value.accepted_at)
    && isIsoDateTime(value.updated_at);
}

export async function handleCheckoutRequest(request: NextRequest, path: string[]): Promise<Response> {
  const route = parseMvpCheckoutRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation) {
    const notFound = route.error?.field === "path";
    return publicErrorResponse(
      notFound ? 404 : 400,
      notFound ? "NOT_FOUND" : "INVALID_REQUEST",
      route.error?.message ?? "Yêu cầu không hợp lệ.",
      "BFF-CHECKOUT-ROUTE",
      route.error ? { errors: [route.error] } : undefined
    );
  }
  if (request.method !== "GET" && !hasTrustedOrigin(request)) {
    return publicErrorResponse(403, "ORIGIN_NOT_ALLOWED", "Request origin is not allowed.", "BFF-CHECKOUT-ORIGIN");
  }
  if (process.env.NODE_ENV === "production" && route.mockScenario) {
    return publicErrorResponse(400, "INVALID_REQUEST", "Mock scenario không khả dụng trong production.", "BFF-CHECKOUT-SCENARIO");
  }

  const accessToken = getTrustedAccessToken(request);
  try {
    await authorizeCustomerCapability(accessToken, "CHECKOUT_CREATE");
  } catch (cause) {
    const failure = customerAccessFailure(cause);
    return publicErrorResponse(failure.status, failure.code, failure.message, "BFF-CHECKOUT-ACCESS");
  }

  try {
    if (route.operation === "get-operation") {
      const result = await callUpstream(request, `checkout-operations/${encodeURIComponent(route.operationId!)}`, "GET", route.mockScenario);
      if (!result.response.ok) return relayUpstreamError(result.payload, result.response);
      if (result.response.status !== 200) return publicErrorResponse(502, "INVALID_CHECKOUT_STATUS", "Checkout operation trả status ngoài contract.", "BFF-CHECKOUT-OPERATION");
      return isCheckoutOperation(result.payload)
        ? Response.json(result.payload, { status: result.response.status, headers: responseHeaders(result.response) })
        : publicErrorResponse(502, "INVALID_CHECKOUT_OPERATION", "Checkout operation không đúng contract.", "BFF-CHECKOUT-OPERATION");
    }

    const body = await readJsonBody(request);
    if (route.operation === "create-quote") {
      if (isRecord(body)) {
        const hasAddressId = Object.hasOwn(body, "shipping_address_id");
        const hasAddress = Object.hasOwn(body, "address");
        if (!hasAddressId && !hasAddress) {
          return publicErrorResponse(422, "ADDRESS_REQUIRED", "Checkout quote cần một địa chỉ giao hàng.", "BFF-CHECKOUT-QUOTE-ADDRESS");
        }
        if (hasAddressId && hasAddress) {
          return publicErrorResponse(400, "AMBIGUOUS_ADDRESS", "Chỉ được gửi shipping_address_id hoặc address.", "BFF-CHECKOUT-QUOTE-ADDRESS");
        }
      }
      const parsed = validateMvpQuoteRequest(body);
      if (!parsed.data) return publicErrorResponse(400, "INVALID_REQUEST", "Yêu cầu tạo quote chưa hợp lệ.", "BFF-CHECKOUT-QUOTE", { errors: parsed.errors });
      if (parsed.data.payment_method === "COD" && !MVP_FEATURE_GATES.cod) {
        return publicErrorResponse(403, "FEATURE_NOT_ENABLED", "COD chưa được bật cho Web MVP hiện tại.", "BFF-CHECKOUT-COD-GATE");
      }
      if (parsed.data.voucher_code && !MVP_FEATURE_GATES.voucher) {
        return publicErrorResponse(403, "FEATURE_NOT_ENABLED", "Voucher chưa được bật cho Web MVP hiện tại.", "BFF-CHECKOUT-VOUCHER-GATE");
      }
      const result = await callUpstream(request, "checkout/quote", "POST", route.mockScenario, parsed.data);
      if (!result.response.ok) return relayUpstreamError(result.payload, result.response);
      if (result.response.status !== 200) return publicErrorResponse(502, "INVALID_CHECKOUT_STATUS", "Checkout quote trả status ngoài contract.", "BFF-CHECKOUT-QUOTE");
      return isOrderQuote(result.payload)
        ? Response.json(result.payload, { status: result.response.status, headers: responseHeaders(result.response) })
        : publicErrorResponse(502, "INVALID_CHECKOUT_QUOTE", "Checkout quote không đúng contract.", "BFF-CHECKOUT-QUOTE");
    }

    const idempotencyKey = request.headers.get("idempotency-key");
    if (!idempotencyKey || !isValidIdempotencyKey(idempotencyKey)) {
      return publicErrorResponse(400, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key không hợp lệ.", "BFF-CHECKOUT-IDEMPOTENCY");
    }
    const parsed = validateMvpAcceptRequest(body);
    if (!parsed.data) return publicErrorResponse(400, "INVALID_REQUEST", "Yêu cầu accept Checkout chưa hợp lệ.", "BFF-CHECKOUT-ACCEPT", { errors: parsed.errors });
    if (parsed.data.payment_method === "COD" && !MVP_FEATURE_GATES.cod) {
      return publicErrorResponse(403, "FEATURE_NOT_ENABLED", "COD chưa được bật cho Web MVP hiện tại.", "BFF-CHECKOUT-COD-GATE");
    }
    const result = await callUpstream(request, "checkout", "POST", route.mockScenario, parsed.data, idempotencyKey);
    if (!result.response.ok) return relayUpstreamError(result.payload, result.response);
    if (result.response.status !== 201 && result.response.status !== 202) {
      return publicErrorResponse(502, "INVALID_CHECKOUT_STATUS", "Checkout accept trả status ngoài contract.", "BFF-CHECKOUT-ACCEPT");
    }
    return isCheckoutOperation(result.payload)
      ? Response.json(result.payload, { status: result.response.status, headers: responseHeaders(result.response) })
      : publicErrorResponse(502, "INVALID_CHECKOUT_OPERATION", "Checkout operation không đúng contract.", "BFF-CHECKOUT-ACCEPT");
  } catch {
    return publicErrorResponse(503, "CHECKOUT_UPSTREAM_UNAVAILABLE", "Dịch vụ Checkout đang tạm gián đoạn. Hãy kiểm tra API Gateway.", "BFF-CHECKOUT-UPSTREAM");
  }
}
