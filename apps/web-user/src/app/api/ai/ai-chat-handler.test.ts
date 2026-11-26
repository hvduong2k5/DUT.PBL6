import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { aiChatProxy } from "./ai-chat-handler";

const origin = "http://127.0.0.1:13046";
const session = "f47ac10b-58cc-4372-a567-0e02b2c3d477";
const csrf = "a".repeat(43);
const conversationId = "3f6d1f52-9d4a-4c2f-9f0e-2f4a6c8d1e55";

beforeEach(() => {
  vi.stubEnv("NEXT_PUBLIC_AI_CHAT_ENABLED", "true");
  vi.stubEnv("AI_AGENT_UPSTREAM_URL", "http://ai.test:8019");
  vi.stubEnv("WEB_AUTH_MODE", "identity-bff");
  vi.stubEnv("WEB_PUBLIC_URL", origin);
  vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true");
});
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

function request(path = "", init?: ConstructorParameters<typeof NextRequest>[1]) {
  return new NextRequest(origin + "/api/ai" + path, init);
}

function mutationHeaders(extra: Record<string, string> = {}) {
  return {
    Origin: origin,
    Cookie: `OMAMA_SESSION=${session}; oma_ai_csrf=${csrf}`,
    "X-CSRF-Token": csrf,
    "Content-Type": "application/json",
    ...extra
  };
}

describe("AI chat BFF", () => {
  it("stays closed while the feature flag is off", async () => {
    vi.stubEnv("NEXT_PUBLIC_AI_CHAT_ENABLED", "false");
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    const response = await aiChatProxy(request("/session"), ["session"]);
    expect(response.status).toBe(503);
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("rejects unknown routes", async () => {
    const response = await aiChatProxy(request("/knowledge"), ["knowledge"]);
    expect(response.status).toBe(404);
  });

  it("issues the CSRF cookie from the session route without calling upstream", async () => {
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    const response = await aiChatProxy(request("/session"), ["session"]);
    expect(response.status).toBe(200);
    expect(response.cookies.get("oma_ai_csrf")?.value).toMatch(/^[A-Za-z0-9_-]{43}$/u);
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("rejects a mutation without a matching CSRF token", async () => {
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    const response = await aiChatProxy(request("/conversations", {
      method: "POST",
      headers: { Origin: origin, "Content-Type": "application/json" },
      body: "{}"
    }), ["conversations"]);
    expect(response.status).toBe(403);
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("creates a conversation and forwards the idempotency key", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json(
      { conversation_id: conversationId, owner_type: "customer", expires_at: "2026-12-17T00:00:00Z" }, { status: 201 }));
    vi.stubGlobal("fetch", fetcher);
    const response = await aiChatProxy(request("/conversations", {
      method: "POST",
      headers: mutationHeaders({ "Idempotency-Key": "7c9e6679-7425-40de-944b-e07fc1f90ae7" }),
      body: "{}"
    }), ["conversations"]);
    expect(response.status).toBe(201);
    const [url, options] = fetcher.mock.calls[0];
    expect(url.href).toBe("http://ai.test:8019/api/v1/ai/shopping/conversations");
    expect(options.headers.get("Idempotency-Key")).toBe("7c9e6679-7425-40de-944b-e07fc1f90ae7");
    expect(options.redirect).toBe("manual");
  });

  it("forwards the guest token when reading a guest conversation", async () => {
    const fetcher = vi.fn().mockResolvedValue(Response.json({ conversation_id: conversationId, messages: [] }));
    vi.stubGlobal("fetch", fetcher);
    const response = await aiChatProxy(request(`/conversations/${conversationId}`, {
      headers: { "x-guest-conversation-token": "guest-secret" }
    }), ["conversations", conversationId]);
    expect(response.status).toBe(200);
    expect(fetcher.mock.calls[0][1].headers.get("x-guest-conversation-token")).toBe("guest-secret");
  });

  it("streams the assistant reply through as server-sent events", async () => {
    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('data: {"type":"token","text":"Dạ"}\n\n'));
        controller.enqueue(new TextEncoder().encode('data: {"type":"done","intent":"shopping"}\n\n'));
        controller.close();
      }
    });
    const fetcher = vi.fn().mockResolvedValue(new Response(stream, {
      status: 200, headers: { "Content-Type": "text/event-stream" }
    }));
    vi.stubGlobal("fetch", fetcher);
    const response = await aiChatProxy(request(`/conversations/${conversationId}/messages`, {
      method: "POST",
      headers: mutationHeaders(),
      body: JSON.stringify({ content: "Mè xửng nào ít ngọt?" })
    }), ["conversations", conversationId, "messages"]);
    expect(response.status).toBe(200);
    expect(response.headers.get("Content-Type")).toContain("text/event-stream");
    const text = await response.text();
    expect(text).toContain('"type":"token"');
    expect(text).toContain('"type":"done"');
    const [url] = fetcher.mock.calls[0];
    expect(url.href).toBe(`http://ai.test:8019/api/v1/ai/shopping/conversations/${conversationId}/messages`);
  });

  it("passes an upstream rate limit through with its status", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({ code: "rate_limited" }, { status: 429 })));
    const response = await aiChatProxy(request("/consent"), ["consent"]);
    expect(response.status).toBe(429);
  });
});
