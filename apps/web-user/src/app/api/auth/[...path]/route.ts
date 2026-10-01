import { NextRequest, NextResponse } from "next/server";
import {
  mapCustomerCoreSession,
  toCustomerCoreLogin,
  toCustomerCoreRegistration,
  type CustomerCoreAuthResponse,
  type CustomerCoreTokens,
  type CustomerCoreUser
} from "@/lib/auth/customer-core";

export const dynamic = "force-dynamic";

const SUPPORTED_ROUTES = new Set(["GET me", "POST login", "POST logout", "POST register", "POST refresh", "POST recovery-requests", "POST recovery-requests/verify", "POST recovery-requests/reset"]);
const EXTENSION_ROUTES = new Set(["recovery-requests", "recovery-requests/verify", "recovery-requests/reset"]);
const LEGACY_UNSUPPORTED = new Set(["registration-verifications/confirm"]);
const RESEND_ROUTE = /^registration-verifications\/[^/]+\/resend$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const ACCESS_COOKIE = "oma_access_token";
const REFRESH_COOKIE = "oma_refresh_token";

function hasTrustedOrigin(request: NextRequest): boolean {
  const origin = request.headers.get("origin");
  if (!origin) return true;
  try {
    const source = new URL(origin);
    return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host;
  } catch { return false; }
}

async function readJson(request: NextRequest): Promise<Record<string, unknown> | undefined> {
  try {
    const value = await request.json();
    return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : undefined;
  } catch { return undefined; }
}

function clean(value: unknown) { return typeof value === "string" ? value.trim() : ""; }

