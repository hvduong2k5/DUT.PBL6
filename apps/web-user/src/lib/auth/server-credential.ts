import type { NextRequest } from "next/server";

export const SESSION_COOKIE = "OMAMA_SESSION";

export function getTrustedSessionCookie(request: NextRequest): string | undefined {
  const session = request.cookies.get(SESSION_COOKIE)?.value.trim();
  return session || undefined;
}

export function applyTrustedSessionCookie(headers: Headers, request: NextRequest): string | undefined {
  const session = getTrustedSessionCookie(request);
  if (session) headers.set("Cookie", `${SESSION_COOKIE}=${session}`);
  return session;
}
