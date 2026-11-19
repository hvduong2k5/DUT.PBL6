import { afterEach, expect, it } from "vitest";
import { NextRequest } from "next/server";
import { hasTrustedOrigin } from "./request-origin";

const original = { ...process.env };
afterEach(() => { for (const key of ["WEB_AUTH_MODE", "WEB_PUBLIC_URL", "WEB_OIDC_ISSUER", "WEB_OIDC_CLIENT_ID", "WEB_OIDC_COOKIE_SECRET"]) { if (original[key] === undefined) delete process.env[key]; else process.env[key] = original[key]; } });

it("pins the public OIDC origin behind a proxy and ignores forwarded host claims", () => {
  Object.assign(process.env, { WEB_AUTH_MODE: "oidc", WEB_PUBLIC_URL: "https://shop.example.test", WEB_OIDC_ISSUER: "https://identity.example.test/realm", WEB_OIDC_CLIENT_ID: "web", WEB_OIDC_COOKIE_SECRET: "a-dedicated-random-secret-at-least-32-bytes" });
  expect(hasTrustedOrigin(new NextRequest("http://internal-web:3000/api/cart", { headers: { origin: "https://shop.example.test" } }))).toBe(true);
  expect(hasTrustedOrigin(new NextRequest("http://internal-web:3000/api/cart", { headers: { origin: "https://attacker.invalid", "x-forwarded-host": "attacker.invalid" } }))).toBe(false);
  expect(hasTrustedOrigin(new NextRequest("http://internal-web:3000/api/cart"))).toBe(false);
});
