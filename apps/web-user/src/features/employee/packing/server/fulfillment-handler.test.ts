import { afterEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { allowedFulfillmentRoute, handleFulfillment } from "./fulfillment-handler";
import { storageUploadTarget } from "./video-upload";
import { canUseFulfillment } from "../fulfillment-types";
import type { IdentityProfile } from "@/lib/auth/identity";

const task = "00000000-0000-4000-8000-000000000001";
const base = "http://127.0.0.1:3000";
function setup() {
  vi.stubEnv("WEB_AUTH_MODE", "identity-bff"); vi.stubEnv("WEB_PUBLIC_URL", base);
  vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true"); vi.stubEnv("IDENTITY_UPSTREAM_URL", "http://identity.test/api/v1");
}
const staff: IdentityProfile = { principalId: task, principalType: "EMPLOYEE", provisioningStatus: "ACTIVE", email: "staff@example.test", displayName: "Staff", roles: [{ roleCode: "ADMIN", scopeType: "WAREHOUSE", scopeId: "WH-1" }], permissions: [] };
function identityResponses() { return [Response.json({ data: { authenticated: true } }), Response.json({ data: staff })]; }
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

describe("Fulfillment cookie BFF", () => {
  it("allowlists exact routes and methods, excludes arbitrary/internal and proof paths", () => {
    expect(allowedFulfillmentRoute(`tasks/${task}/start`, "POST")).toBe(true);
    for (const path of ["../internal/permit", "tasks/evil", `tasks/${task}/reopen`, `tasks/${task}/media-sessions`, `tasks/${task}/seal/extra`]) expect(allowedFulfillmentRoute(path, "POST")).toBe(false);
    expect(allowedFulfillmentRoute(`tasks/${task}/start`, "GET")).toBe(false);
  });
  it("ignores browser bearer headers without a trusted cookie", async () => {
    setup(); const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    const response = await handleFulfillment(new NextRequest(base + "/api/admin/fulfillment/tasks", { headers: { Authorization: "Bearer forged" } }), ["tasks"]);
    expect(response.status).toBe(401); expect(fetcher).not.toHaveBeenCalled();
  });
  it("denies cross-origin mutations before reaching any dependency", async () => {
    setup(); const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    const response = await handleFulfillment(new NextRequest(base + "/api/admin/fulfillment/tasks", { method: "POST", headers: { Origin: "https://attacker.test", Cookie: "OMAMA_SESSION=trusted" }, body: "{}" }), ["tasks", task, "start"]);
    expect(response.status).toBe(403); expect(fetcher).not.toHaveBeenCalled();
  });
  it("preserves command identity and forwards only the verified session through Identity", async () => {
    setup(); const responses = identityResponses();
    const fetcher = vi.fn().mockResolvedValueOnce(responses[0]).mockResolvedValueOnce(responses[1]).mockResolvedValueOnce(Response.json({ code: "VERSION_MISMATCH" }, { status: 412, headers: { ETag: '"7"', "Set-Cookie": "forbidden=secret" } }));
    vi.stubGlobal("fetch", fetcher);
    const body = JSON.stringify({ counted_quantity: 2, allocation_version: 3 });
    const response = await handleFulfillment(new NextRequest(base + "/api/admin/fulfillment/tasks", { method: "PUT", headers: { Origin: base, Cookie: "OMAMA_SESSION=trusted; other=secret", Authorization: "Bearer forged", "If-Match": '"4"', "Idempotency-Key": "stable-command-123", "X-Internal-Token": "forged" }, body }), ["tasks", task, "picks", task]);
    expect(response.status).toBe(412); expect(response.headers.get("etag")).toBe('"7"'); expect(response.headers.get("set-cookie")).toBeNull();
    const [url, init] = fetcher.mock.calls[2];
    expect(url).toBe(`http://identity.test/api/v1/admin/packing-api/tasks/${task}/picks/${task}`);
    expect(init.headers.get("cookie")).toBe("OMAMA_SESSION=trusted");
    expect(init.headers.get("authorization")).toBeNull(); expect(init.headers.get("x-internal-token")).toBeNull();
    expect(init.headers.get("if-match")).toBe('"4"'); expect(init.headers.get("idempotency-key")).toBe("stable-command-123"); expect(init.body).toBe(body);
  });
  it("rejects oversized chunked JSON instead of buffering without limit", async () => {
    setup(); const responses = identityResponses(); vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(responses[0]).mockResolvedValueOnce(responses[1]));
    const response = await handleFulfillment(new NextRequest(base + "/api/admin/fulfillment/tasks", { method: "POST", headers: { Origin: base, Cookie: "OMAMA_SESSION=trusted" }, body: "x".repeat(65537) }), ["tasks", task, "start"]);
    expect(response.status).toBe(413);
  });
  it("rejects duplicate filters instead of changing query semantics", async () => {
    setup(); const responses = identityResponses(); vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(responses[0]).mockResolvedValueOnce(responses[1]));
    expect((await handleFulfillment(new NextRequest(base + "/api/admin/fulfillment/tasks?limit=1&limit=100", { headers: { Cookie: "OMAMA_SESSION=trusted" } }), ["tasks"])).status).toBe(400);
  });
});
describe("Packing upload and access boundaries", () => {
  it("matches the service ADMIN mapping without expanding Customer/STORE access", () => {
    const profile = staff as IdentityProfile;
    expect(canUseFulfillment(profile)).toBe(true);
    expect(canUseFulfillment({ ...profile, principalType: "CUSTOMER" })).toBe(false);
    expect(canUseFulfillment({ ...profile, provisioningStatus: "SUSPENDED" })).toBe(false);
    expect(canUseFulfillment({ ...profile, roles: [{ roleCode: "ADMIN", scopeType: "STORE", scopeId: "WH-1" }] })).toBe(false);
  });
  it("pins upload origins, preserves the signed host and rejects URL credentials", () => {
    vi.stubEnv("FULFILLMENT_STORAGE_UPLOAD_ORIGIN", "http://storage:9000");
    vi.stubEnv("FULFILLMENT_STORAGE_UPLOAD_CONNECT_ORIGIN", "http://127.0.0.1:19000");
    const signed = storageUploadTarget("http://storage:9000/bucket/video?X-Amz-Signature=example");
    expect(signed.url.host).toBe("storage:9000"); expect(signed.target.host).toBe("127.0.0.1:19000");
    for (const url of ["http://attacker.test/video", "http://storage:9000@attacker.test/video", "http://user:secret@storage:9000/video", "file:///etc/passwd"]) expect(() => storageUploadTarget(url)).toThrow();
  });
});

