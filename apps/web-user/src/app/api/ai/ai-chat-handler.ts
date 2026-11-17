import { randomBytes, timingSafeEqual } from "node:crypto";
import { NextRequest, NextResponse } from "next/server";
import { applyTrustedAuthorization } from "@/lib/auth/server-credential";
import { publicWebOrigin } from "@/lib/auth/oidc-config";

const CSRF = "oma_ai_csrf";
const MAX_BODY_BYTES = 8192;
const JSON_TIMEOUT_MS = 15_000;
const STREAM_TIMEOUT_MS = 130_000;

function error(status: number, code: string) {
  return NextResponse.json(
    { code, message: "Trợ lý Ô Mạ đang bận. Bạn thử lại sau ít phút nhé.", requestId: crypto.randomUUID() },
    { status, headers: { "Cache-Control": "private, no-store" } }
  );
}

function origin(request: NextRequest) {
  if (process.env.WEB_PUBLIC_URL || process.env.NODE_ENV === "production") return publicWebOrigin();
  if (!["localhost", "127.0.0.1", "[::1]"].includes(request.nextUrl.hostname)) throw new Error("Web origin required");
  return request.nextUrl.origin;
}

async function limitedBody(request: NextRequest, maxBytes: number) {
  if (request.headers.get("content-type")?.split(";")[0] !== "application/json") throw new Error("JSON required");
  const reader = request.body?.getReader();
  if (!reader) throw new Error("Body required");
  const chunks: Uint8Array[] = [];
  let length = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      length += value.byteLength;
      if (length > maxBytes) { await reader.cancel(); throw new Error("Body exceeds limit"); }
      chunks.push(value);
    }
  } finally { reader.releaseLock(); }
  return Buffer.concat(chunks);
}

type Route =
  | { kind: "session" }
  | { kind: "createConversation" }
  | { kind: "getConversation"; id: string }
  | { kind: "postMessage"; id: string }
  | { kind: "feedback" }
  | { kind: "consent" };

function resolveRoute(request: NextRequest, path: string[]): Route | null {
  const [head, id, tail] = path;
  if (path.length === 1 && head === "session" && request.method === "GET") return { kind: "session" };
  if (path.length === 1 && head === "conversations" && request.method === "POST") return { kind: "createConversation" };
  if (path.length === 2 && head === "conversations" && request.method === "GET" && isConversationId(id)) {
    return { kind: "getConversation", id };
  }
  if (path.length === 3 && head === "conversations" && tail === "messages" && request.method === "POST" && isConversationId(id)) {
    return { kind: "postMessage", id };
  }
  if (path.length === 1 && head === "feedback" && request.method === "POST") return { kind: "feedback" };
  if (path.length === 1 && head === "consent" && ["GET", "PUT"].includes(request.method)) return { kind: "consent" };
  return null;
}

function isConversationId(value: string | undefined): value is string {
  return typeof value === "string" && /^[A-Za-z0-9-]{8,64}$/u.test(value);
}

function upstreamPath(route: Exclude<Route, { kind: "session" }>): string {
  switch (route.kind) {
    case "createConversation": return "/api/v1/ai/shopping/conversations";
    case "getConversation": return `/api/v1/ai/shopping/conversations/${route.id}`;
    case "postMessage": return `/api/v1/ai/shopping/conversations/${route.id}/messages`;
    case "feedback": return "/api/v1/ai/shopping/feedback";
    case "consent": return "/api/v1/ai/consent";
  }
}

export async function aiChatProxy(request: NextRequest, path: string[]) {
  const route = resolveRoute(request, path);
  if (!route) return error(404, "NOT_FOUND");
  if (process.env.NEXT_PUBLIC_AI_CHAT_ENABLED !== "true") return error(503, "AI_CHAT_DISABLED");

  let webOrigin: string;
  try { webOrigin = origin(request); } catch { return error(503, "AI_CHAT_ORIGIN_UNAVAILABLE"); }

  if (route.kind === "session") {
    const response = NextResponse.json({ ok: true }, { headers: { "Cache-Control": "private, no-store" } });
    if (!request.cookies.has(CSRF)) {
      response.cookies.set(CSRF, randomBytes(32).toString("base64url"), {
        path: "/", sameSite: "strict", secure: webOrigin.startsWith("https:"), maxAge: 86400, httpOnly: false
      });
    }
    return response;
  }

  const base = process.env.AI_AGENT_UPSTREAM_URL;
  if (!base) return error(503, "AI_CHAT_UPSTREAM_MISSING");

  const headers = new Headers({ Accept: route.kind === "postMessage" ? "text/event-stream" : "application/json" });
  let body: Buffer | undefined;
  try {
    applyTrustedAuthorization(headers, request);
    const guestToken = request.headers.get("x-guest-conversation-token");
    if (guestToken) headers.set("x-guest-conversation-token", guestToken);
    if (request.method !== "GET") {
      const cookie = request.cookies.get(CSRF)?.value ?? "";
      const token = request.headers.get("x-csrf-token") ?? "";
      if (request.headers.get("origin") !== webOrigin || !/^[A-Za-z0-9_-]{43}$/u.test(cookie)
        || token.length !== cookie.length || !timingSafeEqual(Buffer.from(cookie), Buffer.from(token))) {
        return error(403, "INVALID_CSRF_TOKEN");
      }
      try { body = await limitedBody(request, MAX_BODY_BYTES); } catch { return error(400, "INVALID_AI_CHAT_BODY"); }
      headers.set("Content-Type", "application/json");
      headers.set("Origin", webOrigin);
      for (const name of ["idempotency-key"]) {
        const value = request.headers.get(name);
        if (value) headers.set(name, value);
      }
    }
    const url = new URL(base.replace(/\/$/u, "") + upstreamPath(route));
    const upstream = await fetch(url, {
      method: request.method, headers,
      body: body ? new Uint8Array(body) : undefined, cache: "no-store", redirect: "manual",
      signal: AbortSignal.timeout(route.kind === "postMessage" ? STREAM_TIMEOUT_MS : JSON_TIMEOUT_MS)
    });
    if (upstream.status >= 300 && upstream.status < 400) return error(502, "INVALID_AI_CHAT_REDIRECT");

    if (route.kind === "postMessage" && upstream.ok && upstream.body
      && (upstream.headers.get("content-type") ?? "").includes("text/event-stream")) {
      return new NextResponse(upstream.body, {
        status: upstream.status,
        headers: {
          "Content-Type": "text/event-stream",
          "Cache-Control": "private, no-store",
          "X-Accel-Buffering": "no"
        }
      });
    }

    const responseHeaders = new Headers({ "Cache-Control": "private, no-store" });
    for (const name of ["content-type", "retry-after", "x-request-id"]) {
      const value = upstream.headers.get(name);
      if (value) responseHeaders.set(name, value);
    }
    return new NextResponse(await upstream.arrayBuffer(), { status: upstream.status, headers: responseHeaders });
  } catch { return error(503, "AI_CHAT_TEMPORARILY_UNAVAILABLE"); }
}
