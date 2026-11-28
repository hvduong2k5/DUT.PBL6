import { NextRequest, NextResponse } from "next/server";
import { EmployeeAuthorizationError, requireEmployeeSession } from "@/features/employee/auth/server";
import { publicWebOrigin } from "@/lib/auth/oidc-config";
import { createHmac, timingSafeEqual } from "node:crypto";
import { handleTraceability, readBoundedText } from "./server";
import type { TraceProfile } from "./types";

const UUID = /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/u;
const MIME = new Set(["application/pdf", "image/jpeg", "image/png", "image/webp"]);
function fail(status: number, message: string) { return NextResponse.json({ code: "TRACEABILITY_MEDIA_ERROR", message, requestId: crypto.randomUUID() }, { status, headers: { "Cache-Control": "no-store" } }); }
export function checkedMediaURL(raw: unknown): string {
  if (typeof raw !== "string" || !process.env.TRACEABILITY_MEDIA_PUBLIC_ORIGIN) throw new Error("Missing Media public origin");
  const url = new URL(raw); const allowed = new URL(process.env.TRACEABILITY_MEDIA_PUBLIC_ORIGIN);
  if (url.origin !== allowed.origin || url.username || url.password || url.hash || !(url.protocol === "https:" || process.env.NODE_ENV !== "production" && url.protocol === "http:" && ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname))) throw new Error("Untrusted Media URL");
  return url.href;
}
class UploadPermitError extends Error {}
interface UploadPermit { actor: string; file: string; session: string; expires: number }
function fileMetadata(file: Record<string, unknown>, expected: string) {
  const id = file.fileId ?? file.id;
  if (id !== expected || !["AWAITING_UPLOAD", "PROCESSING", "READY", "PROCESSING_FAILED", "ARCHIVED"].includes(String(file.status))) throw new Error("Invalid Media metadata");
  return { fileId: id, status: file.status, sha256Checksum: file.sha256Checksum ?? null };
}
function permitKey() { const secret = process.env.TRACEABILITY_UPLOAD_CAPABILITY_KEY; if (!secret || secret.length < 32) throw new Error("Missing upload capability key"); return secret; }
function issuePermit(value: UploadPermit): string {
  const payload = Buffer.from(JSON.stringify(value)).toString("base64url");
  return `${payload}.${createHmac("sha256", permitKey()).update(`traceability-upload:${payload}`).digest("base64url")}`;
}
function readPermit(raw: string | null, actor: string): UploadPermit {
  if (!raw || raw.length > 2048) throw new UploadPermitError("Missing upload permit");
  const [payload, mac, extra] = raw.split(".");
  if (!payload || !mac || extra) throw new UploadPermitError("Invalid upload permit");
  const expected = createHmac("sha256", permitKey()).update(`traceability-upload:${payload}`).digest();
  const actual = Buffer.from(mac, "base64url");
  if (actual.length !== expected.length || !timingSafeEqual(actual, expected)) throw new UploadPermitError("Invalid upload permit");
  const value = JSON.parse(Buffer.from(payload, "base64url").toString("utf8")) as UploadPermit;
  if (value.actor !== actor || !UUID.test(value.file) || typeof value.session !== "string" || !/^[A-Za-z0-9_-]{1,128}$/u.test(value.session) || !Number.isInteger(value.expires) || value.expires < Date.now()) throw new UploadPermitError("Expired upload permit");
  return value;
}
async function media(path: string, actor: string, method = "GET", body?: unknown, key?: string) {
  const base = process.env.TRACEABILITY_MEDIA_HTTP_UPSTREAM_URL?.replace(/\/$/u, "");
  if (!base) throw new Error("Missing Media provider");
  const response = await fetch(`${base}/${path}`, { method, headers: { Accept: "application/json", "X-User-Id": actor, ...(body ? { "Content-Type": "application/json", "Idempotency-Key": key! } : {}) }, body: body ? JSON.stringify(body) : undefined, cache: "no-store", redirect: "manual", signal: AbortSignal.timeout(10000) });
  const reader = response.body?.getReader(); let bytes = 0; let text = ""; const decoder = new TextDecoder();
  if (!reader) throw new Error("Invalid Media response");
  try { for (;;) { const chunk = await reader.read(); if (chunk.done) break; bytes += chunk.value.byteLength; if (bytes > 65536) { await reader.cancel(); throw new Error("Media response too large"); } text += decoder.decode(chunk.value, { stream: true }); } } finally { reader.releaseLock(); }
  text += decoder.decode();
  const payload = JSON.parse(text) as Record<string, unknown>;
  if (!response.ok) throw new Error("Media provider unavailable");
  return payload;
}

