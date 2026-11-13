import type { NextRequest } from "next/server";
import { parseIdentityProfile, parseIdentitySession, type IdentityProfile } from "@/lib/auth/identity";
import { usesIdentityBff } from "@/lib/auth/oidc-config";
import { getTrustedAccessToken, getTrustedSessionCookie } from "@/lib/auth/server-credential";
import { toEmployeeSession, type EmployeeSession } from "./types";

export class EmployeeAuthorizationError extends Error {
  constructor(readonly status: 401 | 403 | 503, readonly code: string, message: string) {
    super(message);
    this.name = "EmployeeAuthorizationError";
  }
}

async function identity(path: string, sessionCookie: string): Promise<Response> {
  const base = (process.env.IDENTITY_UPSTREAM_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://127.0.0.1:8016/api/v1").replace(/\/$/u, "");
  try {
    return await fetch(`${base}${path}`, { headers: { Accept: "application/json", Cookie: `OMAMA_SESSION=${sessionCookie}` }, cache: "no-store", redirect: "manual" });
  } catch {
    throw new EmployeeAuthorizationError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên nhân viên lúc này.");
  }
}

async function identityProfileWithBearer(accessToken: string): Promise<Response> {
  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "").replace(/\/$/u, "");
  if (!base) throw new EmployeeAuthorizationError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên nhân viên lúc này.");
  try {
    return await fetch(`${base}/users/me`, { headers: { Accept: "application/json", Authorization: `Bearer ${accessToken}` }, cache: "no-store", redirect: "manual" });
  } catch {
    throw new EmployeeAuthorizationError(503, "IDENTITY_UNAVAILABLE", "Chưa thể xác minh phiên nhân viên lúc này.");
  }
}

function profileFailure(response: Response): EmployeeAuthorizationError {
  if (response.status === 401) return new EmployeeAuthorizationError(401, "AUTH_REQUIRED", "Không thể tải hồ sơ và quyền nhân viên.");
  if (response.status === 403) return new EmployeeAuthorizationError(403, "EMPLOYEE_FORBIDDEN", "Không thể tải hồ sơ và quyền nhân viên.");
  return new EmployeeAuthorizationError(503, "IDENTITY_UNAVAILABLE", "Không thể tải hồ sơ và quyền nhân viên.");
}

function projectEmployeeProfile(profile: IdentityProfile | undefined): EmployeeSession {
  if (!profile) throw new EmployeeAuthorizationError(503, "IDENTITY_PROFILE_INVALID", "Hồ sơ Identity không đúng contract.");
  if (profile.principalType !== "EMPLOYEE") throw new EmployeeAuthorizationError(403, "STAFF_ACCESS_REQUIRED", "Tài khoản Customer không được truy cập nghiệp vụ nội bộ.");
  if (profile.provisioningStatus !== "ACTIVE") throw new EmployeeAuthorizationError(403, "EMPLOYEE_FORBIDDEN", "Tài khoản nhân viên chưa ở trạng thái hoạt động.");
  const employeeSession = toEmployeeSession(profile);
  if (!employeeSession) throw new EmployeeAuthorizationError(503, "EMPLOYEE_SESSION_INVALID", "Không thể tạo projection quyền nhân viên.");
  return employeeSession;
}

export async function requireEmployeeSession(request: NextRequest): Promise<EmployeeSession> {
  if (!usesIdentityBff()) {
    // OIDC mode: the Keycloak access token in the HttpOnly cookie is the
    // credential. Identity verifies it and returns the employee profile;
    // there is no Identity session cookie to probe in this mode.
    const accessToken = getTrustedAccessToken(request);
    if (!accessToken) throw new EmployeeAuthorizationError(401, "AUTH_REQUIRED", "Vui lòng đăng nhập bằng tài khoản nhân viên.");
    const profileResponse = await identityProfileWithBearer(accessToken);
    if (!profileResponse.ok) throw profileFailure(profileResponse);
    return projectEmployeeProfile(parseIdentityProfile(await profileResponse.json()));
  }

  const sessionCookie = getTrustedSessionCookie(request);
  if (!sessionCookie) throw new EmployeeAuthorizationError(401, "AUTH_REQUIRED", "Vui lòng đăng nhập bằng tài khoản nhân viên.");

  const sessionResponse = await identity("/public/auth/session", sessionCookie);
  if (!sessionResponse.ok) throw new EmployeeAuthorizationError(sessionResponse.status === 401 ? 401 : 503, sessionResponse.status === 401 ? "AUTH_REQUIRED" : "IDENTITY_UNAVAILABLE", "Phiên nhân viên không hợp lệ hoặc Identity Service không khả dụng.");
  const session = parseIdentitySession(await sessionResponse.json());
  if (!session?.authenticated) throw new EmployeeAuthorizationError(401, "AUTH_REQUIRED", "Phiên nhân viên không hợp lệ hoặc đã hết hạn.");

  const profileResponse = await identity("/users/me", sessionCookie);
  if (!profileResponse.ok) throw profileFailure(profileResponse);
  return projectEmployeeProfile(parseIdentityProfile(await profileResponse.json()));
}

export function employeeAuthorizationFailure(cause: unknown) {
  return cause instanceof EmployeeAuthorizationError
    ? { status: cause.status, code: cause.code, message: cause.message }
    : { status: 503 as const, code: "IDENTITY_UNAVAILABLE", message: "Chưa thể xác minh quyền nhân viên lúc này." };
}
