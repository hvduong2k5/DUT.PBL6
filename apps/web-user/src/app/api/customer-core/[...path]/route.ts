import { NextRequest } from "next/server";
import { authorizeCustomerCapability, customerAccessFailure } from "@/lib/auth/customer-access-server";
import { credentialHeaders } from "@/lib/auth/server-credential";
import { withCustomerAuth } from "@/lib/auth/customer-auth";
import type { CustomerCapability } from "@/lib/auth/capabilities";
import { gatewayUpstreamUrl, customerCommerceUpstreamUrl } from "@/lib/upstream";

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
  return withCustomerAuth(request, (credential) => proxyWithCredential(request, context, credential));
}

async function proxyWithCredential(request: NextRequest, context: { params: Promise<{ path: string[] }> }, credential: string | undefined): Promise<Response> {
  const parts = (await context.params).path;
  const path = parts.join("/");
  if (!isAllowed(request.method, parts)) {
    return Response.json({ error_code: "ERR_NOT_FOUND", user_message: "Customer route is not available." }, { status: 404 });
  }

  const upstreamBase = path === "promotions/vouchers"
    ? gatewayUpstreamUrl()
    : path === "loyalty/points"
      ? customerCommerceUpstreamUrl()
      : (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const scenario = request.nextUrl.searchParams.get("mockScenario");
  const headers = new Headers({ Accept: "application/json" });
  credentialHeaders(headers, credential);
  if (request.method === "POST") headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) {
    headers.set("X-Mock-Scenario", scenario);
  }

  const requiredCapability: CustomerCapability | undefined = path === "reviews" ? "REVIEW_CREATE" : path === "loyalty/points" ? "LOYALTY_VIEW" : undefined;
  if (requiredCapability) {
    try {
      await authorizeCustomerCapability(credential, requiredCapability);
    } catch (cause) {
      const failure = customerAccessFailure(cause);
      return Response.json({ error_code: failure.code, user_message: failure.message }, { status: failure.status });
    }
  }

  try {
    const query = new URLSearchParams();
    if (path === "promotions/vouchers") {
      for (const name of ["q", "page", "pageSize", "status"]) {
        const value = request.nextUrl.searchParams.get(name);
        if (value !== null) query.set(name, value);
      }
    }
    const upstream = await fetch(`${upstreamBase}/${parts.map(encodeURIComponent).join("/")}${query.size ? `?${query}` : ""}`, {
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
      user_message: "Chưa thể kết nối dịch vụ. Vui lòng thử lại sau."
    }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
