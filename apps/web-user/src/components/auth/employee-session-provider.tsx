"use client";

import { createContext, useCallback, useContext, useMemo, type ReactNode } from "react";
import type { IdentityProfile } from "@/lib/auth/identity";
import { useAuth } from "./auth-provider";

interface EmployeeSessionValue {
  employee: IdentityProfile;
  hasPermission: (permission: string) => boolean;
  hasRole: (roleCode: string) => boolean;
  logout: () => Promise<void>;
}

const EmployeeSessionContext = createContext<EmployeeSessionValue | null>(null);

export function EmployeeSessionProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const employee = auth.profile?.principalType === "EMPLOYEE" ? auth.profile : null;
  const hasPermission = useCallback((permission: string) => Boolean(employee?.permissions.includes(permission)), [employee]);
  const hasRole = useCallback((roleCode: string) => Boolean(employee?.roles.some((role) => role.roleCode === roleCode)), [employee]);
  const value = useMemo(() => employee ? { employee, hasPermission, hasRole, logout: auth.logout } : null, [auth.logout, employee, hasPermission, hasRole]);
  return <EmployeeSessionContext.Provider value={value}>{children}</EmployeeSessionContext.Provider>;
}

export function useEmployeeSession() {
  const value = useContext(EmployeeSessionContext);
  if (!value) throw new Error("useEmployeeSession must be used inside an authenticated EmployeeSessionProvider");
  return value;
}
