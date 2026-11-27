import { NextRequest, NextResponse } from "next/server";
import { publicWebOrigin, usesIdentityBff, usesOidc } from "@/lib/auth/oidc-config";
import { credentialHeaders, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { procurementQuery, procurementRoute } from "../routes";

function error(status: number, code: string) {
  return NextResponse.json({ code, message: "Không thể thực hiện yêu cầu Procurement.", requestId: crypto.randomUUID(), errors: [] }, { status, headers: { "Cache-Control": "no-store" } });
}

async function boundedBody(body: ReadableStream<Uint8Array> | null, limit: number): Promise<string> {
  if (!body) return "";
  const reader = body.getReader();
  const decoder = new TextDecoder("utf-8", { fatal: true });
  let result = "", bytes = 0;
  try {
    while (true) {
      const item = await reader.read();
      if (item.done) return result + decoder.decode();
      bytes += item.value.byteLength;
      if (bytes > limit) throw new RangeError("body limit");
      result += decoder.decode(item.value, { stream: true });
    }
  } finally { await reader.cancel().catch(() => undefined); reader.releaseLock(); }
}

export async function handleProcurement(request: NextRequest, parts: string[]) {
  if (process.env.ADMIN_PROCUREMENT_ENABLED !== "true") return error(404, "NOT_FOUND");
  const path = parts.join("/");
  if (parts.some((part) => !part || part.includes("/")) || !procurementRoute(request.method, path)) return error(404, "NOT_FOUND");
  const query = procurementQuery(path, request.nextUrl.searchParams);
  if (query === undefined || (request.method !== "GET" && query)) return error(400, "INVALID_QUERY");
  const bff = usesIdentityBff();
  if (!bff && !usesOidc()) return error(503, "PROCUREMENT_AUTH_NOT_CONFIGURED");
  const credential = getTrustedAccessToken(request);
  if (!credential) return error(401, "UNAUTHENTICATED");
  let origin: string;
  try { origin = publicWebOrigin(); } catch { return error(503, "PROCUREMENT_UPSTREAM_UNAVAILABLE"); }
  const mutation = request.method !== "GET";
  if (mutation && (request.headers.get("origin") !== origin || request.headers.get("sec-fetch-site") === "cross-site")) return error(403, "ORIGIN_NOT_ALLOWED");
  const key = request.headers.get("idempotency-key");
  if (mutation && (!key || !/^[!-~]{16,128}$/u.test(key))) return error(400, "IDEMPOTENCY_KEY_REQUIRED");
  let base: URL;
  try {
    const raw = bff ? process.env.IDENTITY_UPSTREAM_URL : process.env.PROCUREMENT_ADMIN_UPSTREAM_URL;
    if (!raw) throw new Error("missing upstream");
    base = new URL(raw);
    if (base.username || base.password || base.search || base.hash || base.pathname.replace(/\/$/u, "") !== "/api/v1"
      || (base.protocol !== "https:" && !(base.protocol === "http:" && process.env.PROCUREMENT_ADMIN_ALLOW_INSECURE_HTTP === "true"))) throw new Error("invalid upstream");
  } catch { return error(503, "PROCUREMENT_UPSTREAM_UNAVAILABLE"); }
  let body: string | undefined;
  if (mutation) {
    if (request.headers.get("content-type")?.split(";")[0].trim() !== "application/json") return error(415, "UNSUPPORTED_MEDIA_TYPE");
    try {
      body = await boundedBody(request.body, 65536);
      const parsed: unknown = JSON.parse(body);
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return error(400, "INVALID_ARGUMENT");
    } catch (cause) { return error(cause instanceof RangeError ? 413 : 400, "INVALID_ARGUMENT"); }
  }
  const headers = credentialHeaders(new Headers({ Accept: "application/json" }), credential);
  if (mutation) { headers.set("Content-Type", "application/json"); headers.set("Idempotency-Key", key!); }
  try {
    const response = await fetch(`${base.href.replace(/\/$/u, "")}/admin/${bff ? "procurement-api" : "procurement"}/${path}${query ? `?${query}` : ""}`, {
      method: request.method, headers, body, cache: "no-store", redirect: "manual", signal: AbortSignal.timeout(12000),
    });
    // Spring's security filters may reject before the controller and return no
    // JSON envelope. An expired/forbidden session is not an upstream 502.
    if (response.status === 401 || response.status === 403) {
      await response.body?.cancel();
      return error(response.status, response.status === 401 ? "UNAUTHENTICATED" : "FORBIDDEN");
    }
    if (response.status >= 300 && response.status < 400) { await response.body?.cancel(); return error(502, "PROCUREMENT_INVALID_RESPONSE"); }
    if (response.headers.get("content-type")?.split(";")[0].trim() !== "application/json") { await response.body?.cancel(); return error(502, "PROCUREMENT_INVALID_RESPONSE"); }
    const text = await boundedBody(response.body, 1 << 20);
    const data: unknown = JSON.parse(text);
    if (!data || typeof data !== "object" || Array.isArray(data)) return error(502, "PROCUREMENT_INVALID_RESPONSE");
    if (!response.ok && !(typeof (data as Record<string, unknown>).code === "string" && typeof (data as Record<string, unknown>).requestId === "string")) return error(502, "PROCUREMENT_INVALID_RESPONSE");
    const outgoing = new Headers({ "Cache-Control": "no-store", "Content-Type": "application/json" });
    const retry = response.headers.get("retry-after");
    if (retry && /^[0-9]{1,6}$/u.test(retry)) outgoing.set("Retry-After", retry);
    return new NextResponse(text, { status: response.status, headers: outgoing });
  } catch (cause) { return error(cause instanceof SyntaxError || cause instanceof RangeError || cause instanceof TypeError && cause.message.includes("encoded") ? 502 : 503, "PROCUREMENT_UPSTREAM_UNAVAILABLE"); }
}
