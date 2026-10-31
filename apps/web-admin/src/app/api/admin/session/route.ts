import { NextResponse } from "next/server";
import { parseAdminSession } from "@/lib/auth/types";

export const dynamic = "force-dynamic";

export async function GET() {
  const base = (process.env.ADMIN_API_UPSTREAM_URL ?? "http://127.0.0.1:4030/api/v1").replace(/\/$/u, "");
  const profile = process.env.NODE_ENV === "production" ? "" : process.env.ADMIN_MOCK_PROFILE ?? process.env.ADMIN_MOCK_ROLE ?? "SALES_MANAGER";
  const headers = new Headers({ Accept: "application/json" });
  if (profile) headers.set("X-Admin-Profile", profile);
  try {
    const response = await fetch(`${base}/admin/session`, { headers, cache: "no-store", redirect: "manual" });
    const payload: unknown = await response.json();
    if (!response.ok) return NextResponse.json(payload, { status: response.status });
    const session = parseAdminSession(payload);
    if (!session) return NextResponse.json({ code: "ADMIN_SESSION_INVALID", message: "Projection Role/Permission không hợp lệ." }, { status: 502 });
    return NextResponse.json(session);
  } catch { return NextResponse.json({ code: "ADMIN_SESSION_UNAVAILABLE", message: "Không thể kết nối dịch vụ phân quyền nội bộ." }, { status: 503 }); }
}
