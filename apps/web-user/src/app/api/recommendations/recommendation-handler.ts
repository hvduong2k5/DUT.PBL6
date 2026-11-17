import { randomBytes, timingSafeEqual } from "node:crypto";
import { NextRequest, NextResponse } from "next/server";
import { applyTrustedAuthorization } from "@/lib/auth/server-credential";
import { storefrontDataMode } from "@/lib/storefront-sample";
import { GET as catalogGet } from "../catalog/[...path]/route";
import { publicWebOrigin } from "@/lib/auth/oidc-config";

const CSRF = "oma_rec_csrf";

function error(status: number, code: string) {
  return NextResponse.json(
    {
      code,
      message: "Chưa thể tải gợi ý sản phẩm.",
      requestId: crypto.randomUUID(),
    },
    { status, headers: { "Cache-Control": "private, no-store" } },
  );
}

function origin(request: NextRequest) {
  if (process.env.WEB_PUBLIC_URL || process.env.NODE_ENV === "production")
    return publicWebOrigin();
  if (!["localhost", "127.0.0.1", "[::1]"].includes(request.nextUrl.hostname))
    throw new Error("Web origin required");
  return request.nextUrl.origin;
}

async function limitedBody(request: NextRequest) {
  if (request.headers.get("content-type")?.split(";")[0] !== "application/json")
    throw new Error("JSON required");
  const reader = request.body?.getReader();
  if (!reader) throw new Error("Body required");
  const chunks: Uint8Array[] = [];
  let length = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      length += value.byteLength;
      if (length > 4096) {
        await reader.cancel();
        throw new Error("Body exceeds limit");
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  return Buffer.concat(chunks);
}

async function realRecommendationProxy(request: NextRequest, path: string[]) {
  const route = path.join("/");
  const allowed =
    (route === "" && request.method === "GET") ||
    (route === "consent" && ["GET", "PUT"].includes(request.method)) ||
    (route === "events" && request.method === "POST");
  if (!allowed) return error(404, "NOT_FOUND");
  if (process.env.NEXT_PUBLIC_RECOMMENDATIONS_ENABLED !== "true")
    return error(503, "RECOMMENDATION_DISABLED");
  const base = process.env.ANALYTICS_RECOMMENDATION_UPSTREAM_URL;
  if (!base) return error(503, "RECOMMENDATION_UPSTREAM_MISSING");
  const headers = new Headers({ Accept: "application/json" });
  let body: Buffer | undefined;
  try {
    const webOrigin = origin(request);
    applyTrustedAuthorization(headers, request);
    if (request.method !== "GET") {
      const cookie = request.cookies.get(CSRF)?.value ?? "";
      const token = request.headers.get("x-csrf-token") ?? "";
      if (
        request.headers.get("origin") !== webOrigin ||
        !/^[A-Za-z0-9_-]{43}$/u.test(cookie) ||
        token.length !== cookie.length ||
        !timingSafeEqual(Buffer.from(cookie), Buffer.from(token))
      ) {
        return error(403, "INVALID_CSRF_TOKEN");
      }
      try {
        body = await limitedBody(request);
      } catch {
        return error(400, "INVALID_RECOMMENDATION_BODY");
      }
      headers.set("Content-Type", "application/json");
      headers.set("Origin", webOrigin);
      for (const name of ["if-match", "idempotency-key"]) {
        const value = request.headers.get(name);
        if (value) headers.set(name, value);
      }
    }
    const suffix = route
      ? `/api/v1/recommendations/${route}`
      : "/api/v1/products/recommended";
    const url = new URL(base.replace(/\/$/u, "") + suffix);
    if (!route) url.search = request.nextUrl.search;
    const upstream = await fetch(url, {
      method: request.method,
      headers,
      body: body ? new Uint8Array(body) : undefined,
      cache: "no-store",
      redirect: "manual",
      signal: AbortSignal.timeout(1000),
    });
    if (upstream.status >= 300 && upstream.status < 400)
      return error(502, "INVALID_RECOMMENDATION_REDIRECT");
    const responseHeaders = new Headers({
      "Cache-Control": "private, no-store",
    });
    for (const name of [
      "content-type",
      "etag",
      "retry-after",
      "x-request-id",
    ]) {
      const value = upstream.headers.get(name);
      if (value) responseHeaders.set(name, value);
    }
    const response = new NextResponse(await upstream.arrayBuffer(), {
      status: upstream.status,
      headers: responseHeaders,
    });
    if (
      route === "consent" &&
      request.method === "GET" &&
      upstream.ok &&
      !request.cookies.has(CSRF)
    ) {
      response.cookies.set(CSRF, randomBytes(32).toString("base64url"), {
        path: "/",
        sameSite: "strict",
        secure: webOrigin.startsWith("https:"),
        maxAge: 86400,
        httpOnly: false,
      });
    }
    return response;
  } catch {
    return error(503, "RECOMMENDATION_TEMPORARILY_UNAVAILABLE");
  }
}

async function catalogRecommendations(request: NextRequest) {
  const placement = request.nextUrl.searchParams.get("placement") ?? "home";
  if (!["home", "cart", "product_detail"].includes(placement))
    return error(400, "INVALID_PLACEMENT");
  const listRequest = new NextRequest(
    new URL("/api/catalog/products?pageSize=12", request.url),
  );
  const response = await catalogGet(listRequest, {
    params: Promise.resolve({ path: ["products"] }),
  });
  if (!response.ok) return new NextResponse(response.body, { status: response.status, headers: response.headers });
  const list = await response.json();
  const excluded = new Set(request.nextUrl.searchParams.getAll("skuCodes"));
  const currentProduct = request.nextUrl.searchParams.get("productId");
  const candidates = await Promise.all(
    (list.items ?? []).map(
      async (item: {
        slug: string;
        product_id: string;
        name: string;
        thumbnail_url: string;
      }) => {
        if (item.product_id === currentProduct) return null;
        const detail = await catalogGet(
          new NextRequest(
            new URL(
              "/api/catalog/products/" + encodeURIComponent(item.slug),
              request.url,
            ),
          ),
          { params: Promise.resolve({ path: ["products", item.slug] }) },
        );
        if (!detail.ok) return null;
        const product = await detail.json();
        const variant = product.variants?.find(
          (v: { stock_available: number | null }) =>
            v.stock_available !== null && v.stock_available > 0,
        );
        if (
          !variant ||
          excluded.has(variant.sku_code) ||
          variant.sku_code === request.nextUrl.searchParams.get("skuCode")
        )
          return null;
        return {
          productId: item.product_id,
          skuCode: variant.sku_code,
          slug: item.slug,
          name: item.name,
          thumbnailUrl: item.thumbnail_url,
          priceVnd: Number(variant.price.units),
          algorithmVersion: "catalog-preview-v1",
          reasonCode: "CATALOG_FALLBACK",
        };
      },
    ),
  );
  const limit = Math.min(
    12,
    Math.max(1, Number(request.nextUrl.searchParams.get("limit")) || 6),
  );
  const items = candidates
    .filter(Boolean)
    .slice(0, limit)
    .map((item, index) => ({ ...item, rank: index + 1 }));
  return NextResponse.json(
    {
      recommendationId: "catalog-preview-" + crypto.randomUUID(),
      placement,
      strategy: items.length ? "FALLBACK" : "EMPTY",
      reasonCodes: ["CATALOG_FALLBACK"],
      items,
    },
    {
      headers: {
        "Cache-Control": "private, no-store",
        "X-Storefront-Data-Source":
          response.headers.get("X-Storefront-Data-Source") ?? "real",
        "X-Recommendation-Source": "catalog-preview",
      },
    },
  );
}

export async function recommendationProxy(
  request: NextRequest,
  path: string[],
) {
  const mode = storefrontDataMode();
  if (path.length || request.method !== "GET" || mode === "real")
    return realRecommendationProxy(request, path);
  if (mode === "mock") return catalogRecommendations(request);
  const response = await realRecommendationProxy(request, path);
  if (response.status >= 500) return catalogRecommendations(request);
  if (response.ok) {
    const data = await response
      .clone()
      .json()
      .catch(() => null);
    if (Array.isArray(data?.items) && data.items.length === 0)
      return catalogRecommendations(request);
  }
  return response;
}
