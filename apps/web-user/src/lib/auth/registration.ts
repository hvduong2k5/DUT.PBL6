import { NextRequest, NextResponse } from "next/server";
import { trustedOidcOrigin } from "./oidc";

// Registration creates Identity only. Login always goes through PKCE; Customer
// provisioning is asynchronous and never replaced with a browser-supplied ID.
export async function registerIdentity(request: NextRequest) {
  if (!trustedOidcOrigin(request)) return NextResponse.json({ code: "ORIGIN_NOT_ALLOWED" }, { status: 403 });
  const body = await request.json().catch(() => null);
  const key = request.headers.get("idempotency-key") ?? "";
  if (!body || typeof body.email !== "string" || typeof body.firstName !== "string" || typeof body.lastName !== "string" || typeof body.password !== "string" || !/^[A-Za-z0-9_-]{16,128}$/u.test(key)) {
    return NextResponse.json({ code: "INVALID_REGISTRATION", message: "Vui lòng kiểm tra thông tin đăng ký." }, { status: 400 });
  }
  const base = process.env.CUSTOMER_CORE_UPSTREAM_URL?.replace(/\/$/u, "");
  if (!base) throw new Error("Identity unavailable");
  const response = await fetch(`${base}/public/customer-registrations`, {
    method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": key },
    body: JSON.stringify({ email: body.email, firstName: body.firstName, lastName: body.lastName, rawPassword: body.password, idempotencyKey: key }),
    cache: "no-store", redirect: "error", signal: AbortSignal.timeout(15000)
  });
  const payload = await response.json().catch(() => null);
  if (!response.ok || payload?.success !== true || !["ACTIVE", "COMPLETED", "PENDING", "IDP_CREATED", "EVENT_QUEUED"].includes(payload?.data?.status)) {
    return NextResponse.json({ code: response.status === 409 ? "REGISTRATION_CONFLICT" : "REGISTRATION_FAILED", message: "Chưa thể đăng ký. Hãy kiểm tra thông tin hoặc thử lại với cùng yêu cầu." }, { status: response.status === 409 ? 409 : response.status === 400 || response.status === 422 ? 422 : 503 });
  }
  return NextResponse.json({ status: ["ACTIVE", "COMPLETED"].includes(payload.data.status) ? "REGISTERED" : "PROCESSING" }, { status: 202, headers: { "Cache-Control": "no-store", "Retry-After": "3" } });
}
