import { NextRequest, NextResponse } from "next/server";
import { publicWebOrigin } from "./oidc-config";
import { credentialHeaders } from "./server-credential";
import { getSafeReturnUrl } from "./return-url";

export function identityBase(): string {
  const base = process.env.IDENTITY_UPSTREAM_URL?.replace(/\/$/u, "");
  if (!base) throw Object.assign(new Error("IDENTITY_UNAVAILABLE"), { status: 503 });
  return base;
}

export async function ownerRequest(path: string, credential?: string, method = "GET") {
  return fetch(`${identityBase()}${path}`, { method, headers: credentialHeaders(new Headers({ Accept: "application/json" }), credential),
    cache: "no-store", redirect: "manual", signal: AbortSignal.timeout(8000) });
}

export function ownerLogin(request: NextRequest): NextResponse {
  // Let the browser receive Identity's state-binding cookie and its callback.
  const url = new URL(`${identityBase()}/public/auth/login`);
  url.searchParams.set("redirect_uri", publicWebOrigin() + getSafeReturnUrl(request.nextUrl.searchParams.get("returnUrl") ?? undefined));
  return NextResponse.redirect(url, { headers: { "Cache-Control": "no-store" } });
}

export async function ownerSession(credential: string) {
  const response = await ownerRequest("/public/auth/session", credential);
  if (!response.ok) throw Object.assign(new Error("IDENTITY_UNAVAILABLE"), { status: response.status === 401 ? 401 : 503 });
  const envelope = await response.json();
  if (envelope.success !== true || envelope.data?.authenticated !== true) throw Object.assign(new Error("AUTH_REQUIRED"), { status: 401 });
  return envelope.data as { principalId: string; expiresAt: string };
}

export async function ownerLogout(credential?: string) {
  const upstream = await ownerRequest("/public/auth/logout", credential, "POST");
  const response = NextResponse.json({ success: upstream.ok }, { status: upstream.ok ? 200 : 503, headers: { "Cache-Control": "no-store" } });
  const cookie = upstream.headers.get("set-cookie");
  if (cookie) response.headers.append("Set-Cookie", cookie);
  for (const name of ["OMAMA_SESSION", "oma_access_token", "oma_refresh_token", "oma_oidc_id_token", "oma_oidc_state"]) response.cookies.set(name, "", { httpOnly: true, secure: publicWebOrigin().startsWith("https:"), sameSite: "lax", path: "/", maxAge: 0 });
  return response;
}
