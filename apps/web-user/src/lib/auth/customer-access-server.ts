import type { CustomerCapability, CustomerCapabilityProjection } from "./capabilities";
import { CapabilityProjectionError, fetchCustomerCapabilities } from "./capability-server";
import { parseIdentityProfile, parseIdentitySession } from "./identity";
import type { AuthenticatedSession } from "./types";

export class CustomerAccessError extends Error {
  constructor(readonly status: 401 | 403 | 503, readonly code: string, message: string) {
    super(message);
    this.name = "CustomerAccessError";
  }
}

async function fetchIdentity(path: string, sessionCookie: string): Promise<Response> {
  const base = (process.env.IDENTITY_UPSTREAM_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://127.0.0.1:8016/api/v1").replace(/\/$/u, "");
  try {
    return await fetch(`${base}${path}`, {
      headers: { Accept: "application/json", Cookie: `OMAMA_SESSION=${sessionCookie}` },
      cache: "no-store",
      redirect: "manual"
    });
  } catch {
    throw new CustomerAccessError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên đăng nhập lúc này.");
  }
}

async function fetchVerifiedCustomerSession(sessionCookie: string): Promise<AuthenticatedSession> {
  const sessionResponse = await fetchIdentity("/public/auth/session", sessionCookie);
  if (sessionResponse.status === 401) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
  if (!sessionResponse.ok) throw new CustomerAccessError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên đăng nhập lúc này.");
  const session = parseIdentitySession(await sessionResponse.json());
  if (!session) throw new CustomerAccessError(503, "IDENTITY_INVALID", "Identity session không đúng contract.");
  if (!session.authenticated) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");

  const profileResponse = await fetchIdentity("/users/me", sessionCookie);
  if (profileResponse.status === 401) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
  if (profileResponse.status === 403) throw new CustomerAccessError(403, "PRINCIPAL_FORBIDDEN", "Tài khoản không được phép sử dụng phiên hiện tại.");
  if (!profileResponse.ok) throw new CustomerAccessError(503, "IDENTITY_UNAVAILABLE", "Chưa thể tải hồ sơ định danh lúc này.");
  const profile = parseIdentityProfile(await profileResponse.json());
  if (!profile) throw new CustomerAccessError(503, "IDENTITY_INVALID", "Identity profile không đúng contract.");
  if (profile.principalType === "EMPLOYEE") {
    throw new CustomerAccessError(403, "EMPLOYEE_CUSTOMER_ACCESS_FORBIDDEN", "Tài khoản nhân viên không được sử dụng chức năng mua hàng.");
  }

  return {
    authenticated: true,
    principal: {
      principalId: profile.principalId,
      principalType: "CUSTOMER",
      displayName: profile.displayName,
      email: profile.email,
      roles: profile.roles,
      permissions: profile.permissions,
      customerId: profile.principalId
    },
    customerAccess: true,
    customer: { id: profile.principalId, displayName: profile.displayName, email: profile.email, status: "ACTIVE" }
  };
}

export async function requireCustomerSession(sessionCookie?: string): Promise<AuthenticatedSession> {
  if (!sessionCookie) throw new CustomerAccessError(401, "AUTH_REQUIRED", "Vui lòng đăng nhập để sử dụng chức năng này.");
  return fetchVerifiedCustomerSession(sessionCookie);
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
