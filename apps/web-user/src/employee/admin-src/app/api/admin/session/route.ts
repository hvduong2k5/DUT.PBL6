import { NextRequest, NextResponse } from "next/server";
import { getTrustedSessionCookie } from "@/lib/auth/server-credential";
import { applyAdminMockProfile, getAdminIdentityFailure } from "@/lib/auth/admin-access-server";
import { parseAdminSession } from "@admin/lib/auth/types";

export const dynamic = "force-dynamic";

export async function handleAdminSession(request: NextRequest) {
  const sessionCookie = getTrustedSessionCookie(request);
  if (!sessionCookie) return NextResponse.json({ code: "AUTH_REQUIRED", message: "Vui lòng đăng nhập." }, { status: 401 });
  const identityFailure = await getAdminIdentityFailure(request);
  if (identityFailure) return NextResponse.json(identityFailure, { status: identityFailure.status });
  const adminBase = (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, "");
  const adminHeaders = new Headers({ Accept: "application/json", Cookie: `OMAMA_SESSION=${sessionCookie}` });
  applyAdminMockProfile(adminHeaders);

  try {
    const response = await fetch(`${adminBase}/admin/session`, { headers: adminHeaders, cache: "no-store", redirect: "manual" });
    const payload: unknown = await response.json();
    if (!response.ok) return NextResponse.json(payload, { status: response.status });
    const session = parseAdminSession(payload);
    if (!session) return NextResponse.json({ code: "ADMIN_SESSION_INVALID", message: "Projection Role/Permission không hợp lệ." }, { status: 502 });
    return NextResponse.json(session);
  } catch {
    return NextResponse.json({ code: "ADMIN_SESSION_UNAVAILABLE", message: "Không thể kết nối dịch vụ danh tính hoặc phân quyền nội bộ." }, { status: 503 });
  }
}
