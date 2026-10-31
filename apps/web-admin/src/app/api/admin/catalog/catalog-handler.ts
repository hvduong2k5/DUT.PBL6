import { NextRequest, NextResponse } from "next/server";
import { parseAdminSession } from "@/lib/auth/types";
import type { AdminProductDetail, AdminProductList, AdminProductSku, AdminProductSummary, ProductSaleStatus, SkuSaleStatus } from "@/lib/catalog/types";
import { parseAdminCatalogPath } from "@/lib/catalog/validation";

interface UpstreamError { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
class UpstreamHttpError extends Error { constructor(readonly status: number, readonly body: UpstreamError = {}) { super(body.message || `Admin catalog upstream returned ${status}`); } }
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const string = (value: unknown, fallback = "") => typeof value === "string" ? value : fallback;
const number = (value: unknown, fallback = 0) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
const array = (value: unknown): unknown[] => Array.isArray(value) ? value : [];
const date = (value: unknown) => { const raw = string(value); return raw && !Number.isNaN(Date.parse(raw)) ? raw : new Date().toISOString(); };
const PRODUCT_STATUSES = new Set<ProductSaleStatus>(["ON_SALE", "PAUSED", "DISCONTINUED"]);
const SKU_STATUSES = new Set<SkuSaleStatus>(["ON_SALE", "PAUSED", "DISCONTINUED"]);

function responseError(code: string, message: string, status: number, errors: Array<{ field: string; message: string }> = []) {
  return NextResponse.json({ code, message, errors, requestId: `BFF-ADMIN-CATALOG-${code}` }, { status });
}
function baseUrl() { return (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, ""); }
function mockProfile() { return process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER"; }
function headers(request: NextRequest) {
  const result = new Headers({ Accept: "application/json" });
  const authorization = request.headers.get("authorization");
  if (authorization) result.set("Authorization", authorization);
  const profile = mockProfile();
  if (profile) result.set("X-Admin-Profile", profile);
  return result;
}
async function upstream(request: NextRequest, path: string) {
  const response = await fetch(`${baseUrl()}/${path}`, { headers: headers(request), cache: "no-store", redirect: "manual" });
  const payload: unknown = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new UpstreamHttpError(response.status, record(payload) ? payload : {});
  return record(payload) ? payload : {};
}
async function session(request: NextRequest) {
  const response = await fetch(`${baseUrl()}/admin/session`, { headers: headers(request), cache: "no-store", redirect: "manual" });
  if (!response.ok) throw new UpstreamHttpError(response.status, { code: "ADMIN_SESSION_REQUIRED", message: "Phiên nhân viên không hợp lệ." });
  const parsed = parseAdminSession(await response.json());
  if (!parsed) throw new UpstreamHttpError(502, { code: "ADMIN_SESSION_INVALID", message: "Projection quyền nhân viên không hợp lệ." });
  return parsed;
}
function productStatus(value: unknown): ProductSaleStatus { const status = string(value) as ProductSaleStatus; return PRODUCT_STATUSES.has(status) ? status : "DISCONTINUED"; }
function skuStatus(value: unknown): SkuSaleStatus { const status = string(value) as SkuSaleStatus; return SKU_STATUSES.has(status) ? status : "DISCONTINUED"; }
function mapSummary(value: unknown): AdminProductSummary {
  const item = record(value) ? value : {};
  return { productId: string(item.productId), name: string(item.name), categoryId: string(item.categoryId), categoryName: string(item.categoryName), saleStatus: productStatus(item.saleStatus), saleStatusLabel: string(item.saleStatusLabel), skuCount: Math.max(0, number(item.skuCount)), onSaleSkuCount: Math.max(0, number(item.onSaleSkuCount)), basePriceFromVnd: typeof item.basePriceFromVnd === "number" && Number.isFinite(item.basePriceFromVnd) ? Math.max(0, item.basePriceFromVnd) : null, foodInformationComplete: item.foodInformationComplete === true, updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1)) };
}
function mapList(payload: Record<string, unknown>): AdminProductList {
  const items = array(payload.items).map(mapSummary).filter((item) => item.productId);
  const statusCounts = array(payload.statusCounts).map((value) => { const item = record(value) ? value : {}; const raw = string(item.status) as "ALL" | ProductSaleStatus; return { status: raw === "ALL" || PRODUCT_STATUSES.has(raw as ProductSaleStatus) ? raw : "ALL", label: string(item.label), count: Math.max(0, number(item.count)) }; });
  for (const status of PRODUCT_STATUSES) if (!statusCounts.some((item) => item.status === status)) statusCounts.push({ status, label: status, count: items.filter((item) => item.saleStatus === status).length });
  const categories = array(payload.categories).map((value) => { const item = record(value) ? value : {}; return { categoryId: string(item.categoryId), categoryName: string(item.categoryName) }; }).filter((item) => item.categoryId);
  return { items, statusCounts, categories, calculatedAt: date(payload.calculatedAt) };
}
function mapSku(value: unknown): AdminProductSku {
  const item = record(value) ? value : {};
  return { skuId: string(item.skuId), skuCode: string(item.skuCode), label: string(item.label), weightGrams: typeof item.weightGrams === "number" && Number.isFinite(item.weightGrams) ? Math.max(0, item.weightGrams) : null, flavor: typeof item.flavor === "string" ? item.flavor : null, packageType: string(item.packageType), basePriceVnd: Math.max(0, number(item.basePriceVnd)), currency: "VND", saleStatus: skuStatus(item.saleStatus), saleStatusLabel: string(item.saleStatusLabel), updatedAt: date(item.updatedAt), revision: Math.max(1, number(item.revision, 1)) };
}
function mapDetail(payload: Record<string, unknown>): AdminProductDetail {
  const summary = mapSummary(payload);
  const food = record(payload.foodInformation) ? payload.foodInformation : null;
  return { ...summary, shortDescription: string(payload.shortDescription), longDescription: string(payload.longDescription), coverImageUrl: typeof payload.coverImageUrl === "string" ? payload.coverImageUrl : null, coverImageAlt: string(payload.coverImageAlt, summary.name), foodInformation: food ? { ingredients: string(food.ingredients), allergenStatement: typeof food.allergenStatement === "string" ? food.allergenStatement : null, storageInstructions: string(food.storageInstructions), manufacturingDatePolicy: string(food.manufacturingDatePolicy), shelfLifeDescription: string(food.shelfLifeDescription) } : null, skus: array(payload.skus).map(mapSku).filter((item) => item.skuId) };
}

