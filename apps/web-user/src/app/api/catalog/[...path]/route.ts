import type {
  CatalogCategoriesResponse,
  CatalogProductDetail,
  CatalogProductsResponse,
  CatalogSearchResponse,
  PublicErrorBody
} from "@/lib/api-contract";
import { normalizePublicError } from "@/lib/api-contract";
import { mapCatalogCategoriesResponse } from "@/lib/catalog/mvp";
import { parseCatalogQuery, toPublicCatalogQuery } from "@/lib/catalog/query";
import { parseProductDetailRequest, toProductDetailUpstreamPath } from "@/lib/product-detail/validation";
import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const ALLOWED_PATHS = new Set(["products", "discovery-config"]);
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const FORWARDED_RESPONSE_HEADERS = ["cache-control", "retry-after", "x-request-id"];

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isCategoriesResponse(payload: unknown): payload is CatalogCategoriesResponse {
  return isRecord(payload) && Array.isArray(payload.categories);
}

function isProductList(payload: unknown): payload is CatalogProductsResponse {
  if (!payload || typeof payload !== "object") return false;
  const candidate = payload as Partial<CatalogProductsResponse>;
  return Array.isArray(candidate.items)
    && typeof candidate.total === "number"
    && typeof candidate.page === "number"
    && typeof candidate.page_size === "number"
    && typeof candidate.total_pages === "number";
}

function isSearchResponse(payload: unknown): payload is CatalogSearchResponse {
  return isRecord(payload)
    && typeof payload.query === "string"
    && Array.isArray(payload.items)
    && typeof payload.total === "number";
}

function isProductDetail(payload: unknown): payload is CatalogProductDetail {
  return isRecord(payload)
    && typeof payload.product_id === "string"
    && typeof payload.slug === "string"
    && Array.isArray(payload.images)
    && Array.isArray(payload.variants);
}

function normalizedSearchText(value: string) {
  return value.normalize("NFD").replace(/[\u0300-\u036f]/gu, "").toLocaleLowerCase("vi");
}

function upstreamHeaders(scenario?: string) {
  const headers = new Headers({ Accept: "application/json" });
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) {
    headers.set("X-Mock-Scenario", scenario);
  }
  return headers;
}

function responseHeaders(upstream: Response) {
  const headers = new Headers();
  const contentType = upstream.headers.get("content-type");
  if (contentType) headers.set("Content-Type", contentType);
  for (const name of FORWARDED_RESPONSE_HEADERS) {
    const value = upstream.headers.get(name);
    if (value) headers.set(name, value);
  }
  return headers;
}

function publicErrorResponse(
  status: number,
  code: string,
  message: string,
  requestId: string,
  details?: Record<string, unknown>
): Response {
  const body: PublicErrorBody = { code, message, requestId, ...(details ? { details } : {}) };
  return Response.json(body, { status });
}

async function relayUpstreamError(upstream: Response): Promise<Response> {
  const contentType = upstream.headers.get("content-type") ?? "";
  let payload: unknown;
  if (contentType.includes("application/json")) {
    try { payload = await upstream.json() as unknown; } catch { payload = undefined; }
  }
  const error: PublicErrorBody = normalizePublicError(payload, {
    code: "CATALOG_UPSTREAM_ERROR",
    message: "Không thể tải dữ liệu sản phẩm lúc này.",
    requestId: upstream.headers.get("x-request-id") ?? "BFF-CATALOG-UPSTREAM"
  });
  return Response.json(error, { status: upstream.status, headers: responseHeaders(upstream) });
}

