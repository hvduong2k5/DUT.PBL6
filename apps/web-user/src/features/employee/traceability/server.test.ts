import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { handleTraceability, tracePermission } from "./server";
import { checkedMediaURL, handleTraceMedia } from "./media-server";
import type { IdentityProfile } from "@/lib/auth/identity";

const actor = "11111111-1111-4111-8111-111111111111";
const file = "22222222-2222-4222-8222-222222222222";
const other = "33333333-3333-4333-8333-333333333333";
const origin = "http://localhost:3000";
const staff: IdentityProfile = { principalId: actor, principalType: "EMPLOYEE", provisioningStatus: "ACTIVE", email: "staff@example.test", displayName: "Staff", roles: [], permissions: ["traceability.read_internal", "traceability.create"], permissionGrants: ["traceability.read_internal", "traceability.create"].map((permissionCode) => ({ permissionCode, scopeType: "WAREHOUSE", scopeId: other })) };
function request(path: string, method = "GET", body?: string, permit?: string) {
  return new NextRequest(origin + path, { method, body, headers: { Cookie: `OMAMA_SESSION=${actor}`, Origin: origin, Authorization: "Bearer forged", "X-User-Id": other, "Idempotency-Key": "stable-operation-key", ...(permit ? { "X-Upload-Permit": permit } : {}) } });
}
function setupFetch(profile = staff, ...upstream: Response[]) {
  const fetcher = vi.fn();
  // Every operation verifies the cookie and current Identity profile again.
  for (const response of [Response.json({ data: { authenticated: true } }), Response.json({ data: profile }), ...upstream]) fetcher.mockResolvedValueOnce(response);
  vi.stubGlobal("fetch", fetcher); return fetcher;
}
beforeEach(() => {
  vi.stubEnv("WEB_AUTH_MODE", "identity-bff"); vi.stubEnv("WEB_PUBLIC_URL", origin); vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true");
  vi.stubEnv("IDENTITY_UPSTREAM_URL", "http://identity.test/api/v1");
  vi.stubEnv("TRACEABILITY_MEDIA_HTTP_UPSTREAM_URL", "http://media.test/api/v1/media");
  vi.stubEnv("TRACEABILITY_MEDIA_PUBLIC_ORIGIN", "https://files.example.test");
  vi.stubEnv("TRACEABILITY_UPLOAD_CAPABILITY_KEY", "a-server-only-capability-secret-at-least-32-bytes");
});
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

