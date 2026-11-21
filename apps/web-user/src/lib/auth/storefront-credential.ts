import { NextRequest, NextResponse } from "next/server";
import { isIP } from "node:net";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import type { CustomerCapability } from "./capabilities";
import { authorizeCustomerCapability, CustomerAccessError } from "./customer-access-server";
import { ACCESS_TOKEN_COOKIE, SESSION_COOKIE, applyTrustedAuthorization, getTrustedAccessToken } from "./server-credential";
import { usesIdentityBff } from "./oidc-config";

export const GUEST_COOKIE = "oma_guest_session";
export const GUEST_CAPABILITIES: readonly CustomerCapability[] = ["CART_MANAGE", "CHECKOUT_CREATE", "ORDER_HISTORY_VIEW"];
type StorefrontCredential = { kind: "customer" | "guest"; token?: string; fresh?: boolean };
const credentials = new WeakMap<NextRequest, StorefrontCredential>();

function guestOrderBase(): string {
  if (usesIdentityBff() && !process.env.GUEST_ORDER_UPSTREAM_URL) throw new CustomerAccessError(503, "GUEST_ROUTE_UNAVAILABLE", "Chưa cấu hình cổng mua hàng không cần tài khoản.");
  return (process.env.GUEST_ORDER_UPSTREAM_URL ?? process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
}

export function hasCustomerCredential(request: NextRequest): boolean {
  // Presence is significant, including malformed or expired credentials.
  return request.cookies.has(SESSION_COOKIE) || request.cookies.has(ACCESS_TOKEN_COOKIE);
}

export async function authorizeStorefrontCapability(request: NextRequest, capability: CustomerCapability, issue = false): Promise<StorefrontCredential> {
  if (hasCustomerCredential(request)) {
    const token = getTrustedAccessToken(request);
    if (!token) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
    await authorizeCustomerCapability(token, capability);
    const credential: StorefrontCredential = { kind: "customer", token };
    credentials.set(request, credential);
    return credential;
  }
  if (!MVP_FEATURE_GATES.guestOrder || !GUEST_CAPABILITIES.includes(capability)) {
    throw new CustomerAccessError(401, "AUTH_REQUIRED", "Vui lòng đăng nhập để sử dụng chức năng này.");
  }
  let token = request.cookies.get(GUEST_COOKIE)?.value;
  let fresh = false;
  if (!token && issue) {
    if (process.env.GUEST_SESSION_ISSUANCE_ENABLED !== "true") throw new CustomerAccessError(403, "GUEST_SESSIONS_DISABLED", "Mua hàng không cần tài khoản chưa được bật.");
    const base = guestOrderBase();
    let response: Response;
    try {
      const headers = new Headers({ "Content-Type": "application/json", Accept: "application/json" });
      // Only configure a header overwritten by the trusted storefront ingress.
      const clientIpHeader = process.env.WEB_GUEST_CLIENT_IP_HEADER;
      const clientIp = clientIpHeader ? request.headers.get(clientIpHeader)?.trim() : undefined;
      if (clientIp && isIP(clientIp)) headers.set("X-Forwarded-For", clientIp);
      response = await fetch(`${base}/guest-sessions`, { method: "POST", headers, body: "{}", cache: "no-store", redirect: "manual" });
    } catch { throw new CustomerAccessError(503, "GUEST_SESSION_UNAVAILABLE", "Chưa thể tạo phiên mua hàng."); }
    if (!response.ok) {
      // Keep rate-limit status intact without exposing the upstream body.
      if (response.status === 429) throw new GuestRateLimitError();
      throw new CustomerAccessError(response.status === 403 ? 403 : 503, response.status === 403 ? "GUEST_SESSIONS_DISABLED" : "GUEST_SESSION_UNAVAILABLE", "Chưa thể tạo phiên mua hàng.");
    }
    const body = await response.json() as { access_token?: unknown };
    if (typeof body.access_token !== "string" || !body.access_token || body.access_token.length > 8192) throw new CustomerAccessError(503, "GUEST_SESSION_UNAVAILABLE", "Phiên mua hàng không hợp lệ.");
    token = body.access_token;
    fresh = true;
  }
  const credential: StorefrontCredential = { kind: "guest", token, fresh };
  credentials.set(request, credential);
  return credential;
}

export class GuestRateLimitError extends Error { readonly status = 429; readonly code = "GUEST_RATE_LIMITED"; }

export function applyStorefrontAuthorization(headers: Headers, request: NextRequest): void {
  const credential = credentials.get(request);
  if (credential?.kind === "guest") {
    headers.delete("Cookie");
    headers.delete("Authorization");
    if (credential.token) headers.set("Authorization", `Bearer ${credential.token}`);
  } else applyTrustedAuthorization(headers, request);
}

export function storefrontUpstream(request: NextRequest): string {
  if (credentials.get(request)?.kind === "guest") return guestOrderBase();
  return (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
}

export function guestCredential(request: NextRequest): boolean { return credentials.get(request)?.kind === "guest"; }
export function hasStorefrontToken(request: NextRequest): boolean { return Boolean(credentials.get(request)?.token); }

export async function withGuestCookie(request: NextRequest, handler: () => Promise<Response>): Promise<Response> {
  const response = await handler();
  response.headers.set("Cache-Control", "no-store");
  const credential = credentials.get(request);
  if (credential?.fresh && credential.token) {
    const result = new NextResponse(response.body, { status: response.status, headers: response.headers });
    result.cookies.set(GUEST_COOKIE, credential.token, { httpOnly: true, sameSite: "lax", secure: request.nextUrl.protocol === "https:" || (process.env.NODE_ENV === "production" && process.env.WEB_OIDC_ALLOW_LOCAL_HTTP !== "true"), path: "/", maxAge: 24 * 60 * 60 });
    return result;
  }
  return response;
}
