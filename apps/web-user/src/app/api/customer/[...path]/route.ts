import {
  mapCustomerCoreAddress,
  mapCustomerCoreProfile,
  toCustomerCoreAddressInput,
  toCustomerCoreProfileUpdate,
  type CustomerCoreAddress,
  type CustomerCoreProfile
} from "@/lib/customer/customer-core";
import type { AddressInput, ProfileInput } from "@/lib/customer/types";
import { fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import type { CustomerCapability } from "@/lib/auth/capabilities";
import { parseCheckoutLocations } from "@/lib/checkout/locations";
import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const FIXED_ROUTES = new Set(["GET profile", "PATCH profile", "GET addresses", "POST addresses", "GET locations"]);
const ADDRESS_ROUTE = /^addresses\/[^/]+$/u;
const DEFAULT_ROUTE = /^addresses\/[^/]+\/default$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;

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
  } catch { return false; }
}

async function readBody<T>(request: NextRequest): Promise<T | undefined> {
  try { return await request.json() as T; } catch { return undefined; }
}

function headersFor(request: NextRequest, hasBody = false) {
  const headers = new Headers({ Accept: "application/json" });
  const cookie = request.headers.get("cookie");
  const authorization = request.headers.get("authorization");
  const accessToken = request.cookies.get("oma_access_token")?.value;
  const scenario = request.headers.get("x-mock-scenario");
  if (cookie) headers.set("Cookie", cookie);
  if (authorization) headers.set("Authorization", authorization);
  else if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (hasBody) headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

async function upstream(request: NextRequest, path: string, method: string, body?: unknown) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/${path}`, { method, headers: headersFor(request, body !== undefined), body: body === undefined ? undefined : JSON.stringify(body), cache: "no-store", redirect: "manual" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

async function locationUpstream(request: NextRequest) {
  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/locations/checkout`, { method: "GET", headers: headersFor(request), cache: "no-store", redirect: "manual" });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

function relayError(payload: unknown, status: number) {
  return Response.json(payload && typeof payload === "object" ? payload : { error_code: "ERR_CUSTOMER", user_message: "Không thể xử lý hồ sơ lúc này." }, { status });
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const method = request.method.toUpperCase();
  const parts = (await context.params).path;
  const path = parts.join("/");
  if (!isAllowed(method, path)) return Response.json({ code: "NOT_FOUND", message: "Customer route is not available." }, { status: 404 });
  if (method !== "GET" && !hasTrustedOrigin(request)) return Response.json({ code: "ORIGIN_NOT_ALLOWED", message: "Request origin is not allowed." }, { status: 403 });

  const accessToken = request.cookies.get("oma_access_token")?.value;
  const requiredCapability: CustomerCapability = path === "profile" ? "PROFILE_MANAGE" : "ADDRESS_MANAGE";
  if (!accessToken) return Response.json({ code: "AUTH_REQUIRED", message: "Vui lòng đăng nhập để quản lý tài khoản." }, { status: 401 });
  try {
    const projection = await fetchCustomerCapabilities(accessToken);
    if (!projection.capabilities.includes(requiredCapability)) return Response.json({ code: "CAPABILITY_FORBIDDEN", message: "Tài khoản không có quyền sử dụng chức năng này." }, { status: 403 });
  } catch {
    return Response.json({ code: "CAPABILITY_UNAVAILABLE", message: "Chưa thể kiểm tra quyền Customer lúc này." }, { status: 503 });
  }

  try {
    if (path === "locations") {
      const result = await locationUpstream(request);
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      const locations = parseCheckoutLocations(result.payload);
      if (!locations) return Response.json({ code: "CUSTOMER_LOCATIONS_INVALID", message: "Danh mục Tỉnh/Thành phố và Phường/Xã không hợp lệ." }, { status: 502 });
      return Response.json({ provinces: locations });
    }

    if (path === "profile") {
      const body = method === "PATCH" ? await readBody<ProfileInput>(request) : undefined;
      if (method === "PATCH" && (!body || typeof body.fullName !== "string" || typeof body.email !== "string")) return Response.json({ code: "INVALID_REQUEST", message: "Thông tin hồ sơ không hợp lệ." }, { status: 422 });
      const result = await upstream(request, "profile", method === "PATCH" ? "PUT" : "GET", body ? toCustomerCoreProfileUpdate(body) : undefined);
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      return Response.json(mapCustomerCoreProfile(result.payload as CustomerCoreProfile), { status: result.response.status });
    }

    if (path === "addresses") {
      if (method === "GET") {
        const result = await upstream(request, "profile/addresses", "GET");
        if (!result.response.ok) return relayError(result.payload, result.response.status);
        const addresses = (result.payload as { addresses?: CustomerCoreAddress[] }).addresses ?? [];
        return Response.json({ items: addresses.map(mapCustomerCoreAddress), total: addresses.length });
      }
      const body = await readBody<AddressInput>(request);
      if (!body) return Response.json({ code: "INVALID_REQUEST", message: "Địa chỉ không hợp lệ." }, { status: 422 });
      const result = await upstream(request, "profile/addresses", "POST", toCustomerCoreAddressInput(body));
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      return Response.json(mapCustomerCoreAddress(result.payload as CustomerCoreAddress), { status: result.response.status });
    }

    const addressId = parts[1];
    if (parts[2] === "default") {
      const listResult = await upstream(request, "profile/addresses", "GET");
      if (!listResult.response.ok) return relayError(listResult.payload, listResult.response.status);
      const addresses = (listResult.payload as { addresses?: CustomerCoreAddress[] }).addresses ?? [];
      const selected = addresses.find((item) => item.id === addressId);
      if (!selected) return Response.json({ error_code: "ERR_RESOURCE_NOT_FOUND", user_message: "Không tìm thấy địa chỉ yêu cầu." }, { status: 404 });
      const updateResult = await upstream(request, `profile/addresses/${encodeURIComponent(addressId)}`, "PUT", { ...selected, is_default: true, id: undefined });
      if (!updateResult.response.ok) return relayError(updateResult.payload, updateResult.response.status);
      return Response.json({ items: addresses.map((item) => mapCustomerCoreAddress({ ...item, is_default: item.id === addressId })), total: addresses.length });
    }

    if (method === "DELETE") {
      const result = await upstream(request, `profile/addresses/${encodeURIComponent(addressId)}`, "DELETE");
      if (!result.response.ok) return relayError(result.payload, result.response.status);
      return Response.json({ items: [], total: 0 });
    }

    const body = await readBody<AddressInput>(request);
    if (!body) return Response.json({ code: "INVALID_REQUEST", message: "Địa chỉ không hợp lệ." }, { status: 422 });
    const result = await upstream(request, `profile/addresses/${encodeURIComponent(addressId)}`, "PUT", toCustomerCoreAddressInput(body));
    if (!result.response.ok) return relayError(result.payload, result.response.status);
    return Response.json(mapCustomerCoreAddress(result.payload as CustomerCoreAddress));
  } catch {
    return Response.json({ code: "CUSTOMER_UPSTREAM_UNAVAILABLE", message: "Không thể kết nối dịch vụ hồ sơ. Hãy kiểm tra Mockoon customer API.", requestId: "BFF-CUSTOMER-UPSTREAM" }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
export const PATCH = proxy;
export const PUT = proxy;
export const DELETE = proxy;
