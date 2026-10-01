import {
  mapCustomerCoreConfig,
  mapCustomerCoreProductSummary,
  type CustomerCoreCategory,
  type CustomerCoreProductList
} from "@/lib/catalog/customer-core";
import { parseCatalogQuery, toCustomerCoreQuery } from "@/lib/catalog/query";
import { mapCustomerCoreProductDetail, type CustomerCoreProductDetail } from "@/lib/product-detail/customer-core";
import { parseProductDetailRequest, toProductDetailUpstreamPath } from "@/lib/product-detail/validation";
import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const ALLOWED_PATHS = new Set(["products", "discovery-config"]);
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const FORWARDED_RESPONSE_HEADERS = ["cache-control", "retry-after", "x-request-id"];

function isProductList(payload: unknown): payload is CustomerCoreProductList {
  if (!payload || typeof payload !== "object") return false;
  const candidate = payload as Partial<CustomerCoreProductList>;
  return Array.isArray(candidate.items) && typeof candidate.total === "number";
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

async function resolveCategoryId(upstreamBase: string, category: string, headers: Headers): Promise<string> {
  const categoryResponse = await fetch(`${upstreamBase}/categories`, {
    headers,
    cache: "no-store",
    redirect: "manual"
  });
  if (!categoryResponse.ok) return category;
  const payload = await categoryResponse.json() as { categories?: CustomerCoreCategory[] };
  return payload.categories?.find((item) => item.slug === category || item.category_id === category)?.category_id ?? category;
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const pathParts = (await context.params).path;
  const path = pathParts.join("/");
  const isDiscoveryPath = ALLOWED_PATHS.has(path);
  const detailValidation = !isDiscoveryPath ? parseProductDetailRequest(pathParts, request.nextUrl.searchParams) : undefined;
  const isDetailPath = Boolean(detailValidation?.slug);

  if (request.method !== "GET" || (!isDiscoveryPath && !isDetailPath)) {
    if (request.method === "GET" && detailValidation?.error?.field !== "path") {
      return Response.json({
        code: "INVALID_PRODUCT_SLUG",
        message: detailValidation?.error?.message ?? "Slug sản phẩm không hợp lệ.",
        errors: detailValidation?.error ? [detailValidation.error] : []
      }, { status: 400 });
    }
    return Response.json({ code: "NOT_FOUND", message: "Catalog route is not available." }, { status: 404 });
  }

  const scenario = isDetailPath ? detailValidation?.mockScenario : request.nextUrl.searchParams.get("mockScenario") ?? undefined;
  const headers = upstreamHeaders(scenario);
  const upstreamBase = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");

  try {
    if (path === "discovery-config") {
      const upstream = await fetch(`${upstreamBase}/categories`, { headers, cache: "no-store", redirect: "manual" });
      if (!upstream.ok) return new Response(await upstream.arrayBuffer(), { status: upstream.status, headers: responseHeaders(upstream) });
      const payload = await upstream.json() as { categories: CustomerCoreCategory[] };
      return Response.json(mapCustomerCoreConfig(payload.categories));
    }

    if (path === "products") {
      const validation = parseCatalogQuery(request.nextUrl.searchParams);
      if (!validation.query) {
        return Response.json({ code: "INVALID_QUERY", message: "Bộ lọc sản phẩm không hợp lệ.", errors: validation.errors }, { status: 400 });
      }

      const upstreamParams = toCustomerCoreQuery(validation.query);
      if (validation.query.category) {
        upstreamParams.set("category_id", await resolveCategoryId(upstreamBase, validation.query.category, headers));
      }
      const upstreamPath = validation.query.q ? "products/search" : "products";
      const upstream = await fetch(`${upstreamBase}/${upstreamPath}?${upstreamParams}`, {
        headers,
        cache: "no-store",
        redirect: "manual"
      });
      if (!upstream.ok) return new Response(await upstream.arrayBuffer(), { status: upstream.status, headers: responseHeaders(upstream) });

      const upstreamPayload = await upstream.json() as unknown;
      let payload: CustomerCoreProductList;
      if (isProductList(upstreamPayload)) {
        payload = upstreamPayload;
      } else if (validation.query.q && process.env.NODE_ENV !== "production") {
        // mobile_pbl.json currently places /products/:id_or_slug before /products/search,
        // so Mockoon can capture "search" as a slug. Keep this compatibility local only.
        const fallbackParams = toCustomerCoreQuery({ ...validation.query, q: undefined });
        const fallback = await fetch(`${upstreamBase}/products?${fallbackParams}`, { headers, cache: "no-store", redirect: "manual" });
        if (!fallback.ok) return new Response(await fallback.arrayBuffer(), { status: fallback.status, headers: responseHeaders(fallback) });
        const fallbackPayload = await fallback.json() as unknown;
        if (!isProductList(fallbackPayload)) {
          return Response.json({ code: "INVALID_CATALOG_RESPONSE", message: "Dữ liệu danh mục không đúng cấu trúc mong đợi." }, { status: 502 });
        }
        const keyword = normalizedSearchText(validation.query.q);
        const items = fallbackPayload.items.filter((item) => normalizedSearchText(`${item.name} ${item.summary}`).includes(keyword));
        payload = { ...fallbackPayload, query: validation.query.q, items, total: items.length };
      } else {
        return Response.json({ code: "INVALID_CATALOG_RESPONSE", message: "Dữ liệu danh mục không đúng cấu trúc mong đợi." }, { status: 502 });
      }
      const publicHeaders = responseHeaders(upstream);
      publicHeaders.set("Content-Type", "application/json");
      publicHeaders.set("x-filtered-count", String(payload.total));
      publicHeaders.set("x-total-count", String(payload.total));
      return new Response(JSON.stringify(payload.items.map(mapCustomerCoreProductSummary)), { status: upstream.status, headers: publicHeaders });
    }

    const upstream = await fetch(`${upstreamBase}/${toProductDetailUpstreamPath(detailValidation!.slug!)}`, {
      headers,
      cache: "no-store",
      redirect: "manual"
    });
    if (!upstream.ok) return new Response(await upstream.arrayBuffer(), { status: upstream.status, headers: responseHeaders(upstream) });
    const payload = await upstream.json() as CustomerCoreProductDetail;
    return Response.json(mapCustomerCoreProductDetail(payload));
  } catch {
    return Response.json({
      code: "CATALOG_UPSTREAM_UNAVAILABLE",
      message: "Không thể kết nối dịch vụ danh mục. Hãy kiểm tra Mockoon hoặc API Gateway.",
      requestId: "BFF-CATALOG-UPSTREAM"
    }, { status: 503 });
  }
}

export const GET = proxy;
