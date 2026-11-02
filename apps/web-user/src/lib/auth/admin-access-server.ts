import type { NextRequest } from "next/server";
import { getTrustedSessionCookie } from "@/lib/auth/server-credential";
import { CustomerAccessError, fetchVerifiedSession } from "@/lib/auth/customer-access-server";

export interface AdminIdentityFailure {
  status: 401 | 403 | 502 | 503;
  code: string;
  message: string;
}

export async function getAdminIdentityFailure(request: NextRequest): Promise<AdminIdentityFailure | undefined> {
  const sessionCookie = getTrustedSessionCookie(request);
  if (!sessionCookie) return { status: 401, code: "AUTH_REQUIRED", message: "Vui lòng đăng nhập." };
  try {
    const identity = await fetchVerifiedSession(sessionCookie);
    if (identity.principal.principalType !== "EMPLOYEE") return { status: 403, code: "EMPLOYEE_REQUIRED", message: "Chức năng này dành cho nhân viên nội bộ." };
    return undefined;
  } catch (cause) {
    if (cause instanceof CustomerAccessError) return { status: cause.status, code: cause.code, message: cause.message };
    return { status: 503, code: "IDENTITY_UNAVAILABLE", message: "Chưa thể xác minh phiên đăng nhập." };
  }
}

export function applyAdminMockProfile(headers: Headers): void {
  const profile = process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER";
  if (profile) headers.set("X-Admin-Profile", profile);
}
