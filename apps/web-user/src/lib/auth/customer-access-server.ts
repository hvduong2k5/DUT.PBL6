import type { CustomerCapability, CustomerCapabilityProjection } from "./capabilities";
import { CapabilityProjectionError, fetchCustomerCapabilities } from "./capability-server";
import { mapCustomerCoreSession, type CustomerCoreUser } from "./customer-core";
import type { AuthenticatedSession } from "./types";

export class CustomerAccessError extends Error {
  constructor(readonly status: 401 | 403 | 503, readonly code: string, message: string) {
    super(message);
    this.name = "CustomerAccessError";
  }
}

function identityBaseUrl(): string {
  return (process.env.IDENTITY_API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8016/api/v1").replace(/\/$/u, "");
}

function unwrap(value: unknown): unknown {
  return value && typeof value === "object" && !Array.isArray(value) && "data" in value ? (value as { data: unknown }).data : value;
}

export async function fetchVerifiedSession(sessionCookie: string): Promise<AuthenticatedSession> {
  const headers = { Accept: "application/json", Cookie: `OMAMA_SESSION=${sessionCookie}` };
  try {
    const sessionResponse = await fetch(`${identityBaseUrl()}/public/auth/session`, { headers, cache: "no-store", redirect: "manual" });
    if (sessionResponse.status === 401) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
    if (sessionResponse.status === 403) throw new CustomerAccessError(403, "PRINCIPAL_FORBIDDEN", "Tài khoản không được phép sử dụng phiên hiện tại.");
    if (!sessionResponse.ok) throw new CustomerAccessError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên đăng nhập lúc này.");
    const sessionProjection = unwrap(await sessionResponse.json()) as { isAuthenticated?: boolean; authenticated?: boolean };
    if (sessionProjection?.isAuthenticated !== true && sessionProjection?.authenticated !== true) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");

    const profileResponse = await fetch(`${identityBaseUrl()}/users/me`, { headers, cache: "no-store", redirect: "manual" });
    if (profileResponse.status === 401) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
    if (profileResponse.status === 403) throw new CustomerAccessError(403, "PRINCIPAL_FORBIDDEN", "Tài khoản không được phép sử dụng chức năng này.");
    if (!profileResponse.ok) throw new CustomerAccessError(503, "IDENTITY_UNAVAILABLE", "Chưa thể tải hồ sơ định danh lúc này.");
    const session = mapCustomerCoreSession(unwrap(await profileResponse.json()) as CustomerCoreUser);
    if (!session) throw new CustomerAccessError(503, "IDENTITY_INVALID", "Identity projection không đúng contract.");
    return session;
  } catch (cause) {
    if (cause instanceof CustomerAccessError) throw cause;
    throw new CustomerAccessError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên đăng nhập lúc này.");
  }
}

export async function requireCustomerSession(sessionCookie?: string): Promise<AuthenticatedSession> {
  if (!sessionCookie) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Vui lòng đăng nhập để sử dụng chức năng này.");
  const session = await fetchVerifiedSession(sessionCookie);
  if (!session.customerAccess || !session.customer || !session.principal.customerId) {
    throw new CustomerAccessError(403, "CUSTOMER_MAPPING_REQUIRED", "Tài khoản chưa được backend liên kết với Customer.");
  }
  return session;
}

export async function authorizeCustomerCapability(
  sessionCookie: string | undefined,
  capability: CustomerCapability
): Promise<{ session: AuthenticatedSession; projection: CustomerCapabilityProjection }> {
  const session = await requireCustomerSession(sessionCookie);
  let projection: CustomerCapabilityProjection;
  try {
    projection = await fetchCustomerCapabilities(sessionCookie);
  } catch (cause) {
    if (cause instanceof CapabilityProjectionError && cause.status === 401) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
    if (cause instanceof CapabilityProjectionError && cause.status === 403) throw new CustomerAccessError(403, "CAPABILITY_FORBIDDEN", "Tài khoản không có quyền sử dụng chức năng này.");
    throw new CustomerAccessError(503, "CAPABILITY_UNAVAILABLE", "Chưa thể kiểm tra quyền Customer lúc này.");
  }
  if (!projection.capabilities.includes(capability)) throw new CustomerAccessError(403, "CAPABILITY_FORBIDDEN", "Tài khoản không có quyền sử dụng chức năng này.");
  return { session, projection };
}

export function customerAccessFailure(cause: unknown): { status: 401 | 403 | 503; code: string; message: string } {
  if (cause instanceof CustomerAccessError) return { status: cause.status, code: cause.code, message: cause.message };
  if (cause instanceof CapabilityProjectionError && cause.status === 401) return { status: 401, code: "AUTH_REQUIRED", message: "Phiên đăng nhập không hợp lệ hoặc đã hết hạn." };
  if (cause instanceof CapabilityProjectionError && cause.status === 403) return { status: 403, code: "CAPABILITY_FORBIDDEN", message: "Tài khoản không có quyền sử dụng chức năng này." };
  return { status: 503, code: "IDENTITY_UNAVAILABLE", message: "Chưa thể xác minh danh tính và quyền truy cập lúc này." };
}
