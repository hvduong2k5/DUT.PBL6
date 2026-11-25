import { createHash } from "node:crypto";
import { NextRequest, NextResponse } from "next/server";
import { hasTrustedOrigin } from "@/lib/auth/request-origin";
import { customerAccessFailure } from "@/lib/auth/customer-access-server";
import { authorizeStorefrontCapability, hasCustomerCredential, storefrontUpstream, withStorefrontAuth } from "@/lib/auth/storefront-credential";
import { normalizePublicError } from "@/lib/api-contract";

export function guestPaymentCookie(orderId: string): string { return `oma_guest_payment_${orderId.replaceAll("-", "")}`; }

export async function handleGuestPaymentAccess(request: NextRequest, orderId: string): Promise<NextResponse> {
  return withStorefrontAuth(request, () => handleResolvedGuestPaymentAccess(request, orderId)) as Promise<NextResponse>;
}

async function handleResolvedGuestPaymentAccess(request: NextRequest, orderId: string): Promise<NextResponse> {
  const error = (status: number, code: string, message: string) => NextResponse.json({ code, message, requestId: "BFF-GUEST-PAYMENT" }, { status, headers: { "Cache-Control": "no-store" } });
  if (!/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/iu.test(orderId)) return error(400, "INVALID_ORDER_ID", "Mã đơn không hợp lệ.");
  if (!hasTrustedOrigin(request)) return error(403, "ORIGIN_NOT_ALLOWED", "Request origin is not allowed.");
  let force = false;
  if (request.body) {
    let body: unknown; try { body = await request.json(); } catch { return error(400, "INVALID_REQUEST", "Yêu cầu không hợp lệ."); }
    if (!body || typeof body !== "object" || Array.isArray(body) || Object.keys(body).some((key) => key !== "force") || ("force" in body && typeof body.force !== "boolean")) return error(400, "INVALID_REQUEST", "Yêu cầu không hợp lệ.");
    force = "force" in body && body.force === true;
  }
  try {
    const credential = await authorizeStorefrontCapability(request, "ORDER_HISTORY_VIEW");
    if (hasCustomerCredential(request)) return new NextResponse(null, { status: 204 });
    if (!credential.token) return error(401, "AUTH_REQUIRED", "Phiên mua hàng đã hết hạn.");
    if (process.env.GUEST_VIETQR_CHECKOUT_ENABLED !== "true") return error(403, "GUEST_CHECKOUT_DISABLED", "VietQR cho khách chưa được bật.");
    const cookie = guestPaymentCookie(orderId);
    if (!force && request.cookies.has(cookie)) return new NextResponse(null, { status: 204 });
    const key = createHash("sha256").update(`${credential.token}:${orderId}:${Math.floor(Date.now() / 600000)}`).digest("hex");
    const response = await fetch(`${storefrontUpstream(request)}/orders/${encodeURIComponent(orderId)}/guest-payment-access`, { method: "POST", headers: { Accept: "application/json", Authorization: `Bearer ${credential.token}`, "Idempotency-Key": key }, cache: "no-store", redirect: "manual" });
    const body = await response.json() as unknown;
    if (!response.ok) return NextResponse.json(normalizePublicError(body, { code: "GUEST_PAYMENT_ACCESS_UNAVAILABLE", requestId: "BFF-GUEST-PAYMENT" }), { status: response.status, headers: { "Cache-Control": "no-store" } });
    if (!body || typeof body !== "object" || Array.isArray(body)) return error(502,"INVALID_GUEST_PAYMENT_GRANT","Quyền xem thanh toán không hợp lệ.");
    const grant = body as { access_token?: unknown; order_id?: unknown; expires_at?: unknown; token_type?: unknown };
    if (typeof grant.access_token !== "string" || !grant.access_token || grant.access_token.length > 8192 || grant.order_id !== orderId || grant.token_type !== "Bearer" || typeof grant.expires_at !== "string" || !Number.isFinite(Date.parse(grant.expires_at)) || Date.parse(grant.expires_at) <= Date.now()) return error(502, "INVALID_GUEST_PAYMENT_GRANT", "Quyền xem thanh toán không hợp lệ.");
    const result = new NextResponse(null, { status: 204, headers: { "Cache-Control": "no-store" } });
    result.cookies.set(cookie, grant.access_token, { httpOnly: true, sameSite: "lax", secure: request.nextUrl.protocol === "https:" || process.env.NODE_ENV === "production" && process.env.WEB_OIDC_ALLOW_LOCAL_HTTP !== "true", path: `/api/payments/orders/${orderId}`, expires: new Date(grant.expires_at) });
    return result;
  } catch (cause) {
    const failure = customerAccessFailure(cause);
    return error(failure.status, failure.code, failure.message);
  }
}