function upstreamHeaders(request: NextRequest, hasBody = false, accessToken?: string) {
  const headers = new Headers({ Accept: "application/json" });
  if (hasBody) headers.set("Content-Type", "application/json");
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  const scenario = request.headers.get("x-mock-scenario");
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

async function upstream(request: NextRequest, path: string, method: string, body?: unknown, accessToken?: string) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/auth/${path}`, {
    method,
    headers: upstreamHeaders(request, body !== undefined, accessToken),
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

function relayError(payload: unknown, status: number) {
  return NextResponse.json(payload && typeof payload === "object" ? payload : { error_code: "ERR_AUTH", user_message: "Không thể xử lý yêu cầu xác thực lúc này." }, { status });
}

function setTokenCookies(response: NextResponse, tokens: CustomerCoreTokens, remember = false) {
  const shared = { httpOnly: true, sameSite: "lax" as const, secure: process.env.NODE_ENV === "production", path: "/" };
  response.cookies.set(ACCESS_COOKIE, tokens.access_token, { ...shared, maxAge: Math.max(60, tokens.expires_in) });
  response.cookies.set(REFRESH_COOKIE, tokens.refresh_token, { ...shared, ...(remember ? { maxAge: 30 * 24 * 60 * 60 } : {}) });
}

function clearTokenCookies(response: NextResponse) {
  const shared = { httpOnly: true, sameSite: "lax" as const, secure: process.env.NODE_ENV === "production", path: "/", maxAge: 0 };
  response.cookies.set(ACCESS_COOKIE, "", shared);
  response.cookies.set(REFRESH_COOKIE, "", shared);
}

async function extensionUpstream(request: NextRequest, path: string): Promise<NextResponse> {
  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json", "Content-Type": request.headers.get("content-type") ?? "application/json" });
  const scenario = request.headers.get("x-mock-scenario");
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) headers.set("X-Mock-Scenario", scenario);
  const upstreamResponse = await fetch(`${base}/auth/${path}`, {
    method: "POST",
    headers,
    body: await request.text(),
    cache: "no-store",
    redirect: "manual"
  });
  const responseHeaders = new Headers();
  const contentType = upstreamResponse.headers.get("content-type");
  const retryAfter = upstreamResponse.headers.get("retry-after");
  if (contentType) responseHeaders.set("Content-Type", contentType);
  if (retryAfter) responseHeaders.set("Retry-After", retryAfter);
  return new NextResponse(upstreamResponse.status === 204 ? null : await upstreamResponse.arrayBuffer(), { status: upstreamResponse.status, headers: responseHeaders });
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<NextResponse> {
  const method = request.method.toUpperCase();
  const path = (await context.params).path.join("/");
  if ((LEGACY_UNSUPPORTED.has(path) || RESEND_ROUTE.test(path)) && method === "POST") {
    return NextResponse.json({ code: "AUTH_FEATURE_NOT_SUPPORTED", message: "API customer hiện chưa hỗ trợ quy trình xác minh đăng ký riêng." }, { status: 501 });
  }
  if (!SUPPORTED_ROUTES.has(`${method} ${path}`)) return NextResponse.json({ code: "NOT_FOUND", message: "Auth route is not available." }, { status: 404 });
  if (method !== "GET" && !hasTrustedOrigin(request)) return NextResponse.json({ code: "ORIGIN_NOT_ALLOWED", message: "Request origin is not allowed." }, { status: 403 });

  try {
    if (EXTENSION_ROUTES.has(path)) return await extensionUpstream(request, path);

    if (path === "login") {
      const body = await readJson(request);
      const phoneNumber = clean(body?.phoneNumber).replace(/[ .-]/gu, "");
      const password = typeof body?.password === "string" ? body.password : "";
      if (!phoneNumber || !password) return NextResponse.json({ code: "ERR_VALIDATION", message: "Số điện thoại và mật khẩu là bắt buộc." }, { status: 422 });
      const result = await upstream(request, "login", "POST", toCustomerCoreLogin({ phoneNumber, password }));
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      const payload = result.payload as CustomerCoreAuthResponse;
      const response = NextResponse.json(mapCustomerCoreSession(payload.user));
      setTokenCookies(response, payload.tokens, body?.rememberMe === true);
      return response;
    }

    if (path === "register") {
      const body = await readJson(request);
      const fullName = clean(body?.fullName);
      const phoneNumber = clean(body?.phoneNumber).replace(/[ .-]/gu, "");
      const email = clean(body?.email);
      const password = typeof body?.password === "string" ? body.password : "";
      if (!fullName || !phoneNumber || !password) return NextResponse.json({ code: "ERR_VALIDATION", message: "Họ tên, số điện thoại và mật khẩu là bắt buộc." }, { status: 422 });
      const result = await upstream(request, "register", "POST", toCustomerCoreRegistration({ fullName, phoneNumber, email: email || undefined, password }));
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      const payload = result.payload as CustomerCoreAuthResponse;
      const response = NextResponse.json(mapCustomerCoreSession(payload.user), { status: 201 });
      setTokenCookies(response, payload.tokens);
      return response;
    }

    if (path === "me") {
      const accessToken = request.cookies.get(ACCESS_COOKIE)?.value;
      if (accessToken) {
        const result = await upstream(request, "me", "GET", undefined, accessToken);
        if (result.response.ok) return NextResponse.json(mapCustomerCoreSession(result.payload as CustomerCoreUser));
        if (result.response.status !== 401) return relayError(result.payload, result.response.status);
      }

      const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
      if (!refreshToken) return NextResponse.json({ error_code: "ERR_AUTH_UNAUTHORIZED", user_message: "Bạn chưa đăng nhập hoặc phiên đã hết hạn." }, { status: 401 });
      const refreshResult = await upstream(request, "refresh", "POST", { refresh_token: refreshToken });
      if (!refreshResult.response.ok) {
        const response = relayError(refreshResult.payload, refreshResult.response.status);
        clearTokenCookies(response);
        return response;
      }
      const tokens = refreshResult.payload as CustomerCoreTokens;
      const meResult = await upstream(request, "me", "GET", undefined, tokens.access_token);
      if (!meResult.response.ok) return relayError(meResult.payload, meResult.response.status);
      const response = NextResponse.json(mapCustomerCoreSession(meResult.payload as CustomerCoreUser));
      setTokenCookies(response, tokens);
      return response;
    }

    if (path === "refresh") {
      const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
      if (!refreshToken) return NextResponse.json({ error_code: "ERR_AUTH_UNAUTHORIZED", user_message: "Phiên đăng nhập không thể làm mới." }, { status: 401 });
      const result = await upstream(request, "refresh", "POST", { refresh_token: refreshToken });
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      const response = NextResponse.json({ refreshed: true });
      setTokenCookies(response, result.payload as CustomerCoreTokens, true);
      return response;
    }

    const accessToken = request.cookies.get(ACCESS_COOKIE)?.value;
    const result = await upstream(request, "logout", "POST", undefined, accessToken);
    const response = result.response.ok ? NextResponse.json(result.payload ?? { success: true }) : relayError(result.payload, result.response.status);
    clearTokenCookies(response);
    return response;
  } catch {
    return NextResponse.json({ code: "AUTH_UPSTREAM_UNAVAILABLE", message: "Không thể kết nối dịch vụ xác thực. Hãy kiểm tra Mockoon customer API." }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
