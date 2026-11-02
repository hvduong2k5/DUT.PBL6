import { parseIdentityProfile, parseIdentitySession, type IdentityProfile } from "@/lib/auth/identity";
import { AuthApiError, type ApiErrorBody } from "@/lib/auth/types";

export const IDENTITY_API_BASE_URL = (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8016/api/v1").replace(/\/$/u, "");

async function payload(response: Response): Promise<unknown> {
  return (response.headers.get("content-type") ?? "").includes("application/json") ? response.json() : undefined;
}

async function request(path: string, init: RequestInit = {}): Promise<unknown> {
  const response = await fetch(`${IDENTITY_API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
    cache: "no-store",
    headers: { Accept: "application/json", ...init.headers }
  });
  const body = await payload(response);
  if (!response.ok) throw new AuthApiError(response.status, (body ?? {}) as ApiErrorBody);
  return body;
}

export const identityService = {
  async currentProfile(): Promise<IdentityProfile | null> {
    const session = parseIdentitySession(await request("/public/auth/session"));
    if (!session) throw new AuthApiError(502, { code: "IDENTITY_SESSION_INVALID", message: "Dữ liệu phiên đăng nhập không đúng contract." });
    if (!session.authenticated) return null;
    const profile = parseIdentityProfile(await request("/users/me"));
    if (!profile) throw new AuthApiError(502, { code: "IDENTITY_PROFILE_INVALID", message: "Hồ sơ định danh không đúng contract." });
    return profile;
  },
  loginUrl(returnUrl: string) {
    const url = new URL(`${IDENTITY_API_BASE_URL}/public/auth/login`);
    url.searchParams.set("redirect_uri", returnUrl);
    return url.toString();
  },
  async logout() {
    await request("/public/auth/logout", { method: "POST" });
  },
  async changePassword(currentPassword: string, newPassword: string) {
    return request("/users/me/password", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ currentPassword, newPassword })
    });
  }
};
