import { NextRequest } from "next/server";
import { authorizeCustomerCapability, customerAccessFailure } from "@/lib/auth/customer-access-server";
import { applyTrustedAuthorization, getTrustedAccessToken } from "@/lib/auth/server-credential";
import type { CustomerCapability } from "@/lib/auth/capabilities";

export const dynamic = "force-dynamic";

const SAFE_SEGMENT = /^[A-Za-z0-9._~-]{1,160}$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;

function isAllowed(method: string, parts: string[]) {
  const path = parts.join("/");
  if (method === "POST" && (path === "reviews" || path === "promotions/validate")) return true;
  if (method === "GET" && (path === "promotions/vouchers" || path === "loyalty/points")) return true;
  if (method !== "GET" || parts.length !== 3 && parts.length !== 2) return false;
  if (parts.length === 3) {
    return parts[0] === "reviews" && parts[1] === "products" && SAFE_SEGMENT.test(parts[2]);
  }
  return parts[0] === "trace" && SAFE_SEGMENT.test(parts[1]);
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const parts = (await context.params).path;
  const path = parts.join("/");
  if (!isAllowed(request.method, parts)) {
    return Response.json({ error_code: "ERR_NOT_FOUND", user_message: "Customer route is not available." }, { status: 404 });
  }

  const upstreamBase = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const scenario = request.nextUrl.searchParams.get("mockScenario");
  const headers = new Headers({ Accept: "application/json" });
  const accessToken = getTrustedAccessToken(request);
  applyTrustedAuthorization(headers, request);
  if (request.method === "POST") headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) {
    headers.set("X-Mock-Scenario", scenario);
  }

  const requiredCapability: CustomerCapability | undefined = path === "reviews" ? "REVIEW_CREATE" : path === "loyalty/points" ? "LOYALTY_VIEW" : undefined;
  if (requiredCapability) {
    try {
      await authorizeCustomerCapability(accessToken, requiredCapability);
    } catch (cause) {
      const failure = customerAccessFailure(cause);
      return Response.json({ error_code: failure.code, user_message: failure.message }, { status: failure.status });
    }
  }

  try {
    const upstream = await fetch(`${upstreamBase}/${parts.map(encodeURIComponent).join("/")}`, {
      method: request.method,
      headers,
      body: request.method === "POST" ? await request.text() : undefined,
      cache: "no-store",
      redirect: "manual"
    });
    const responseHeaders = new Headers();
    const contentType = upstream.headers.get("content-type");
    if (contentType) responseHeaders.set("Content-Type", contentType);
    const retryAfter = upstream.headers.get("retry-after");
    if (retryAfter) responseHeaders.set("Retry-After", retryAfter);
    return new Response(await upstream.arrayBuffer(), { status: upstream.status, headers: responseHeaders });
  } catch {
    return Response.json({
      error_code: "ERR_UPSTREAM_UNAVAILABLE",
      user_message: "Chưa thể kết nối dữ liệu customer. Hãy kiểm tra Mockoon mobile_pbl.json."
    }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
