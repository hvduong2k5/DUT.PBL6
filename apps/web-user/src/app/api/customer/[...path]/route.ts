import { hasTrustedOrigin } from "@/lib/auth/request-origin";
import {
  normalizePublicError,
  parseStrongVersionEtag,
  type AddressUpdate,
  type AddressWrite,
  type Profile,
  type ProfileUpdate,
  type PublicErrorBody,
  type ShippingAddress
} from "@/lib/api-contract";
import type { CustomerCapability } from "@/lib/auth/capabilities";
import { authorizeCustomerCapability, customerAccessFailure } from "@/lib/auth/customer-access-server";
import { applyTrustedAuthorization, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { parseCheckoutLocations } from "@/lib/checkout/locations";
import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const FIXED_ROUTES = new Set(["GET profile", "PUT profile", "GET addresses", "POST addresses", "GET locations"]);
const ADDRESS_ROUTE = /^addresses\/[^/]+$/u;
const DEFAULT_ROUTE = /^addresses\/[^/]+\/default$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const DEVELOPMENT_TIMESTAMP = "2026-09-26T00:00:00.000Z";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function text(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function positiveVersion(value: unknown): number | undefined {
  return typeof value === "number" && Number.isSafeInteger(value) && value > 0 ? value : undefined;
}

function isAllowed(method: string, path: string): boolean {
  if (FIXED_ROUTES.has(`${method} ${path}`)) return true;
  if (ADDRESS_ROUTE.test(path)) return method === "GET" || method === "PUT" || method === "DELETE";
  return DEFAULT_ROUTE.test(path) && (method === "PUT" || method === "PATCH");
}



async function readBody<T>(request: NextRequest): Promise<T | undefined> {
  try {
    return await request.json() as T;
  } catch {
    return undefined;
  }
}

function headersFor(request: NextRequest, hasBody = false): Headers {
  const headers = new Headers({ Accept: "application/json" });
  const scenario = request.headers.get("x-mock-scenario");
  const ifMatch = request.headers.get("if-match");
  applyTrustedAuthorization(headers, request);
  if (ifMatch) headers.set("If-Match", ifMatch);
  if (hasBody) headers.set("Content-Type", "application/json");
  if (process.env.NODE_ENV !== "production" && scenario && SCENARIO_PATTERN.test(scenario)) {
    headers.set("X-Mock-Scenario", scenario);
  }
  return headers;
}

async function upstream(request: NextRequest, path: string, method: string, body?: unknown) {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/${path}`, {
    method,
    headers: headersFor(request, body !== undefined),
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

async function locationUpstream(request: NextRequest) {
  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const response = await fetch(`${base}/locations/checkout`, {
    method: "GET",
    headers: headersFor(request),
    cache: "no-store",
    redirect: "manual"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() as unknown : undefined;
  return { response, payload };
}

function responseHeaders(response: Response, version?: number): Headers {
  const headers = new Headers();
  for (const name of ["cache-control", "etag", "location", "retry-after", "x-request-id"]) {
    const value = response.headers.get(name);
    if (value) headers.set(name, value);
  }
  if (!headers.has("etag") && version && process.env.NODE_ENV !== "production") headers.set("ETag", `"${version}"`);
  return headers;
}

function publicErrorResponse(status: number, code: string, message: string, requestId: string, details?: Record<string, unknown>): Response {
  const body: PublicErrorBody = { code, message, requestId, ...(details ? { details } : {}) };
  return Response.json(body, { status });
}

function relayError(payload: unknown, response: Response): Response {
  const error = normalizePublicError(payload, {
    code: "PROFILE_UPSTREAM_ERROR",
    message: "Không thể xử lý hồ sơ lúc này.",
    requestId: response.headers.get("x-request-id") ?? "BFF-PROFILE-UPSTREAM"
  });
  return Response.json(error, { status: response.status, headers: responseHeaders(response) });
}

function codeFromName(value: string, level: "province" | "ward"): string {
  const normalized = value.normalize("NFD").replace(/[\u0300-\u036f]/gu, "").replace(/đ/giu, "d").toUpperCase().replace(/[^A-Z0-9]+/gu, "-").replace(/^-|-$/gu, "");
  if (level === "province" && normalized.includes("HUE")) return "HUE";
  if (level === "ward" && normalized.includes("THUAN-HOA")) return "THUAN-HOA";
  if (level === "province" && normalized.includes("DA-NANG")) return "DA-NANG";
  return level === "ward" ? normalized.replace(/^(?:PHUONG|XA|THI-TRAN)-/u, "") : normalized;
}

function normalizeProfile(payload: unknown, response: Response): Profile | undefined {
  if (!isRecord(payload)) return undefined;
  const version = positiveVersion(payload.version) ?? parseStrongVersionEtag(response.headers.get("etag"));
  if (text(payload.full_name) && (text(payload.id) || text(payload.user_id)) && version && text(payload.updated_at ?? payload.created_at)) {
    return { ...payload, version } as Profile;
  }
  if (process.env.NODE_ENV === "production" || !text(payload.customer_id) || !text(payload.full_name)) return undefined;
  return {
    id: text(payload.customer_id),
    user_id: text(payload.customer_id),
    full_name: text(payload.full_name),
    phone_number: text(payload.phone_number) ?? "",
    email: text(payload.email) ?? "",
    avatar_url: text(payload.avatar_url) ?? "",
    version: version ?? 1,
    created_at: DEVELOPMENT_TIMESTAMP,
    updated_at: DEVELOPMENT_TIMESTAMP
  };
}

function normalizeAddress(payload: unknown, response?: Response): ShippingAddress | undefined {
  if (!isRecord(payload)) return undefined;
  const version = positiveVersion(payload.version) ?? parseStrongVersionEtag(response?.headers.get("etag"));
  const canonical = text(payload.id)
    && text(payload.recipient_name)
    && text(payload.phone_number)
    && text(payload.street_address)
    && text(payload.ward_code)
    && text(payload.province_code)
    && text(payload.ward_name)
    && text(payload.province_name)
    && version
    && text(payload.updated_at ?? payload.created_at);
  if (canonical) return { ...payload, version } as ShippingAddress;
  if (process.env.NODE_ENV === "production") return undefined;
  const wardName = text(payload.ward);
  const provinceName = text(payload.province);
  if (!text(payload.id) || !text(payload.recipient_name) || !text(payload.phone_number) || !text(payload.street_address) || !wardName || !provinceName) {
    return undefined;
  }
  return {
    id: text(payload.id),
    customer_id: text(payload.customer_id) ?? "DEV-CUSTOMER",
    recipient_name: text(payload.recipient_name),
    phone_number: text(payload.phone_number),
    street_address: text(payload.street_address),
    ward_code: codeFromName(wardName, "ward"),
    province_code: codeFromName(provinceName, "province"),
    ward_name: wardName,
    province_name: provinceName,
    label: "HOME",
    is_default: payload.is_default === true,
    is_deleted: false,
    version: version ?? 1,
    created_at: DEVELOPMENT_TIMESTAMP,
    updated_at: DEVELOPMENT_TIMESTAMP
  };
}

function normalizeAddressList(payload: unknown, response?: Response): ShippingAddress[] | undefined {
  const source = Array.isArray(payload)
    ? payload
    : process.env.NODE_ENV !== "production" && isRecord(payload) && Array.isArray(payload.addresses)
      ? payload.addresses
      : undefined;
  if (!source) return undefined;
  const addresses = source.map((item) => normalizeAddress(item, response));
  return addresses.every((item): item is ShippingAddress => Boolean(item)) ? addresses : undefined;
}

function isProfileUpdate(value: unknown): value is ProfileUpdate {
  return isRecord(value) && typeof value.full_name === "string" && typeof value.phone_number === "string";
}

function isAddressInput(value: unknown): value is AddressWrite | AddressUpdate {
  return isRecord(value)
    && typeof value.recipient_name === "string"
    && typeof value.phone_number === "string"
    && typeof value.street_address === "string"
    && typeof value.ward_code === "string"
    && typeof value.province_code === "string";
}

function addressUpdateFrom(address: ShippingAddress): AddressUpdate {
  return {
    recipient_name: address.recipient_name ?? "",
    phone_number: address.phone_number ?? "",
    street_address: address.street_address ?? "",
    ward_code: address.ward_code ?? "",
    province_code: address.province_code ?? "",
    ward_name: address.ward_name,
    province_name: address.province_name,
    latitude: address.latitude,
    longitude: address.longitude,
    label: address.label
  };
}

async function developmentAddressList(request: NextRequest): Promise<{ response: Response; addresses: ShippingAddress[] } | undefined> {
  if (process.env.NODE_ENV === "production") return undefined;
  const result = await upstream(request, "profile/addresses", "GET");
  if (!result.response.ok) return undefined;
  const addresses = normalizeAddressList(result.payload, result.response);
  return addresses ? { response: result.response, addresses } : undefined;
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const method = request.method.toUpperCase();
  const parts = (await context.params).path;
  const path = parts.join("/");
  if (!isAllowed(method, path)) return publicErrorResponse(404, "NOT_FOUND", "Customer route is not available.", "BFF-CUSTOMER-ROUTE");
  if (method !== "GET" && !hasTrustedOrigin(request)) {
    return publicErrorResponse(403, "ORIGIN_NOT_ALLOWED", "Request origin is not allowed.", "BFF-CUSTOMER-ORIGIN");
  }

  const accessToken = getTrustedAccessToken(request);
  const requiredCapability: CustomerCapability = path === "profile" ? "PROFILE_MANAGE" : "ADDRESS_MANAGE";
  try {
    await authorizeCustomerCapability(accessToken, requiredCapability);
  } catch (cause) {
    const failure = customerAccessFailure(cause);
    return publicErrorResponse(failure.status, failure.code, failure.message, "BFF-CUSTOMER-ACCESS");
  }

  try {
    if (path === "locations") {
      const result = await locationUpstream(request);
      if (!result.response.ok) return relayError(result.payload, result.response);
      const locations = parseCheckoutLocations(result.payload);
      if (!locations) return publicErrorResponse(502, "CUSTOMER_LOCATIONS_INVALID", "Danh mục Tỉnh/Thành phố và Phường/Xã không hợp lệ.", "BFF-CUSTOMER-LOCATIONS");
      return Response.json({ provinces: locations });
    }

    if (path === "profile") {
      const body = method === "PUT" ? await readBody<unknown>(request) : undefined;
      if (method === "PUT" && !isProfileUpdate(body)) {
        return publicErrorResponse(400, "INVALID_REQUEST", "Thông tin hồ sơ không hợp lệ.", "BFF-PROFILE-REQUEST");
      }
      const result = await upstream(request, "profile", method, body);
      if (!result.response.ok) return relayError(result.payload, result.response);
      const profile = normalizeProfile(result.payload, result.response);
      if (!profile) return publicErrorResponse(502, "INVALID_PROFILE_RESPONSE", "Dữ liệu hồ sơ không đúng contract.", "BFF-PROFILE-RESPONSE");
      return Response.json(profile, { status: result.response.status, headers: responseHeaders(result.response, profile.version) });
    }

    if (path === "addresses") {
      if (method === "GET") {
        const result = await upstream(request, "profile/addresses", "GET");
        if (!result.response.ok) return relayError(result.payload, result.response);
        const addresses = normalizeAddressList(result.payload, result.response);
        if (!addresses) return publicErrorResponse(502, "INVALID_ADDRESS_RESPONSE", "Danh sách địa chỉ không đúng contract.", "BFF-ADDRESS-LIST");
        return Response.json(addresses, { headers: responseHeaders(result.response) });
      }
      const body = await readBody<unknown>(request);
      if (!isAddressInput(body)) return publicErrorResponse(400, "INVALID_REQUEST", "Địa chỉ không hợp lệ.", "BFF-ADDRESS-CREATE");
      const result = await upstream(request, "profile/addresses", "POST", body);
      if (!result.response.ok) return relayError(result.payload, result.response);
      const address = normalizeAddress(result.payload, result.response);
      if (!address) return publicErrorResponse(502, "INVALID_ADDRESS_RESPONSE", "Địa chỉ trả về không đúng contract.", "BFF-ADDRESS-CREATE");
      return Response.json(address, { status: result.response.status, headers: responseHeaders(result.response, address.version) });
    }

    const addressId = parts[1];
    if (parts[2] === "default") {
      const result = await upstream(request, `profile/addresses/${encodeURIComponent(addressId)}/default`, method);
      if (result.response.ok) {
        const addresses = normalizeAddressList(result.payload, result.response);
        if (!addresses) return publicErrorResponse(502, "INVALID_ADDRESS_RESPONSE", "Danh sách địa chỉ không đúng contract.", "BFF-ADDRESS-DEFAULT");
        return Response.json(addresses, { headers: responseHeaders(result.response) });
      }
      if (process.env.NODE_ENV === "production" || result.response.status !== 404) return relayError(result.payload, result.response);

      const legacy = await developmentAddressList(request);
      const selected = legacy?.addresses.find((item) => item.id === addressId);
      if (!legacy || !selected) return publicErrorResponse(404, "ADDRESS_NOT_FOUND", "Không tìm thấy địa chỉ yêu cầu.", "BFF-ADDRESS-DEFAULT");
      const update = await upstream(request, `profile/addresses/${encodeURIComponent(addressId)}`, "PUT", addressUpdateFrom(selected));
      if (!update.response.ok) return relayError(update.payload, update.response);
      return Response.json(legacy.addresses.map((item) => ({ ...item, is_default: item.id === addressId })));
    }

    if (method === "GET") {
      const result = await upstream(request, `profile/addresses/${encodeURIComponent(addressId)}`, "GET");
      if (result.response.ok) {
        const address = normalizeAddress(result.payload, result.response);
        if (!address) return publicErrorResponse(502, "INVALID_ADDRESS_RESPONSE", "Địa chỉ trả về không đúng contract.", "BFF-ADDRESS-DETAIL");
        return Response.json(address, { headers: responseHeaders(result.response, address.version) });
      }
      if (process.env.NODE_ENV === "production" || result.response.status !== 404) return relayError(result.payload, result.response);
      const legacy = await developmentAddressList(request);
      const address = legacy?.addresses.find((item) => item.id === addressId);
      return address
        ? Response.json(address, { headers: { ETag: `"${address.version}"` } })
        : publicErrorResponse(404, "ADDRESS_NOT_FOUND", "Không tìm thấy địa chỉ yêu cầu.", "BFF-ADDRESS-DETAIL");
    }

    if (method === "DELETE") {
      const result = await upstream(request, `profile/addresses/${encodeURIComponent(addressId)}`, "DELETE");
      if (!result.response.ok) return relayError(result.payload, result.response);
      const addresses = normalizeAddressList(result.payload, result.response);
      if (addresses) return Response.json(addresses, { headers: responseHeaders(result.response) });
      const legacy = await developmentAddressList(request);
      if (!legacy) return publicErrorResponse(502, "INVALID_ADDRESS_RESPONSE", "Danh sách địa chỉ không đúng contract.", "BFF-ADDRESS-DELETE");
      return Response.json(legacy.addresses.filter((item) => item.id !== addressId));
    }

    const body = await readBody<unknown>(request);
    if (!isAddressInput(body)) return publicErrorResponse(400, "INVALID_REQUEST", "Địa chỉ không hợp lệ.", "BFF-ADDRESS-UPDATE");
    const result = await upstream(request, `profile/addresses/${encodeURIComponent(addressId)}`, "PUT", body);
    if (!result.response.ok) return relayError(result.payload, result.response);
    const address = normalizeAddress(result.payload, result.response);
    if (!address) return publicErrorResponse(502, "INVALID_ADDRESS_RESPONSE", "Địa chỉ trả về không đúng contract.", "BFF-ADDRESS-UPDATE");
    return Response.json(address, { headers: responseHeaders(result.response, address.version) });
  } catch {
    return publicErrorResponse(503, "CUSTOMER_UPSTREAM_UNAVAILABLE", "Không thể kết nối dịch vụ hồ sơ. Hãy kiểm tra API Gateway hoặc Mockoon.", "BFF-CUSTOMER-UPSTREAM");
  }
}

export const GET = proxy;
export const POST = proxy;
export const PATCH = proxy;
export const PUT = proxy;
export const DELETE = proxy;
