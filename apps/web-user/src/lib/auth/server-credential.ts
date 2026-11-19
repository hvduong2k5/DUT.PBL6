import type { NextRequest } from "next/server";
import { publicWebOrigin, usesIdentityBff } from "./oidc-config";

export const ACCESS_TOKEN_COOKIE = "oma_access_token";
export const SESSION_COOKIE = "OMAMA_SESSION";

export function getTrustedSessionCookie(request: NextRequest): string | undefined {
  return request.cookies.get(SESSION_COOKIE)?.value.trim() || undefined;
}

export function applyTrustedSessionCookie(headers: Headers, request: NextRequest): string | undefined {
  const session = getTrustedSessionCookie(request);
  if (session) headers.set("Cookie", `${SESSION_COOKIE}=${session}`);
  if (usesIdentityBff()) headers.set("Origin", publicWebOrigin());
  return session;
}

export function getTrustedAccessToken(request: NextRequest): string | undefined {
  // In owner BFF mode this value is an opaque session ID, never an access token.
  const token = request.cookies.get(usesIdentityBff() ? "OMAMA_SESSION" : ACCESS_TOKEN_COOKIE)?.value.trim();
  if (usesIdentityBff() && (!token || !/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/iu.test(token))) return undefined;
  return token || undefined;
}

export function applyTrustedAuthorization(headers: Headers, request: NextRequest): string | undefined {
  const accessToken = getTrustedAccessToken(request);
  credentialHeaders(headers, accessToken);
  return accessToken;
}

export function credentialHeaders(headers: Headers, credential?: string): Headers {
  if (usesIdentityBff()) {
    headers.delete("Authorization");
    if (credential && /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/iu.test(credential)) headers.set("Cookie", `OMAMA_SESSION=${credential}`);
    headers.set("Origin", publicWebOrigin());
  } else if (credential) headers.set("Authorization", `Bearer ${credential}`);
  return headers;
}
