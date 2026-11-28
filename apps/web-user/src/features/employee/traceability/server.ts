import { readBoundedText } from "@/lib/server/read-bounded-text";
export { readBoundedText } from "@/lib/server/read-bounded-text";
import { NextRequest, NextResponse } from "next/server";
import { EmployeeAuthorizationError, requireEmployeeSession } from "@/features/employee/auth/server";
import { credentialHeaders, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { publicWebOrigin, usesIdentityBff } from "@/lib/auth/oidc-config";

const ID = "[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}";
const KEY = /^[!-~]{16,128}$/u;
export function tracePermission(method: string, segments: string[]): string | undefined {
  const path = segments.join("/");
  if (method === "GET" && new RegExp(`^records(?:/by-batch|/${ID})?$|^batches/${ID}/context$`, "u").test(path)) return "traceability.read_internal";
  if (method === "POST" && path === "records") return "traceability.create";
  if (method === "PATCH" && new RegExp(`^records/${ID}/revisions/[1-9][0-9]{0,8}$`, "u").test(path)) return "traceability.edit";
  if (method === "POST" && new RegExp(`^records/${ID}/revisions$`, "u").test(path)) return "traceability.edit";
  const revision = new RegExp(`^records/${ID}/revisions/[1-9][0-9]{0,8}/(submit|approve|reject|publish)$`, "u").exec(path);
  if (method === "POST" && revision) return `traceability.${revision[1] === "reject" ? "approve" : revision[1]}`;
  const root = new RegExp(`^records/${ID}/(suspend|archive)$`, "u").exec(path);
  return method === "POST" && root ? `traceability.${root[1]}` : undefined;
}
function fail(status: number, code: string, message: string, requestId = crypto.randomUUID()) {
  return NextResponse.json({ code, message, requestId }, { status, headers: { "Cache-Control": "no-store", "X-Request-Id": requestId } });
}
export async function handleTraceability(request: NextRequest, parts: string[]) {
  const permission = tracePermission(request.method, parts);
  if (!permission) return fail(404, "NOT_FOUND", "Không có thao tác này.");
  if (request.method !== "GET") {
    let origin: string;
    try { origin = publicWebOrigin(); } catch { origin = process.env.NODE_ENV === "production" || process.env.WEB_PUBLIC_URL ? "" : request.nextUrl.origin; }
    if (request.headers.get("origin") !== origin) return fail(403, "ORIGIN_NOT_ALLOWED", "Nguồn yêu cầu không hợp lệ.");
    if (!KEY.test(request.headers.get("Idempotency-Key") ?? "")) return fail(400, "INVALID_ARGUMENT", "Thiếu mã thao tác để thử lại an toàn.");
  }
  try {
    const session = await requireEmployeeSession(request);
    if (!session.permissions.some((item) => item === "traceability.read_internal") || !session.permissions.some((item) => item === permission)) {
      return fail(403, "PERMISSION_DENIED", "Bạn chưa có quyền thực hiện thao tác hồ sơ này.");
    }
    let base: string;
    if (usesIdentityBff()) {
      base = `${(process.env.IDENTITY_UPSTREAM_URL ?? "http://127.0.0.1:8016/api/v1").replace(/\/api\/v1\/?$/u, "").replace(/\/$/u, "")}/api/v1/admin/trace-api`;
    } else {
      if (!process.env.TRACEABILITY_ADMIN_UPSTREAM_URL) return fail(503, "UPSTREAM_UNAVAILABLE", "Chưa cấu hình dịch vụ hồ sơ nguồn gốc.");
      base = process.env.TRACEABILITY_ADMIN_UPSTREAM_URL.replace(/\/$/u, "");
    }
    const headers = credentialHeaders(new Headers({ Accept: "application/json" }), getTrustedAccessToken(request));
    const id = crypto.randomUUID(); headers.set("X-Request-Id", id);
    let body: string | undefined;
    if (request.method !== "GET") {
      try { body = await readBoundedText(request.body, 262144); JSON.parse(body); }
      catch (error) { return fail(error instanceof RangeError ? 413 : 400, "INVALID_ARGUMENT", "Nội dung hồ sơ không hợp lệ."); }
      headers.set("Content-Type", "application/json"); headers.set("Idempotency-Key", request.headers.get("Idempotency-Key")!);
    }
    if (request.nextUrl.search.length > 4096) return fail(400, "INVALID_ARGUMENT", "Điều kiện tra cứu quá dài.");
    const upstream = await fetch(`${base}/${parts.join("/")}${request.nextUrl.search}`, { method: request.method, body, headers, cache: "no-store", redirect: "manual", signal: AbortSignal.timeout(9000) });
    let payload: unknown;
    try { payload = JSON.parse(await readBoundedText(upstream.body, 1048576)); }
    catch { return fail(502, "INVALID_UPSTREAM_RESPONSE", "Phản hồi hồ sơ chưa hợp lệ; vui lòng tải lại.", id); }
    if (upstream.status < 200 || upstream.status >= 600 || upstream.status >= 300 && upstream.status < 400) return fail(502, "INVALID_UPSTREAM_RESPONSE", "Phản hồi hồ sơ chưa hợp lệ.", id);
    return NextResponse.json(payload, { status: upstream.status, headers: { "Cache-Control": "no-store", "X-Request-Id": upstream.headers.get("x-request-id") ?? id } });
  } catch (error) {
    if (error instanceof EmployeeAuthorizationError) return fail(error.status, error.code, error.message);
    return fail(503, "UPSTREAM_UNAVAILABLE", "Dịch vụ hồ sơ đang gián đoạn. Phiếu nhập kho đã ghi nhận vẫn được giữ nguyên.");
  }
}
