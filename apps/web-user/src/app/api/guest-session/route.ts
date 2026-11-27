import { NextRequest } from "next/server";
import { hasTrustedOrigin } from "@/lib/auth/request-origin";
import { customerAccessFailure } from "@/lib/auth/customer-access-server";
import { authorizeStorefrontCapability, GuestRateLimitError, withGuestCookie } from "@/lib/auth/storefront-credential";

export const dynamic = "force-dynamic";
export async function POST(request: NextRequest) {
  if (!hasTrustedOrigin(request)) return Response.json({ code: "ORIGIN_NOT_ALLOWED", message: "Request origin is not allowed.", requestId: "BFF-GUEST-ORIGIN" }, { status: 403 });
  if (request.body) {
    let body: unknown;
    try { body = await request.json(); } catch { body = undefined; }
    if (!body || typeof body !== "object" || Array.isArray(body) || Object.keys(body).length) return Response.json({ code: "INVALID_REQUEST", message: "Phiên mua hàng không nhận thông tin phân quyền từ client.", requestId: "BFF-GUEST-INPUT" }, { status: 400 });
  }
  return withGuestCookie(request, async () => {
    try {
      await authorizeStorefrontCapability(request, "CART_MANAGE", true);
      return Response.json({ ready: true });
    } catch (cause) {
      const failure = cause instanceof GuestRateLimitError ? { status: 429, code: cause.code, message: "Vui lòng thử lại sau." } : customerAccessFailure(cause);
      return Response.json({ code: failure.code, message: failure.message, requestId: "BFF-GUEST-SESSION" }, { status: failure.status });
    }
  });
}