describe("Traceability BFF boundaries", () => {
  it("allowlists exact paths and methods", () => {
    expect(tracePermission("POST", ["records", actor, "revisions", "1", "reject"])).toBe("traceability.approve");
    for (const path of [["records", "..", "credentials"], ["records", actor, "revisions", "0"], ["records", actor, "unknown"]]) expect(tracePermission("GET", path)).toBeUndefined();
    expect(tracePermission("DELETE", ["records", actor])).toBeUndefined();
  });
  it("forwards the verified cookie and conflict response without browser credentials", async () => {
    const fetcher = setupFetch(staff, Response.json({ code: "CONFLICT", message: "Changed", requestId: actor }, { status: 409 }));
    const result = await handleTraceability(request("/api/admin/trace/records", "POST", "{}"), ["records"]);
    expect(result.status).toBe(409); expect(result.headers.get("cache-control")).toBe("no-store");
    const [url, init] = fetcher.mock.calls[2];
    expect(url).toBe("http://identity.test/api/v1/admin/trace-api/records");
    expect(init.headers.get("cookie")).toBe(`OMAMA_SESSION=${actor}`); expect(init.headers.get("origin")).toBe(origin);
    expect(init.headers.get("authorization")).toBeNull(); expect(init.headers.get("x-user-id")).toBeNull();
    expect(init.headers.get("idempotency-key")).toBe("stable-operation-key");
  });
  it("rejects missing Origin and oversized bodies before forwarding commands", async () => {
    const missing = request("/api/admin/trace/records", "POST", "{}"); missing.headers.delete("origin");
    expect((await handleTraceability(missing, ["records"])).status).toBe(403);
    const fetcher = setupFetch();
    expect((await handleTraceability(request("/api/admin/trace/records", "POST", "x".repeat(262145)), ["records"])).status).toBe(413);
    expect(fetcher).toHaveBeenCalledTimes(2);
  });
  it("rejects customer and unbound flat permissions", async () => {
    setupFetch({ ...staff, principalType: "CUSTOMER" });
    expect((await handleTraceability(request("/api/admin/trace/records"), ["records"])).status).toBe(403);
    const fetcher = setupFetch({ ...staff, permissionGrants: [] });
    expect((await handleTraceability(request("/api/admin/trace/records", "POST", "{}"), ["records"])).status).toBe(403);
    expect(fetcher).toHaveBeenCalledTimes(2);
  });
  it("rejects redirects, oversized and malformed upstream responses", async () => {
    for (const response of [new Response("{}", { status: 302 }), new Response("x"), new Response("x".repeat(1048577))]) {
      setupFetch(staff, response);
      expect((await handleTraceability(request("/api/admin/trace/records"), ["records"])).status).toBe(502);
    }
  });
});
describe("Certificate upload capability", () => {
  const body = JSON.stringify({ purpose: "OCOP_CERTIFICATE", fileName: "certificate.pdf", mediaType: "application/pdf", sizeBytes: 100, sha256: "a".repeat(64) });
  async function create() {
    const fetcher = setupFetch(staff, Response.json({ fileId: file, uploadSessionId: "upload-1", uploadUrl: "https://files.example.test/bucket/file?signature=x", requiredHeaders: {} }));
    const result = await handleTraceMedia(request("/api/admin/trace-media/upload-sessions", "POST", body), ["upload-sessions"]);
    expect(result.status).toBe(201); return { fetcher, payload: await result.json() };
  }
  it("derives the uploader from Identity and binds metadata/complete to actor, file and session", async () => {
    const { payload, fetcher } = await create();
    expect(fetcher.mock.calls[2][1].headers["X-User-Id"]).toBe(actor);
    expect(payload.uploadPermit).toContain(".");
    setupFetch(staff, Response.json({ id: file, purpose: "OCOP_CERTIFICATE", status: "READY", sha256Checksum: "a".repeat(64) }));
    const metadata = await handleTraceMedia(request(`/api/admin/trace-media/files/${file}`, "GET", undefined, payload.uploadPermit), ["files", file]);
    expect(metadata.status).toBe(200); expect((await metadata.json()).status).toBe("READY");
    for (const [profile, path, method] of [[{ ...staff, principalId: other }, ["files", file], "GET"], [staff, ["files", other], "GET"], [staff, ["upload-sessions", "another", "complete"], "POST"]] as const) {
      const calls = setupFetch(profile);
      expect((await handleTraceMedia(request("/api/admin/trace-media/test", method, method === "POST" ? "{}" : undefined, payload.uploadPermit), [...path])).status).toBe(403);
      expect(calls).toHaveBeenCalledTimes(2);
    }
  });
  it("rejects missing permits and invalid JSON without calling Media", async () => {
    let fetcher = setupFetch();
    expect((await handleTraceMedia(request(`/api/admin/trace-media/files/${file}`), ["files", file])).status).toBe(403);
    expect(fetcher).toHaveBeenCalledTimes(2);
    fetcher = setupFetch();
    expect((await handleTraceMedia(request("/api/admin/trace-media/upload-sessions", "POST", "null"), ["upload-sessions"])).status).toBe(400);
    expect(fetcher).toHaveBeenCalledTimes(2);
  });
  it("pins public storage URLs without credentials or alternative origins", () => {
    expect(checkedMediaURL("https://files.example.test/bucket/file?signature=x")).toContain("signature=x");
    for (const url of ["https://attacker.test/file", "https://user:secret@files.example.test/file", "http://files.example.test/file", "file:///secret"]) expect(() => checkedMediaURL(url)).toThrow();
  });
});
