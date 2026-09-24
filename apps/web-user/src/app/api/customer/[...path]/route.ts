import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const FIXED_ROUTES = new Set([
  "GET profile",
  "PATCH profile",
  "GET addresses",
  "POST addresses"
]);
const ADDRESS_ROUTE = /^addresses\/[^/]+$/u;
const DEFAULT_ROUTE = /^addresses\/[^/]+\/default$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const FORWARDED_RESPONSE_HEADERS = ["cache-control", "etag", "location", "retry-after", "x-request-id"];

function isAllowed(method: string, path: string): boolean {
  if (FIXED_ROUTES.has(`${method} ${path}`)) return true;
  if (ADDRESS_ROUTE.test(path)) return method === "PATCH" || method === "DELETE";
  return DEFAULT_ROUTE.test(path) && method === "PUT";
}

function hasTrustedOrigin(request: NextRequest): boolean {
  const origin = request.headers.get("origin");
  if (!origin) return true;
  try {
    const source = new URL(origin);
    return source.protocol === request.nextUrl.protocol && source.host === request.nextUrl.host;
  } catch {
    return false;
  }
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const method = request.method.toUpperCase();
  const path = (await context.params).path.join("/");
  if (!isAllowed(method, path)) {
    return Response.json({ code: "NOT_FOUND", message: "Customer route is not available." }, { status: 404 });
  }
  if (method !== "GET" && !hasTrustedOrigin(request)) {
    return Response.json({ code: "ORIGIN_NOT_ALLOWED", message: "Request origin is not allowed." }, { status: 403 });
  }

  const upstreamBase = (process.env.CUSTOMER_UPSTREAM_URL ?? "http://127.0.0.1:4011/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json" });
  for (const name of ["content-type", "cookie", "if-match"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  const scenario = request.headers.get("x-mock-scenario");
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) {
    headers.set("X-Mock-Scenario", scenario);
  }

  try {
    const upstream = await fetch(`${upstreamBase}/customers/me/${path}`, {
      method,
      headers,
      body: method === "GET" || method === "HEAD" ? undefined : await request.text(),
      cache: "no-store",
      redirect: "manual"
    });
    const responseHeaders = new Headers();
    const responseContentType = upstream.headers.get("content-type");
    if (responseContentType) responseHeaders.set("Content-Type", responseContentType);
    for (const name of FORWARDED_RESPONSE_HEADERS) {
      const value = upstream.headers.get(name);
      if (value) responseHeaders.set(name, value);
    }
    return new Response(upstream.status === 204 ? null : await upstream.arrayBuffer(), {
      status: upstream.status,
      headers: responseHeaders
    });
  } catch {
    return Response.json({
      code: "CUSTOMER_UPSTREAM_UNAVAILABLE",
      message: "Không thể kết nối dịch vụ hồ sơ. Hãy kiểm tra Mockoon hoặc API Gateway.",
      requestId: "BFF-CUSTOMER-UPSTREAM"
    }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
export const PATCH = proxy;
export const PUT = proxy;
export const DELETE = proxy;