export async function handleTraceMedia(request: NextRequest, parts: string[]) {
  const create = request.method === "POST" && parts.join("/") === "upload-sessions";
  const complete = request.method === "POST" && parts.length === 3 && parts[0] === "upload-sessions" && /^[A-Za-z0-9_-]{1,128}$/u.test(parts[1]) && parts[2] === "complete";
  const metadata = request.method === "GET" && parts.length === 2 && parts[0] === "files" && UUID.test(parts[1]);
  if (!create && !complete && !metadata) return fail(404, "Không có thao tác tập tin này.");
  try {
    const session = await requireEmployeeSession(request);
    if (!session.permissions.includes("traceability.read_internal") || !session.permissions.some((permission) => permission === "traceability.create" || permission === "traceability.edit")) return fail(403, "Bạn chưa có quyền thêm chứng nhận.");
    if (request.method === "POST") {
      let origin: string; try { origin = publicWebOrigin(); } catch { origin = process.env.NODE_ENV === "production" || process.env.WEB_PUBLIC_URL ? "" : request.nextUrl.origin; }
      if (!origin || request.headers.get("origin") !== origin) return fail(403, "Nguồn yêu cầu không hợp lệ.");
      if (!/^[!-~]{16,128}$/u.test(request.headers.get("Idempotency-Key") ?? "")) return fail(400, "Thiếu mã thao tác.");
    }
    const actor = session.employee.employeeId;
    if (metadata) {
      const permit = readPermit(request.headers.get("X-Upload-Permit"), actor);
      if (permit.file !== parts[1]) return fail(403, "Mã tập tin không đúng phiên tải lên.");
      const file = await media(parts[1], actor);
      if (file.purpose !== "OCOP_CERTIFICATE") return fail(403, "Bạn không có quyền xem trạng thái tập tin này.");
      return NextResponse.json(fileMetadata(file, permit.file), { headers: { "Cache-Control": "no-store" } });
    }
    let raw: string; try { raw = await readBoundedText(request.body, 16384); } catch { return fail(413, "Thông tin tập tin quá lớn."); }
    let input: Record<string, unknown>;
    try { const value: unknown = JSON.parse(raw); if (!value || typeof value !== "object" || Array.isArray(value)) return fail(400, "Thông tin tập tin không hợp lệ."); input = value as Record<string, unknown>; } catch { return fail(400, "Thông tin tập tin không hợp lệ."); }
    if (create && (input.purpose !== "OCOP_CERTIFICATE" || typeof input.fileName !== "string" || !input.fileName || input.fileName.length > 250 || typeof input.mediaType !== "string" || !MIME.has(input.mediaType) || !Number.isInteger(input.sizeBytes) || Number(input.sizeBytes) < 1 || Number(input.sizeBytes) > 20971520 || !/^[0-9a-f]{64}$/u.test(String(input.sha256 ?? "")))) return fail(400, "Chọn bản scan PDF, JPEG, PNG hoặc WebP không quá 20 MB.");
    if (complete) { const permit = readPermit(request.headers.get("X-Upload-Permit"), actor); if (permit.session !== parts[1]) return fail(403, "Phiên tải lên không đúng tập tin."); }
    if (create) permitKey();
    const result = await media(parts.join("/"), actor, "POST", create ? input : {}, request.headers.get("Idempotency-Key")!);
    if (create) {
      result.uploadUrl = checkedMediaURL(result.uploadUrl);
      if (typeof result.fileId !== "string" || !UUID.test(result.fileId) || typeof result.uploadSessionId !== "string" || !/^[A-Za-z0-9_-]{1,128}$/u.test(result.uploadSessionId)) throw new Error("Invalid upload session");
      const headers = result.requiredHeaders;
      if (!headers || typeof headers !== "object" || Array.isArray(headers) || Object.entries(headers).some(([name, value]) => name.toLowerCase() !== "content-type" || value !== input.mediaType)) throw new Error("Invalid upload headers");
      result.uploadPermit = issuePermit({ actor, file: result.fileId, session: result.uploadSessionId, expires: Date.now() + 1200000 });
      return NextResponse.json({ fileId: result.fileId, uploadSessionId: result.uploadSessionId, uploadUrl: result.uploadUrl, requiredHeaders: headers, uploadPermit: result.uploadPermit, expiresAt: result.expiresAt }, { status: 201, headers: { "Cache-Control": "no-store" } });
    }
    const permit = readPermit(request.headers.get("X-Upload-Permit"), actor);
    if (result.purpose !== "OCOP_CERTIFICATE") return fail(403, "Tập tin không thuộc chứng nhận OCOP.");
    return NextResponse.json(fileMetadata(result, permit.file), { headers: { "Cache-Control": "no-store" } });
  } catch (error) {
    if (error instanceof EmployeeAuthorizationError) return fail(error.status, error.message);
    if (error instanceof UploadPermitError) return fail(403, "Phiên tải lên không hợp lệ hoặc đã hết hạn.");
    return fail(503, "Chưa thể xử lý bản scan. Hồ sơ và phiếu nhập đã lưu vẫn giữ nguyên.");
  }
}

export async function traceEvidenceURL(request: NextRequest, profileId: string, evidenceId: string) {
  if (!UUID.test(profileId) || !UUID.test(evidenceId)) return fail(404, "Không tìm thấy chứng nhận.");
  try {
    const readRequest = new NextRequest(new URL(request.nextUrl.pathname, request.nextUrl.origin), { headers: request.headers });
    const response = await handleTraceability(readRequest, ["records", profileId]);
    if (!response.ok) return response;
    const profile = await response.json() as TraceProfile;
    const revision = Number(request.nextUrl.searchParams.get("revision"));
    const evidence = profile.revisions.find((item) => item.number === revision)?.content.evidence.find((item) => item.id === evidenceId);
    // Reference is returned by Traceability only after current permission-and-warehouse checks.
    const fileId = evidence?.reference?.startsWith("document:") ? evidence.reference.slice(9) : "";
    if (!UUID.test(fileId)) return fail(403, "Bạn chưa có quyền mở bản scan hoặc tài liệu cũ chưa được chuyển sang Media.");
    const session = await requireEmployeeSession(request);
    const file = await media(fileId, session.employee.employeeId);
    if (file.purpose !== "OCOP_CERTIFICATE") return fail(403, "Tập tin không thuộc chứng nhận OCOP.");
    return NextResponse.redirect(checkedMediaURL(file.viewUrl), { status: 307, headers: { "Cache-Control": "no-store" } });
  } catch { return fail(503, "Chưa thể mở bản scan; vui lòng thử lại."); }
}
