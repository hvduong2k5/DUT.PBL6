import { NextResponse, type NextRequest } from "next/server";
import { employeeAuthorizationFailure, requireEmployeeSession } from "@/features/employee/auth/server";
import { applyTrustedSessionCookie } from "@/lib/auth/server-credential";

const reportNames = new Set(["workbench", "velocity", "expiry-risk"]);
const headers = { "Cache-Control": "private, no-store" };

export async function GET(request: NextRequest, { params }: { params: Promise<{ report: string }> }) {
  const { report } = await params;
  const base = process.env.ANALYTICS_ADMIN_UPSTREAM_URL?.replace(/\/$/u, "");
  if (!base || !reportNames.has(report)) return NextResponse.json({ message: "Báo cáo không khả dụng." }, { status: 404, headers });

  try {
    const { permissions } = await requireEmployeeSession(request);
    if (!permissions.includes("INVENTORY_MANAGEMENT_VIEW") || !permissions.includes("BATCH_VIEW")) {
      return NextResponse.json({ message: "Bạn cần quyền xem quản lý tồn kho và lô hàng." }, { status: 403, headers });
    }
  } catch (cause) {
    const failure = employeeAuthorizationFailure(cause);
    return NextResponse.json({ message: failure.message }, { status: failure.status, headers });
  }

  try {
    const upstreamHeaders = new Headers({ Accept: "application/json" });
    applyTrustedSessionCookie(upstreamHeaders, request);
    const response = await fetch(`${base}/supply/${report}${request.nextUrl.search}`, {
      headers: upstreamHeaders,
      cache: "no-store",
      redirect: "manual",
      signal: AbortSignal.timeout(10_000),
    });
    if (response.ok) return NextResponse.json(await response.json(), { headers });
    if (response.status === 401 || response.status === 403) {
      return NextResponse.json({ message: "Tài khoản chưa có quyền truy cập Analytics pilot." }, { status: response.status, headers });
    }
    if (response.status === 400) {
      return NextResponse.json({ message: "Kiểm tra lại mã kho, khoảng ngày và số ngày cảnh báo." }, { status: 400, headers });
    }
    if (response.status === 429) {
      const retryAfter = response.headers.get("Retry-After");
      return NextResponse.json({ message: "Báo cáo đang nhận nhiều yêu cầu. Vui lòng thử lại sau." }, {
        status: 429,
        headers: retryAfter ? { ...headers, "Retry-After": retryAfter } : headers,
      });
    }
  } catch {
    // Use the same localized response for network and upstream failures.
  }
  return NextResponse.json({ message: "Chưa thể tải dữ liệu từ Analytics." }, { status: 503, headers });
}
