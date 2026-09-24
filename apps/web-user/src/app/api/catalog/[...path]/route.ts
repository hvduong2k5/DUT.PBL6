import { parseCatalogQuery, toMockoonQuery } from "@/lib/catalog/query";
import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const ALLOWED_PATHS = new Set(["products", "discovery-config"]);
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const FORWARDED_RESPONSE_HEADERS = ["cache-control", "retry-after", "x-request-id", "x-total-count", "x-filtered-count"];

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const path = (await context.params).path.join("/");
  if (request.method !== "GET" || !ALLOWED_PATHS.has(path)) {
    return Response.json({ code: "NOT_FOUND", message: "Catalog route is not available." }, { status: 404 });
  }

  const scenario = request.nextUrl.searchParams.get("mockScenario");
  const upstreamParams = new URLSearchParams();
  if (path === "products") {
    const validation = parseCatalogQuery(request.nextUrl.searchParams);
    if (!validation.query) {
      return Response.json({ code: "INVALID_QUERY", message: "Bộ lọc sản phẩm không hợp lệ.", errors: validation.errors }, { status: 400 });
    }
    toMockoonQuery(validation.query).forEach((value, key) => upstreamParams.set(key, value));
  }

  const upstreamBase = (process.env.CATALOG_UPSTREAM_URL ?? "http://127.0.0.1:4012/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json" });
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) {
    headers.set("X-Mock-Scenario", scenario);
  }

  try {
    const suffix = upstreamParams.size ? `?${upstreamParams}` : "";
    const upstream = await fetch(`${upstreamBase}/catalog/${path}${suffix}`, {
      method: "GET",
      headers,
      cache: "no-store",
      redirect: "manual"
    });
    const responseHeaders = new Headers();
    const contentType = upstream.headers.get("content-type");
    if (contentType) responseHeaders.set("Content-Type", contentType);
    for (const name of FORWARDED_RESPONSE_HEADERS) {
      const value = upstream.headers.get(name);
      if (value) responseHeaders.set(name, value);
    }
    let body: BodyInit;
    if (path === "products" && upstream.ok && contentType?.includes("application/json")) {
      const payload = await upstream.json() as Array<Record<string, unknown>>;
      body = JSON.stringify(payload.map((item) => {
        const publicItem = { ...item };
        delete publicItem.priceVnd;
        delete publicItem.weightGrams;
        delete publicItem.isAvailable;
        delete publicItem.curatedRank;
        if (publicItem.matchedOffer && typeof publicItem.matchedOffer === "object") {
          const publicOffer = { ...(publicItem.matchedOffer as Record<string, unknown>) };
          delete publicOffer.availableQuantity;
          publicItem.matchedOffer = publicOffer;
        }
        return publicItem;
      }));
    } else {
      body = await upstream.arrayBuffer();
    }
    return new Response(body, { status: upstream.status, headers: responseHeaders });
  } catch {
    return Response.json({
      code: "CATALOG_UPSTREAM_UNAVAILABLE",
      message: "Không thể kết nối dịch vụ danh mục. Hãy kiểm tra Mockoon hoặc API Gateway.",
      requestId: "BFF-CATALOG-UPSTREAM"
    }, { status: 503 });
  }
}

export const GET = proxy;
