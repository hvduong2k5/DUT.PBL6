import { PAYMENT_ACCESS_COOKIE, decodePaymentAccess, paymentAccessCookie, type PaymentAccessClaim } from "@/lib/payment/access";
import type { OrderPaymentState, PaymentInstructions, PaymentStatus } from "@/lib/payment/types";
import { parsePaymentRoute, validateRetryInput } from "@/lib/payment/validation";
import { NextRequest, NextResponse } from "next/server";

const PAYMENT_STATUSES = new Set<PaymentStatus>(["PENDING", "SUCCEEDED", "FAILED", "EXPIRED", "REQUIRES_RECONCILIATION", "COD_PENDING_COLLECTION"]);
const ORDER_PAYMENT_STATES = new Set<OrderPaymentState>(["PENDING_PAYMENT", "PAID", "PAYMENT_EXPIRED", "PAYMENT_REVIEW", "PAYMENT_ON_DELIVERY"]);

interface UpstreamError {
  code?: string;
  message?: string;
  requestId?: string;
}

class UpstreamHttpError extends Error {
  constructor(readonly status: number, readonly body: UpstreamError = {}) {
    super(body.message || `Payment upstream returned ${status}`);
  }
}

function errorResponse(code: string, message: string, status: number) {
  return NextResponse.json({ code, message, requestId: `BFF-PAYMENT-${code}` }, { status });
}

async function readJson(request: NextRequest): Promise<unknown> {
  try { return await request.json(); } catch { return undefined; }
}

