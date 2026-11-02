import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

const RECOVERY_ROUTES = new Set(["recovery-requests", "recovery-requests/verify", "recovery-requests/reset"]);
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;

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

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<NextResponse> {
  const path = (await context.params).path.join("/");
  if (request.method !== "POST" || !RECOVERY_ROUTES.has(path)) {
    return NextResponse.json({ code: "NOT_FOUND", message: "Auth route is not available." }, { status: 404 });
  }
  if (!hasTrustedOrigin(request)) {
    return NextResponse.json({ code: "ORIGIN_NOT_ALLOWED", message: "Request origin is not allowed." }, { status: 403 });
  }

  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json", "Content-Type": request.headers.get("content-type") ?? "application/json" });
  const scenario = request.headers.get("x-mock-scenario");
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) headers.set("X-Mock-Scenario", scenario);
  try {
    const upstream = await fetch(`${base}/auth/${path}`, { method: "POST", headers, body: await request.text(), cache: "no-store", redirect: "manual" });
    const responseHeaders = new Headers();
    const contentType = upstream.headers.get("content-type");
    const retryAfter = upstream.headers.get("retry-after");
    if (contentType) responseHeaders.set("Content-Type", contentType);
    if (retryAfter) responseHeaders.set("Retry-After", retryAfter);
    return new NextResponse(upstream.status === 204 ? null : await upstream.arrayBuffer(), { status: upstream.status, headers: responseHeaders });
  } catch {
    return NextResponse.json({ code: "AUTH_UPSTREAM_UNAVAILABLE", message: "Không thể kết nối dịch vụ xác thực." }, { status: 503 });
  }
}

export const POST = proxy;
