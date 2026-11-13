import { NextRequest, NextResponse } from "next/server";
import { requireEmployeeSession, EmployeeAuthorizationError } from "@/features/employee/auth/server";
import { applyTrustedSessionCookie, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { hasTrustedOrigin } from "@/lib/auth/request-origin";
import { usesIdentityBff } from "@/lib/auth/oidc-config";
import { uploadPackingVideo } from "./video-upload";

const id = "[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}";
export function allowedFulfillmentRoute(path: string, method: string): boolean {
  if (method === "GET") return new RegExp(`^(?:tasks(?:/${id}(?:/history)?)?|media/${id})$`, "u").test(path);
  if (method === "POST") return new RegExp(`^(?:tasks/${id}/(?:assignments|start|unassign|seal|video)|media/${id}/complete)$`, "u").test(path);
  return method === "PUT" && new RegExp(`^tasks/${id}/(?:picks/${id}|checklist/[A-Za-z0-9_-]{1,80})$`, "u").test(path);
}
export function fulfillmentFailure(code: string, status: number) {
  return NextResponse.json({ code, message: code }, { status, headers: { "Cache-Control": "no-store" } });
}
export async function fulfillmentUpstream(request: NextRequest, path: string, method = "GET", body?: string, key?: string) {
  const bff = usesIdentityBff();
  const base = (bff ? process.env.IDENTITY_UPSTREAM_URL : process.env.NEXT_PUBLIC_API_URL)?.replace(/\/$/u, "");
  if (!base) throw new Error("Fulfillment upstream URL missing");
  const headers = new Headers({ Accept: "application/json" });
  if (bff) applyTrustedSessionCookie(headers, request);
  else { const accessToken = getTrustedAccessToken(request); if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`); }
  if (body !== undefined) headers.set("Content-Type", "application/json");
  if (key) headers.set("Idempotency-Key", key);
  const etag = request.headers.get("if-match");
  if (etag) headers.set("If-Match", etag);
  return fetch(`${base}${bff ? "/admin/packing-api/" : "/admin/fulfillment/"}${path}`, { method, headers, body, cache: "no-store", redirect: "manual", signal: AbortSignal.timeout(15000) });
}
export async function handleFulfillment(request: NextRequest, segments: string[]) {
  const path = segments.join("/");
  if (!allowedFulfillmentRoute(path, request.method)) return fulfillmentFailure("NOT_FOUND", 404);
  if (request.method !== "GET" && !hasTrustedOrigin(request)) return fulfillmentFailure("ORIGIN_NOT_ALLOWED", 403);
  try {
    await requireEmployeeSession(request);
    if (path.endsWith("/video")) return await uploadPackingVideo(request, path.replace(/\/video$/u, ""));
    const query = new URLSearchParams();
    for (const [name, value] of request.nextUrl.searchParams) {
      if (!["warehouse_id", "workflow_state", "assignee_id", "cursor", "limit"].includes(name) || query.has(name)) return fulfillmentFailure("INVALID_QUERY", 400);
      query.set(name, value);
    }
    // A bounded JSON stream prevents chunked requests from bypassing the size cap.
    let body: string | undefined;
    if (request.method !== "GET") {
      const reader = request.body?.getReader();
      const chunks: Uint8Array[] = []; let size = 0;
      if (reader) {
        for (;;) {
          const part = await reader.read(); if (part.done) break;
          size += part.value.length;
          if (size > 65536) { await reader.cancel(); return fulfillmentFailure("PAYLOAD_TOO_LARGE", 413); }
          chunks.push(part.value);
        }
      }
      body = Buffer.concat(chunks).toString("utf8");
    }
    const suffix = query.size ? `?${query}` : "";
    const response = await fulfillmentUpstream(request, path + suffix, request.method, body, request.headers.get("idempotency-key") ?? undefined);
    const headers = new Headers({ "Cache-Control": "no-store", "Content-Type": "application/json" });
    for (const name of ["etag", "retry-after", "x-request-id"]) { const value = response.headers.get(name); if (value) headers.set(name, value); }
    if (response.status >= 300 && response.status < 400) return fulfillmentFailure("UNEXPECTED_UPSTREAM_REDIRECT", 502);
    return new NextResponse(response.body, { status: response.status, headers });
  } catch (error) {
    if (error instanceof EmployeeAuthorizationError) return fulfillmentFailure(error.code, error.status);
    return fulfillmentFailure("FULFILLMENT_UPSTREAM_UNAVAILABLE", 503);
  }
}