export async function handleAdminCatalog(request: NextRequest, path: string[]) {
  const route = parseAdminCatalogPath(path);
  if (route.kind === "invalid") return responseError("NOT_FOUND", "Admin catalog route không tồn tại.", 404);
  if (request.method !== "GET") return responseError("METHOD_NOT_ALLOWED", "Method không được hỗ trợ.", 405);
  try {
    const currentSession = await session(request);
    if (!currentSession.permissions.includes("PRODUCT_VIEW")) return responseError("PERMISSION_FORBIDDEN", "Nhân viên không có quyền xem danh mục sản phẩm.", 403);
    if (route.kind === "product-list") return NextResponse.json(mapList(await upstream(request, "admin/catalog/products")));
    return NextResponse.json(mapDetail(await upstream(request, `admin/catalog/products/${encodeURIComponent(route.productId)}`)));
  } catch (cause) {
    if (cause instanceof UpstreamHttpError) return responseError(cause.body.code || "ADMIN_CATALOG_ERROR", cause.body.message || "Không thể xử lý yêu cầu danh mục sản phẩm.", cause.status >= 400 && cause.status <= 599 ? cause.status : 500, cause.body.errors ?? []);
    return responseError("ADMIN_CATALOG_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ danh mục sản phẩm nội bộ.", 503);
  }
}
