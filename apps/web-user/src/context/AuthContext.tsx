"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { API_BASE_URL, apiClient, unwrapApiData } from "@/lib/api";

export interface UserRole {
  roleCode: string;
  roleName?: string;
  scopeType?: string;
  scopeId: string | null;
}

export interface UserProfile {
  principalId: string;
  email: string;
  principalType: "CUSTOMER" | "EMPLOYEE";
  roles: UserRole[];
  permissions: string[];
  displayName?: string;
  firstName?: string;
  lastName?: string;
}

interface SessionProjection {
  isAuthenticated?: boolean;
  authenticated?: boolean;
}

interface AuthContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  user: UserProfile | null;
  login: (returnUrl?: string) => void;
  logout: () => Promise<void>;
  checkSession: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

function normalizeProfile(value: unknown): UserProfile | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const source = value as Record<string, unknown>;
  const principalId = typeof source.principalId === "string" ? source.principalId : typeof source.principal_id === "string" ? source.principal_id : "";
  const principalType = source.principalType ?? source.principal_type;
  if (!principalId || (principalType !== "CUSTOMER" && principalType !== "EMPLOYEE")) return null;
  const roles = Array.isArray(source.roles) ? source.roles.flatMap((item): UserRole[] => {
    if (typeof item === "string" && item) return [{ roleCode: item, scopeId: null }];
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    const role = item as Record<string, unknown>;
    if (typeof role.roleCode !== "string" || !role.roleCode) return [];
    return [{
      roleCode: role.roleCode,
      ...(typeof role.roleName === "string" ? { roleName: role.roleName } : {}),
      ...(typeof role.scopeType === "string" ? { scopeType: role.scopeType } : {}),
      scopeId: typeof role.scopeId === "string" ? role.scopeId : null
    }];
  }) : [];
  return {
    principalId,
    principalType,
    email: typeof source.email === "string" ? source.email : "",
    roles,
    permissions: Array.isArray(source.permissions) ? source.permissions.filter((item): item is string => typeof item === "string") : [],
    ...(typeof source.displayName === "string" ? { displayName: source.displayName } : {}),
    ...(typeof source.firstName === "string" ? { firstName: source.firstName } : {}),
    ...(typeof source.lastName === "string" ? { lastName: source.lastName } : {})
  };
}

function safeReturnUrl(returnUrl?: string): string {
  if (typeof window === "undefined") return "/";
  const candidate = new URL(returnUrl || "/", window.location.origin);
  return candidate.origin === window.location.origin ? candidate.href : `${window.location.origin}/`;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<UserProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const checkSession = useCallback(async () => {
    setIsLoading(true);
    try {
      const sessionResponse = await apiClient.get("/public/auth/session");
      const session = unwrapApiData<SessionProjection>(sessionResponse.data);
      if (session?.isAuthenticated !== true && session?.authenticated !== true) {
        setIsAuthenticated(false);
        setUser(null);
        return;
      }
      const profileResponse = await apiClient.get("/users/me");
      const profile = normalizeProfile(unwrapApiData<unknown>(profileResponse.data));
      if (!profile) throw new Error("Identity profile does not match the expected contract.");
      setIsAuthenticated(true);
      setUser(profile);
    } catch {
      setIsAuthenticated(false);
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => { void Promise.resolve().then(checkSession); }, [checkSession]);

  const login = useCallback((returnUrl?: string) => {
    const redirectUri = encodeURIComponent(safeReturnUrl(returnUrl));
    // OIDC PKCE must leave the Next.js application so the browser can follow the backend 302 chain.
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.href = `${API_BASE_URL}/public/auth/login?redirect_uri=${redirectUri}`;
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiClient.post("/public/auth/logout");
    } finally {
      setIsAuthenticated(false);
      setUser(null);
      router.replace("/");
      router.refresh();
    }
  }, [router]);

  const value = useMemo(() => ({ isAuthenticated, isLoading, user, login, logout, checkSession }), [checkSession, isAuthenticated, isLoading, login, logout, user]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within AuthProvider");
  return context;
}
