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

async function fetchVerifiedSession(accessToken: string): Promise<AuthenticatedSession> {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  let response: Response;
  try {
    response = await fetch(`${base}/auth/me`, {
      headers: { Accept: "application/json", Authorization: `Bearer ${accessToken}` },
      cache: "no-store",
      redirect: "manual"
    });
  } catch {
    throw new CustomerAccessError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên đăng nhập lúc này.");
  }
  if (response.status === 401) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
  if (response.status === 403) throw new CustomerAccessError(403, "PRINCIPAL_FORBIDDEN", "Tài khoản không được phép sử dụng phiên hiện tại.");
  if (!response.ok) throw new CustomerAccessError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên đăng nhập lúc này.");
  const session = mapCustomerCoreSession(await response.json() as CustomerCoreUser);
  if (!session) throw new CustomerAccessError(503, "IDENTITY_INVALID", "Identity projection không đúng contract.");
  return session;
}

export async function requireCustomerSession(accessToken?: string): Promise<AuthenticatedSession> {
  if (!accessToken) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Vui lòng đăng nhập để sử dụng chức năng này.");
  const session = await fetchVerifiedSession(accessToken);
  if (!session.customerAccess || !session.customer || !session.principal.customerId) {
    throw new CustomerAccessError(403, "CUSTOMER_MAPPING_REQUIRED", "Tài khoản chưa được backend liên kết với Customer.");
  }
  return session;
}

export async function authorizeCustomerCapability(
  accessToken: string | undefined,
  capability: CustomerCapability
): Promise<{ session: AuthenticatedSession; projection: CustomerCapabilityProjection }> {
  const session = await requireCustomerSession(accessToken);
  let projection: CustomerCapabilityProjection;
  try {
    projection = await fetchCustomerCapabilities(accessToken);
  } catch (cause) {
    if (cause instanceof CapabilityProjectionError && cause.status === 401) {
      throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
    }
    if (cause instanceof CapabilityProjectionError && cause.status === 403) {
      throw new CustomerAccessError(403, "CAPABILITY_FORBIDDEN", "Tài khoản không có quyền sử dụng chức năng này.");
    }
    throw new CustomerAccessError(503, "CAPABILITY_UNAVAILABLE", "Chưa thể kiểm tra quyền Customer lúc này.");
  }
  if (!projection.capabilities.includes(capability)) {
    throw new CustomerAccessError(403, "CAPABILITY_FORBIDDEN", "Tài khoản không có quyền sử dụng chức năng này.");
  }
  return { session, projection };
}

export function customerAccessFailure(cause: unknown): { status: 401 | 403 | 503; code: string; message: string } {
  if (cause instanceof CustomerAccessError) return { status: cause.status, code: cause.code, message: cause.message };
  if (cause instanceof CapabilityProjectionError && cause.status === 401) {
    return { status: 401, code: "AUTH_REQUIRED", message: "Phiên đăng nhập không hợp lệ hoặc đã hết hạn." };
  }
  if (cause instanceof CapabilityProjectionError && cause.status === 403) {
    return { status: 403, code: "CAPABILITY_FORBIDDEN", message: "Tài khoản không có quyền sử dụng chức năng này." };
  }
  return { status: 503, code: "IDENTITY_UNAVAILABLE", message: "Chưa thể xác minh danh tính và quyền truy cập lúc này." };
}
