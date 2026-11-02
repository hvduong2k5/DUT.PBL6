import type { NextRequest } from "next/server";
import { describe, expect, it } from "vitest";
import { applyTrustedSessionCookie, getTrustedSessionCookie } from "./server-credential";

function request(cookieToken?: string, browserAuthorization?: string): NextRequest {
  return {
    cookies: { get: (name: string) => name === "OMAMA_SESSION" && cookieToken ? { name, value: cookieToken } : undefined },
    headers: new Headers(browserAuthorization ? { Authorization: browserAuthorization } : {})
  } as unknown as NextRequest;
}

describe("server credential boundary", () => {
  it("uses only the HttpOnly BFF session cookie", () => {
    const incoming = request("cookie-token", "Bearer browser-token");
    const outgoing = new Headers();

    expect(getTrustedSessionCookie(incoming)).toBe("cookie-token");
    expect(applyTrustedSessionCookie(outgoing, incoming)).toBe("cookie-token");
    expect(outgoing.get("Cookie")).toBe("OMAMA_SESSION=cookie-token");
    expect(outgoing.has("Authorization")).toBe(false);
  });

  it("does not promote a browser Authorization header when the cookie is absent", () => {
    const incoming = request(undefined, "Bearer forged-browser-token");
    const outgoing = new Headers();

    expect(getTrustedSessionCookie(incoming)).toBeUndefined();
    expect(applyTrustedSessionCookie(outgoing, incoming)).toBeUndefined();
    expect(outgoing.has("Authorization")).toBe(false);
  });
});
