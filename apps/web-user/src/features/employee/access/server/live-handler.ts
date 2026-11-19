import { NextRequest, NextResponse } from "next/server";
import { usesIdentityBff } from "@/lib/auth/oidc-config";
import { applyTrustedSessionCookie, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { AccessProjectionError, decodeAccessProjection } from "../live";
import { parseAdminAccessPath } from "../validation";
import { AccessUpstreamError, composeAccessProjection } from "./compose";
import { handleLiveAdminAccessWrite } from "./write-handler";

const failure = (status: number, code: string, message: string) => NextResponse.json({ code, message, requestId: `BFF-ACCESS-${crypto.randomUUID()}`, errors: [] }, { status, headers: { "Cache-Control": "no-store" } });

export async function handleLiveAdminAccess(request: NextRequest, path: string[]) {
  if (path.some((part) => !/^[A-Za-z0-9_.:-]{1,160}$/u.test(part))) return failure(404, "NOT_FOUND", "Đường dẫn quản trị không tồn tại.");
  if (request.method !== "GET") {
    // Writes exist only in OIDC mode, where the BFF can forward the caller's
    // staff token to the identity admin APIs (F-47). The identity-bff fixture
    // mode has no write surface.
    if (usesIdentityBff()) return failure(405, "METHOD_NOT_ALLOWED", "Chức năng này chỉ hỗ trợ tra cứu.");
    return handleLiveAdminAccessWrite(request, path);
  }
  const route = parseAdminAccessPath(path);
  if (route.kind === "invalid") return failure(404, "NOT_FOUND", "Đường dẫn quản trị không tồn tại.");
  if (!getTrustedAccessToken(request)) return failure(401, "AUTH_REQUIRED", "Vui lòng đăng nhập bằng tài khoản nhân viên.");
  if (request.nextUrl.search) return failure(400, "INVALID_QUERY", "Bộ lọc chưa được hỗ trợ.");
  const base = process.env.IDENTITY_UPSTREAM_URL?.replace(/\/$/u, "");
  if (!base) return failure(503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
  if (!usesIdentityBff()) {
    // OIDC mode: no service implements the iam-admin-read.v1 projection, so the
    // BFF composes it from the real identity/profile admin APIs (see compose.ts).
    const token = getTrustedAccessToken(request);
    if (!token) return failure(401, "AUTH_REQUIRED", "Vui lòng đăng nhập bằng tài khoản nhân viên.");
    const target = route.kind === "detail" ? route.employeeId : route.kind === "role-detail" ? route.roleCode : route.kind === "permission-detail" ? route.permissionCode : undefined;
    try {
      const envelope = await composeAccessProjection(route.kind, target, base, token);
      const data = decodeAccessProjection(route.kind, envelope);
      return NextResponse.json(data, { headers: { "Cache-Control": "no-store" } });
    } catch (cause) {
      if (cause instanceof AccessUpstreamError) {
        if (cause.status === 401) return failure(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
        if (cause.status === 403) return failure(403, "PERMISSION_FORBIDDEN", "Bạn chưa được cấp quyền tra cứu nội dung này.");
        if (cause.status === 404) return failure(404, "NOT_FOUND", "Không tìm thấy dữ liệu quản trị truy cập.");
        return failure(cause.status === 429 ? 429 : 503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
      }
      if (cause instanceof AccessProjectionError || cause instanceof SyntaxError) return failure(502, "ADMIN_ACCESS_INVALID_RESPONSE", "Dữ liệu quản trị truy cập chưa thể hiển thị lúc này.");
      return failure(503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
    }
  }
  const headers = new Headers({ Accept: "application/vnd.omama.iam-admin-read.v1+json" });
  applyTrustedSessionCookie(headers, request);
  try {
    const response = await fetch(`${base}/admin/access/${path.map(encodeURIComponent).join("/")}`, { headers, cache: "no-store", redirect: "manual", signal: AbortSignal.timeout(10000) });
    if (!response.ok) {
      if (response.status === 401) return failure(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
      if (response.status === 403) return failure(403, "PERMISSION_FORBIDDEN", "Bạn chưa được cấp quyền tra cứu nội dung này.");
      // Missing projection support and unknown resources are distinguished by the producer.
      if (response.status === 404 && response.headers.get("X-IAM-Read-Contract") === "iam-admin-read.v1") return failure(404, "NOT_FOUND", "Không tìm thấy dữ liệu quản trị truy cập.");
      return failure(response.status === 429 ? 429 : 503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
    }
    if (!(response.headers.get("content-type") ?? "").includes("json")) throw new AccessProjectionError();
    const data = decodeAccessProjection(route.kind, await response.json());
    return NextResponse.json(data, { headers: { "Cache-Control": "no-store" } });
  } catch (cause) {
    if (cause instanceof AccessProjectionError || cause instanceof SyntaxError) return failure(502, "ADMIN_ACCESS_INVALID_RESPONSE", "Dữ liệu quản trị truy cập chưa thể hiển thị lúc này.");
    return failure(503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
  }
}
