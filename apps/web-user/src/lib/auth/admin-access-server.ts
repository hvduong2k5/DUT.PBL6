import type { NextRequest } from "next/server";
import { getTrustedAccessToken } from "@/lib/auth/server-credential";
import { mapCustomerCoreSession, type CustomerCoreUser } from "@/lib/auth/customer-core";

export interface AdminIdentityFailure {
  status: 401 | 403 | 502 | 503;
  code: string;
  message: string;
}

export async function getAdminIdentityFailure(request: NextRequest): Promise<AdminIdentityFailure | undefined> {
  const accessToken = getTrustedAccessToken(request);
  if (!accessToken) return { status: 401, code: "AUTH_REQUIRED", message: "Vui lòng đăng nhập." };

  const base = (process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u, "");
  try {
    const response = await fetch(`${base}/auth/me`, {
      headers: { Accept: "application/json", Authorization: `Bearer ${accessToken}` },
      cache: "no-store",
      redirect: "manual"
    });
    if (response.status === 401) return { status: 401, code: "AUTH_REQUIRED", message: "Phiên đăng nhập không hợp lệ hoặc đã hết hạn." };
    if (response.status === 403) return { status: 403, code: "PRINCIPAL_FORBIDDEN", message: "Tài khoản không được phép sử dụng phiên hiện tại." };
    if (!response.ok) return { status: 503, code: "IDENTITY_UNAVAILABLE", message: "Chưa thể xác minh phiên đăng nhập." };
    const identity = mapCustomerCoreSession(await response.json() as CustomerCoreUser);
    if (!identity) return { status: 502, code: "IDENTITY_INVALID", message: "Identity projection không đúng contract." };
    if (identity.principal.principalType !== "EMPLOYEE") return { status: 403, code: "EMPLOYEE_REQUIRED", message: "Chức năng này dành cho nhân viên nội bộ." };
    return undefined;
  } catch {
    return { status: 503, code: "IDENTITY_UNAVAILABLE", message: "Chưa thể xác minh phiên đăng nhập." };
  }
}

export function applyAdminMockProfile(headers: Headers): void {
  const profile = process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER";
  if (profile) headers.set("X-Admin-Profile", profile);
}
