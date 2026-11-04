import type { NextRequest } from "next/server";
import { oidcConfig, usesOidc, usesIdentityBff, publicWebOrigin } from "./oidc-config";

// A deployed BFF may see its internal host in nextUrl. Pin the browser origin
// explicitly for OIDC rather than trusting forwarded headers supplied by callers.
export function hasTrustedOrigin(request: NextRequest): boolean {
  const origin = request.headers.get("origin");
  if (usesIdentityBff()) return origin === publicWebOrigin();
  if (usesOidc()) return origin === oidcConfig().origin;
  if (!origin) return true;
  try { return new URL(origin).origin === request.nextUrl.origin; }
  catch { return false; }
}