describe("Fulfillment OIDC BFF", () => {
  function setupOidc() {
    vi.stubEnv("WEB_AUTH_MODE", "oidc"); vi.stubEnv("WEB_PUBLIC_URL", base);
    vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "http://identity.test/api/v1/customer-api");
    vi.stubEnv("NEXT_PUBLIC_API_URL", "http://gateway.test/api/v1");
  }
  it("ignores a browser bearer <redacted> without the trusted token cookie", async () => {
    setupOidc(); const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    const response = await handleFulfillment(new NextRequest(base + "/api/admin/fulfillment/tasks", { headers: { Authorization: "Bearer forged" } }), ["tasks"]);
    expect(response.status).toBe(401); expect(fetcher).not.toHaveBeenCalled();
  });
  it("forwards only the trusted access token to the gateway fulfillment API", async () => {
    setupOidc();
    const fetcher = vi.fn()
      .mockResolvedValueOnce(Response.json({ data: staff }))
      .mockResolvedValueOnce(Response.json({ items: [], next_cursor: null }));
    vi.stubGlobal("fetch", fetcher);
    const response = await handleFulfillment(new NextRequest(base + "/api/admin/fulfillment/tasks", { headers: { Cookie: "oma_access_token=staff-token; other=secret", Authorization: "Bearer forged" } }), ["tasks"]);
    expect(response.status).toBe(200);
    expect(fetcher).toHaveBeenCalledTimes(2);
    const [sessionUrl] = fetcher.mock.calls[0];
    expect(sessionUrl).toBe("http://identity.test/api/v1/customer-api/users/me");
    const [url, init] = fetcher.mock.calls[1];
    expect(url).toBe("http://gateway.test/api/v1/admin/fulfillment/tasks");
    expect(init.headers.get("authorization")).toBe("Bearer staff-token");
    expect(init.headers.get("cookie")).toBeNull();
  });
});
