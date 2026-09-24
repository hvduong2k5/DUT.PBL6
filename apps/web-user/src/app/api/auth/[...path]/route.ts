import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const ALLOWED_ROUTES = new Set([
  "GET me",
  "POST login",
  "POST logout",
  "POST register",
  "POST registration-verifications/confirm",
  "POST recovery-requests",
  "POST recovery-requests/verify",
  "POST recovery-requests/reset"
]);

const RESEND_ROUTE = /^registration-verifications\/[^/]+\/resend$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const FORWARDED_RESPONSE_HEADERS = ["cache-control", "clear-site-data", "retry-after", "set-cookie", "x-request-id"];

function isAllowed(method: string, path: string): boolean {
  return ALLOWED_ROUTES.has(`${method} ${path}`) || (method === "POST" && RESEND_ROUTE.test(path));
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
    return Response.json({ code: "NOT_FOUND", message: "Auth route is not available." }, { status: 404 });
  }

  if (method !== "GET" && !hasTrustedOrigin(request)) {
    return Response.json({ code: "ORIGIN_NOT_ALLOWED", message: "Request origin is not allowed." }, { status: 403 });
  }

  const upstreamBase = (process.env.AUTH_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json" });
  const contentType = request.headers.get("content-type");
  const cookie = request.headers.get("cookie");
  const scenario = request.headers.get("x-mock-scenario");

  if (contentType) headers.set("Content-Type", contentType);
  if (cookie) headers.set("Cookie", cookie);
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) {
    headers.set("X-Mock-Scenario", scenario);
  }

  try {
    const upstream = await fetch(`${upstreamBase}/auth/${path}`, {
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
    return Response.json(
      {
        code: "AUTH_UPSTREAM_UNAVAILABLE",
        message: "Không thể kết nối dịch vụ xác thực. Hãy kiểm tra Mockoon hoặc API Gateway.",
        requestId: "BFF-AUTH-UPSTREAM"
      },
      { status: 503 }
    );
  }
}

export const GET = proxy;
export const POST = proxy;
