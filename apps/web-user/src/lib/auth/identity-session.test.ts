import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createServer, type Server } from "node:http";
import { generateKeyPair, exportJWK, SignJWT } from "jose";
import { identitySession } from "./identity-session";

// Real-module harness in the style of oidc.test.ts: one local server plays the
// OIDC provider (discovery + JWKS) and both upstreams. Paths are prefixed so
// the test can prove WHICH base /users/me was read from:
//   /id/... = identity upstream (IDENTITY_UPSTREAM_URL)
//   /cc/... = customer core upstream (CUSTOMER_CORE_UPSTREAM_URL, the
//             customer-only proxy path that rejects employees with 403)

const principalId = "331a7515-739c-4209-a24b-7021d7840878";
const subject = "8617a70b-5a1c-4fdc-9962-d8cfe8240e38";
const customerId = "41e91164-32ba-43de-ab5f-e86ba49525ca";

let server: Server;
let issuer = "";
let keyPair: Awaited<ReturnType<typeof generateKeyPair>>;
let hits: string[] = [];

async function signAccess(principalType: "CUSTOMER" | "EMPLOYEE") {
  return new SignJWT({ user_id: principalId, typ: "Bearer", principal_type: principalType })
    .setProtectedHeader({ alg: "RS256", kid: "test-key" })
    .setIssuer(issuer)
    .setAudience("omama-api")
    .setSubject(subject)
    .setIssuedAt()
    .setExpirationTime("5m")
    .sign(keyPair.privateKey);
}

describe("identitySession /users/me routing", () => {
  beforeEach(async () => {
    hits = [];
    keyPair = await generateKeyPair("RS256");
    const jwk = { ...await exportJWK(keyPair.publicKey), kid: "test-key", use: "sig", alg: "RS256" };
    server = createServer((request, response) => {
      response.setHeader("Content-Type", "application/json");
      const send = (body: unknown, status = 200) => { response.statusCode = status; response.end(JSON.stringify(body)); };
      const url = request.url ?? "";
      if (url === "/.well-known/openid-configuration") return send({ issuer, authorization_endpoint: `${issuer}/auth`, token_endpoint: `${issuer}/token`, jwks_uri: `${issuer}/certs`, end_session_endpoint: `${issuer}/logout` });
      if (url === "/certs") return send({ keys: [jwk] });
      hits.push(url);
      if (url === "/id/users/me") return send({ success: true, data: { principalId, idpSubject: subject, principalType: "EMPLOYEE", provisioningStatus: "ACTIVE", email: "staff@test.local", roles: [{ roleCode: "ADMIN", scopeType: "GLOBAL", scopeId: null }], permissions: ["ORDER_VIEW", "FULFILLMENT_TASK_READ"] } });
      if (url === "/cc/users/me") return send({ success: true, data: { principalId, idpSubject: subject, principalType: "CUSTOMER", provisioningStatus: "ACTIVE", email: "customer@test.local", roles: ["CUSTOMER"], permissions: [] } });
      if (url === "/cc/profile") return send({ id: customerId, user_id: principalId, full_name: "Cust Omer", status: "ACTIVE" });
      return send({ code: "UNSUPPORTED" }, 501);
    });
    await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
    const address = server.address();
    if (!address || typeof address === "string") throw new Error("No provider address");
    issuer = `http://127.0.0.1:${address.port}`;
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    vi.stubEnv("WEB_PUBLIC_URL", "http://localhost:13000");
    vi.stubEnv("WEB_OIDC_ISSUER", issuer);
    vi.stubEnv("WEB_OIDC_CLIENT_ID", "frontend-web");
    vi.stubEnv("WEB_OIDC_COOKIE_SECRET", "test-cookie-secret-more-than-thirty-two-bytes");
    vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP", "true");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", `${issuer}/cc`);
    vi.stubEnv("IDENTITY_UPSTREAM_URL", `${issuer}/id`);
  });

  afterEach(async () => {
    vi.unstubAllEnvs();
    server.closeAllConnections();
    await new Promise<void>(resolve => server.close(() => resolve()));
  });

  it("reads an EMPLOYEE profile from the identity base, not the customer-only proxy", async () => {
    const session = await identitySession(await signAccess("EMPLOYEE"));
    expect(hits).toEqual(["/id/users/me"]);
    expect(session.principal.principalType).toBe("EMPLOYEE");
    expect(session.principal.permissions).toContain("ORDER_VIEW");
  });

  it("keeps a CUSTOMER profile on the customer core for /users/me and /profile", async () => {
    const session = await identitySession(await signAccess("CUSTOMER"));
    expect(hits).toEqual(["/cc/users/me", "/cc/profile"]);
    expect(session.principal.principalType).toBe("CUSTOMER");
    expect(session.customerAccess).toBe(true);
  });
});
