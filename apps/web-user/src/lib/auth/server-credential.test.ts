import type { NextRequest } from "next/server";
import { describe, expect, it } from "vitest";
import { applyTrustedAuthorization, getTrustedAccessToken } from "./server-credential";

function request(cookieToken?: string, browserAuthorization?: string): NextRequest {
  return {
    cookies: { get: (name: string) => name === "oma_access_token" && cookieToken ? { name, value: cookieToken } : undefined },
    headers: new Headers(browserAuthorization ? { Authorization: browserAuthorization } : {})
  } as unknown as NextRequest;
}

describe("server credential boundary", () => {
  it("uses only the HttpOnly access-token cookie", () => {
    const incoming = request("cookie-token", "Bearer browser-token");
    const outgoing = new Headers();

    expect(getTrustedAccessToken(incoming)).toBe("cookie-token");
    expect(applyTrustedAuthorization(outgoing, incoming)).toBe("cookie-token");
    expect(outgoing.get("Authorization")).toBe("Bearer cookie-token");
  });

  it("does not promote a browser Authorization header when the cookie is absent", () => {
    const incoming = request(undefined, "Bearer forged-browser-token");
    const outgoing = new Headers();

    expect(getTrustedAccessToken(incoming)).toBeUndefined();
    expect(applyTrustedAuthorization(outgoing, incoming)).toBeUndefined();
    expect(outgoing.has("Authorization")).toBe(false);
  });
});
