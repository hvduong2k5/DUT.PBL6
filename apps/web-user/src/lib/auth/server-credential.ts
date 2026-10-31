import type { NextRequest } from "next/server";

export const ACCESS_TOKEN_COOKIE = "oma_access_token";

export function getTrustedAccessToken(request: NextRequest): string | undefined {
  const token = request.cookies.get(ACCESS_TOKEN_COOKIE)?.value.trim();
  return token || undefined;
}

export function applyTrustedAuthorization(headers: Headers, request: NextRequest): string | undefined {
  const accessToken = getTrustedAccessToken(request);
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  return accessToken;
}
