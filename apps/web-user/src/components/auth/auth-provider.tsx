"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import type { IdentityProfile } from "@/lib/auth/identity";
import { identityService } from "@/services/identity-service";
import { authService } from "@/services/auth-service";

export type AuthStatus = "loading" | "guest" | "authenticated" | "error";

interface AuthContextValue {
  status: AuthStatus;
  profile: IdentityProfile | null;
  user: IdentityProfile | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isCustomer: boolean;
  isEmployee: boolean;
  hasRole: (roleCode: string) => boolean;
  hasPermission: (permissionCode: string) => boolean;
  login: (returnUrl?: string) => void;
  logout: () => Promise<void>;
  reload: () => Promise<void>;
  refreshProfile: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function safeReturnUrl(value?: string): string {
  if (typeof window === "undefined") return "/";
  try {
    const url = new URL(value || "/", window.location.origin);
    return url.origin === window.location.origin ? url.href : `${window.location.origin}/`;
  } catch {
    return `${window.location.origin}/`;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [profile, setProfile] = useState<IdentityProfile | null>(null);

  const reload = useCallback(async () => {
    setStatus("loading");
    try {
      const current = await identityService.currentProfile();
      setProfile(current);
      setStatus(current ? "authenticated" : "guest");
    } catch {
      setProfile(null);
      setStatus("error");
    }
  }, []);

  useEffect(() => { void Promise.resolve().then(reload); }, [reload]);

  const login = useCallback((returnUrl?: string) => {
    window.location.assign(identityService.loginUrl(safeReturnUrl(returnUrl)));
  }, []);

  const logout = useCallback(async () => {
    let logoutUrl: string | undefined;
    try {
      const result = await authService.logout();
      logoutUrl = result?.logoutUrl;
    } catch {
      // The local sign-out below still applies when the auth route is
      // unreachable; the session cookies are httpOnly and expire with
      // the refresh token at the latest.
    }
    if (logoutUrl) {
      // OIDC mode (F-13): end the provider-side SSO session, which
      // returns the browser via /api/auth/oidc/logged-out. Stay pending
      // so callers cannot navigate locally and race the redirect.
      window.location.assign(logoutUrl);
      return new Promise<void>(() => {});
    }
    try { await identityService.logout(); } catch { /* local sign-out proceeds */ }
    setProfile(null);
    setStatus("guest");
    router.replace("/");
    router.refresh();
  }, [router]);

  const hasRole = useCallback((roleCode: string) => Boolean(profile?.roles.some((role) => role.roleCode.toLocaleUpperCase("en") === roleCode.toLocaleUpperCase("en"))), [profile]);
  const hasPermission = useCallback((permissionCode: string) => Boolean(profile?.permissions.includes(permissionCode)), [profile]);
  const value = useMemo(() => ({
    status,
    profile,
    user: profile,
    isAuthenticated: status === "authenticated",
    isLoading: status === "loading",
    isCustomer: profile?.principalType === "CUSTOMER",
    isEmployee: profile?.principalType === "EMPLOYEE",
    hasRole,
    hasPermission,
    login,
    logout,
    reload,
    refreshProfile: reload
  }), [hasPermission, hasRole, login, logout, profile, reload, status]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