async function upstreamJson(path: string, method: string, claim: PaymentAccessClaim, scenario?: string, body?: object): Promise<Record<string, unknown>> {
  const base = (process.env.PAYMENT_UPSTREAM_URL ?? "http://127.0.0.1:4016/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json", "X-Payment-Method": claim.method });
  if (body) headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario) headers.set("X-Mock-Scenario", scenario);
  const response = await fetch(`${base}/${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as Record<string, unknown> : {};
  if (!response.ok) throw new UpstreamHttpError(response.status, payload);
  return payload;
}

function paymentStatus(value: unknown, fallback: PaymentStatus): PaymentStatus {
  return typeof value === "string" && PAYMENT_STATUSES.has(value as PaymentStatus) ? value as PaymentStatus : fallback;
}

function orderPaymentState(value: unknown, fallback: OrderPaymentState): OrderPaymentState {
  return typeof value === "string" && ORDER_PAYMENT_STATES.has(value as OrderPaymentState) ? value as OrderPaymentState : fallback;
}

function publicInstructions(value: unknown): PaymentInstructions | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const source = value as Record<string, unknown>;
  if (![source.providerLabel, source.beneficiary, source.accountNumberMasked, source.transferReference].every((field) => typeof field === "string")) return null;
  if (source.qrPayloadType !== "DEMO_ONLY") return null;
  return {
    providerLabel: source.providerLabel as string,
    beneficiary: source.beneficiary as string,
    accountNumberMasked: source.accountNumberMasked as string,
    transferReference: source.transferReference as string,
    qrPayloadType: "DEMO_ONLY"
  };
}

export async function handlePaymentRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parsePaymentRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation || !route.orderId) {
    const notFound = route.error?.field === "path";
    return errorResponse(notFound ? "NOT_FOUND" : "INVALID_REQUEST", route.error?.message ?? "Yêu cầu Payment không hợp lệ.", notFound ? 404 : 400);
  }
  if (process.env.NODE_ENV === "production" && route.mockScenario) return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400);

  const claim = decodePaymentAccess(request.cookies.get(PAYMENT_ACCESS_COOKIE)?.value);
  if (!claim) return errorResponse("PAYMENT_ACCESS_REQUIRED", "Phiên truy cập Payment đã hết hạn hoặc không hợp lệ. Vui lòng tạo lại Order từ Checkout.", 401);
  if (claim.orderId !== route.orderId) return errorResponse("PAYMENT_NOT_FOUND", "Không tìm thấy thông tin thanh toán phù hợp.", 404);

  try {
    if (route.operation === "summary") {
      const source = await upstreamJson(`payments/orders/${encodeURIComponent(route.orderId)}`, "GET", claim, route.mockScenario);
      const fallbackStatus: PaymentStatus = claim.method === "COD" ? "COD_PENDING_COLLECTION" : "PENDING";
      const fallbackOrderState: OrderPaymentState = claim.method === "COD" ? "PAYMENT_ON_DELIVERY" : "PENDING_PAYMENT";
      return NextResponse.json({
        orderId: claim.orderId,
        orderNumber: claim.orderNumber,
        method: claim.method,
        amountVnd: claim.amountVnd,
        currency: "VND",
        paymentId: typeof source.paymentId === "string" ? source.paymentId : "payment-pending",
        status: paymentStatus(source.status, fallbackStatus),
        orderPaymentState: orderPaymentState(source.orderPaymentState, fallbackOrderState),
        expiresAt: claim.paymentExpiresAt,
        instructions: claim.method === "BANK_TRANSFER" ? publicInstructions(source.instructions) : null,
        checkedAt: typeof source.checkedAt === "string" && !Number.isNaN(Date.parse(source.checkedAt)) ? source.checkedAt : new Date().toISOString()
      });
    }

    if (route.operation === "status") {
      const source = await upstreamJson(`payments/orders/${encodeURIComponent(route.orderId)}/status`, "GET", claim, route.mockScenario);
      const fallbackStatus: PaymentStatus = claim.method === "COD" ? "COD_PENDING_COLLECTION" : "PENDING";
      const fallbackOrderState: OrderPaymentState = claim.method === "COD" ? "PAYMENT_ON_DELIVERY" : "PENDING_PAYMENT";
      return NextResponse.json({
        paymentId: typeof source.paymentId === "string" ? source.paymentId : "payment-pending",
        status: paymentStatus(source.status, fallbackStatus),
        orderPaymentState: orderPaymentState(source.orderPaymentState, fallbackOrderState),
        verifiedAt: typeof source.verifiedAt === "string" && !Number.isNaN(Date.parse(source.verifiedAt)) ? source.verifiedAt : null,
        message: typeof source.message === "string" ? source.message : "Chưa ghi nhận xác nhận thanh toán từ nguồn tin cậy.",
        canRetry: source.canRetry === true && claim.method === "BANK_TRANSFER"
      });
    }

    if (claim.method !== "BANK_TRANSFER") return errorResponse("PAYMENT_RETRY_NOT_ALLOWED", "COD không tạo lần thử thanh toán trực tuyến.", 409);
    const parsed = validateRetryInput(await readJson(request));
    if (!parsed.idempotencyKey) return NextResponse.json({ code: "INVALID_REQUEST", message: "Yêu cầu tạo lại Payment không hợp lệ.", errors: parsed.error ? [parsed.error] : [], requestId: "BFF-PAYMENT-INVALID_REQUEST" }, { status: 422 });
    const source = await upstreamJson(`payments/orders/${encodeURIComponent(route.orderId)}/attempts`, "POST", claim, route.mockScenario, { idempotencyKey: parsed.idempotencyKey });
    const expiresInSeconds = Number.isInteger(source.expiresInSeconds) ? Math.min(1800, Math.max(60, source.expiresInSeconds as number)) : 900;
    const expiresAt = new Date(Date.now() + expiresInSeconds * 1000).toISOString();
    const refreshedClaim = { ...claim, paymentExpiresAt: expiresAt };
    const response = NextResponse.json({
      paymentId: typeof source.paymentId === "string" ? source.paymentId : "payment-retry",
      status: "PENDING",
      orderPaymentState: "PENDING_PAYMENT",
      expiresAt,
      message: typeof source.message === "string" ? source.message : "Đã tạo lần thử thanh toán mới."
    }, { status: 201 });
    response.headers.append("Set-Cookie", paymentAccessCookie(refreshedClaim));
    return response;
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) {
      const status = cause.status >= 400 && cause.status <= 599 ? cause.status : 500;
      return errorResponse(cause.body.code || "PAYMENT_ERROR", cause.body.message || "Không thể xử lý Payment lúc này.", status);
    }
    return errorResponse("PAYMENT_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ Payment. Hãy kiểm tra Mockoon hoặc API Gateway.", 503);
  }
}