async function resolveCategoryId(upstreamBase: string, category: string, headers: Headers): Promise<string> {
  const categoryResponse = await fetch(`${upstreamBase}/categories`, {
    headers,
    cache: "no-store",
    redirect: "manual"
  });
  if (!categoryResponse.ok) return category;
  const payload = await categoryResponse.json() as unknown;
  if (!isCategoriesResponse(payload)) return category;
  return payload.categories.find((item) => item.slug === category || item.category_id === category)?.category_id ?? category;
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const pathParts = (await context.params).path;
  const path = pathParts.join("/");
  const isDiscoveryPath = ALLOWED_PATHS.has(path);
  const detailValidation = !isDiscoveryPath ? parseProductDetailRequest(pathParts, request.nextUrl.searchParams) : undefined;
  const isDetailPath = Boolean(detailValidation?.slug);

  if (request.method !== "GET" || (!isDiscoveryPath && !isDetailPath)) {
    if (request.method === "GET" && detailValidation?.error?.field !== "path") {
      return publicErrorResponse(
        400,
        "INVALID_PRODUCT_SLUG",
        detailValidation?.error?.message ?? "Slug sản phẩm không hợp lệ.",
        "BFF-CATALOG-SLUG",
        { errors: detailValidation?.error ? [detailValidation.error] : [] }
      );
    }
    return publicErrorResponse(404, "NOT_FOUND", "Catalog route is not available.", "BFF-CATALOG-ROUTE");
  }

  const scenario = isDetailPath ? detailValidation?.mockScenario : request.nextUrl.searchParams.get("mockScenario") ?? undefined;
  const headers = upstreamHeaders(scenario);
  const upstreamBase = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");

  try {
    if (path === "discovery-config") {
      const upstream = await fetch(`${upstreamBase}/categories`, { headers, cache: "no-store", redirect: "manual" });
      if (!upstream.ok) return relayUpstreamError(upstream);
      const payload = await upstream.json() as unknown;
      if (!isCategoriesResponse(payload)) {
        return publicErrorResponse(502, "INVALID_CATALOG_RESPONSE", "Dữ liệu danh mục không đúng contract.", "BFF-CATALOG-CATEGORIES");
      }
      return Response.json(mapCatalogCategoriesResponse(payload));
    }

    if (path === "products") {
      const validation = parseCatalogQuery(request.nextUrl.searchParams);
      if (!validation.query) {
        return publicErrorResponse(400, "INVALID_QUERY", "Bộ lọc sản phẩm không hợp lệ.", "BFF-CATALOG-QUERY", { errors: validation.errors });
      }

      const upstreamParams = toPublicCatalogQuery(validation.query);
      if (validation.query.category) {
        upstreamParams.set("category_id", await resolveCategoryId(upstreamBase, validation.query.category, headers));
      }
      const upstreamPath = validation.query.q ? "products/search" : "products";
      const upstream = await fetch(`${upstreamBase}/${upstreamPath}?${upstreamParams}`, {
        headers,
        cache: "no-store",
        redirect: "manual"
      });
      if (!upstream.ok) return relayUpstreamError(upstream);

      const upstreamPayload = await upstream.json() as unknown;
      let payload: CatalogProductsResponse | CatalogSearchResponse;
      if (validation.query.q && isSearchResponse(upstreamPayload)) {
        payload = upstreamPayload;
      } else if (!validation.query.q && isProductList(upstreamPayload)) {
        payload = upstreamPayload;
      } else if (validation.query.q && process.env.NODE_ENV !== "production") {
        // mobile_pbl.json currently places /products/:id_or_slug before /products/search,
        // so Mockoon can capture "search" as a slug. Keep this compatibility local only.
        const fallbackParams = toPublicCatalogQuery({ ...validation.query, q: undefined });
        const fallback = await fetch(`${upstreamBase}/products?${fallbackParams}`, { headers, cache: "no-store", redirect: "manual" });
        if (!fallback.ok) return relayUpstreamError(fallback);
        const fallbackPayload = await fallback.json() as unknown;
        if (!isProductList(fallbackPayload)) {
          return publicErrorResponse(502, "INVALID_CATALOG_RESPONSE", "Dữ liệu danh mục không đúng cấu trúc mong đợi.", "BFF-CATALOG-LIST");
        }
        const keyword = normalizedSearchText(validation.query.q);
        const items = fallbackPayload.items.filter((item) => normalizedSearchText(`${item.name} ${item.summary}`).includes(keyword));
        payload = { query: validation.query.q, items, total: items.length };
      } else {
        return publicErrorResponse(502, "INVALID_CATALOG_RESPONSE", "Dữ liệu danh mục không đúng cấu trúc mong đợi.", "BFF-CATALOG-LIST");
      }
      return Response.json(payload, { status: upstream.status, headers: responseHeaders(upstream) });
    }

    const upstream = await fetch(`${upstreamBase}/${toProductDetailUpstreamPath(detailValidation!.slug!)}`, {
      headers,
      cache: "no-store",
      redirect: "manual"
    });
    if (!upstream.ok) return relayUpstreamError(upstream);
    const payload = await upstream.json() as unknown;
    if (!isProductDetail(payload)) {
      return publicErrorResponse(502, "INVALID_CATALOG_RESPONSE", "Chi tiết sản phẩm không đúng contract.", "BFF-CATALOG-DETAIL");
    }
    return Response.json(payload);
  } catch {
    return publicErrorResponse(
      503,
      "CATALOG_UPSTREAM_UNAVAILABLE",
      "Không thể kết nối dịch vụ danh mục. Hãy kiểm tra Mockoon hoặc API Gateway.",
      "BFF-CATALOG-UPSTREAM"
    );
  }
}

export const GET = proxy;
