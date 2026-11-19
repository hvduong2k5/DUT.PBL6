import { type IdentityProfile } from "@/lib/auth/identity";
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
    const response = await fetch("/api/auth/me", { credentials: "include", cache: "no-store" });
    if (response.status === 401) return null;
    if (!response.ok) throw new AuthApiError(response.status, { code: "IDENTITY_UNAVAILABLE", message: "Chưa thể xác minh danh tính." });
    const session = await response.json();
    if (session.authenticated !== true || !session.principal) throw new AuthApiError(502, { code: "IDENTITY_PROFILE_INVALID", message: "Hồ sơ định danh không đúng contract." });
    return { ...session.principal, provisioningStatus: "ACTIVE", email: session.principal.email ?? "", customerId: session.customer?.id, customerStatus: session.customer?.status } as IdentityProfile;
  },
  loginUrl(returnUrl: string) {
    const url = new URL("/api/auth/oidc/login", window.location.origin);
    url.searchParams.set("returnUrl", new URL(returnUrl, window.location.origin).pathname);
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
