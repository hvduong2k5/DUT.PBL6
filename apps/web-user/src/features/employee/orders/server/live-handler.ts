import { NextRequest, NextResponse } from "next/server";
import { usesIdentityBff } from "@/lib/auth/oidc-config";
import { applyTrustedSessionCookie, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { AdminOrderProjectionError, liveOrderDetail, liveOrderList } from "../live";
import { parseAdminOrderPath } from "../validation";

function error(status: number, code: string, message: string) {
  return NextResponse.json({ code, message, requestId: `BFF-ORDER-${crypto.randomUUID()}`, errors: [] }, { status, headers: { "Cache-Control": "no-store" } });
}

/** The owning API checks the server-held JWT. Browser role headers and fresh
 * configuration permissions do not replace the token's authorization snapshot. */
export async function handleLiveAdminOrders(request: NextRequest, path: string[]) {
  if (!usesIdentityBff()) return error(503, "ADMIN_AUTH_MODE_UNAVAILABLE", "Tra cứu nội bộ cần phiên đăng nhập Identity.");
  const route = parseAdminOrderPath(path);
  if (route.kind === "invalid") return error(404, "NOT_FOUND", "Đường dẫn đơn hàng không tồn tại.");
  if (request.method !== "GET") return error(405, "METHOD_NOT_ALLOWED", "Chức năng này chỉ hỗ trợ tra cứu.");
  if (route.kind === "order-detail" && !/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/u.test(route.orderId)) return error(404, "NOT_FOUND", "Đường dẫn đơn hàng không tồn tại.");
  if (!getTrustedAccessToken(request)) return error(401, "AUTH_REQUIRED", "Vui lòng đăng nhập bằng tài khoản nhân viên.");
  const base = process.env.IDENTITY_UPSTREAM_URL?.replace(/\/$/u, "");
  if (!base) return error(503, "ADMIN_ORDER_UPSTREAM_UNAVAILABLE", "Chưa thể kết nối dịch vụ đơn hàng.");
  const query = new URLSearchParams();
  for (const [name, value] of request.nextUrl.searchParams) {
    if (route.kind !== "order-list" || !["cursor", "limit", "status", "view"].includes(name) || query.has(name)) return error(400, "INVALID_QUERY", "Bộ lọc đơn hàng không hợp lệ.");
    query.set(name, value);
  }
  const view = query.get("view");
  if (view !== null && view !== "queue") return error(400, "INVALID_QUERY", "Bộ lọc đơn hàng không hợp lệ.");
  query.delete("view");
  const target = route.kind === "order-list" ? `orders${view ? "/queue" : ""}${query.size ? `?${query}` : ""}` : `orders/${encodeURIComponent(route.orderId)}`;
  const headers = new Headers({ Accept: "application/vnd.omama.admin-order+json" });
  applyTrustedSessionCookie(headers, request);
  try {
    const response = await fetch(`${base}/admin/order-api/${target}`, { headers, cache: "no-store", redirect: "manual", signal: AbortSignal.timeout(10000) });
    if (!response.ok) {
      if (response.status === 401) return error(401, "AUTH_REQUIRED", "Phiên đăng nhập đã hết hạn hoặc không hợp lệ.");
      if (response.status === 403) return error(403, "PERMISSION_FORBIDDEN", "Bạn chưa được cấp quyền tra cứu đơn hàng này.");
      if (response.status === 404) return error(404, "ORDER_NOT_FOUND", "Không tìm thấy đơn hàng trong phạm vi được cấp.");
      if (response.status === 400) return error(400, "INVALID_QUERY", "Bộ lọc đơn hàng không hợp lệ.");
      return error(response.status === 429 ? 429 : 503, "ADMIN_ORDER_UPSTREAM_UNAVAILABLE", "Chưa thể tải dữ liệu đơn hàng, vui lòng thử lại.");
    }
    if (!(response.headers.get("content-type") ?? "").includes("application/json")) throw new AdminOrderProjectionError();
    const payload: unknown = await response.json();
    const data = route.kind === "order-list" ? liveOrderList(payload) : liveOrderDetail(payload);
    return NextResponse.json(data, { headers: { "Cache-Control": "no-store" } });
  } catch (cause) {
    if (cause instanceof AdminOrderProjectionError || cause instanceof SyntaxError) return error(502, "ADMIN_ORDER_INVALID_RESPONSE", "Dữ liệu đơn hàng chưa thể hiển thị lúc này.");
    return error(503, "ADMIN_ORDER_UPSTREAM_UNAVAILABLE", "Chưa thể tải dữ liệu đơn hàng, vui lòng thử lại.");
  }
}
