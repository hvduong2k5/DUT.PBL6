"use client";

import { createContext, useCallback, useContext, useMemo, type ReactNode } from "react";
import type { IdentityProfile } from "@/lib/auth/identity";
import { toEmployeeSession, type EmployeePermission, type EmployeeSession } from "@/features/employee/auth/types";
import { useAuth } from "./auth-provider";

interface EmployeeSessionValue {
  employee: IdentityProfile;
  session: EmployeeSession;
  hasPermission: (permission: EmployeePermission | string) => boolean;
  hasRole: (roleCode: string) => boolean;
  logout: () => Promise<void>;
}

const EmployeeSessionContext = createContext<EmployeeSessionValue | null>(null);

export function EmployeeSessionProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const employee = auth.profile?.principalType === "EMPLOYEE" ? auth.profile : null;
  const session = useMemo(() => employee ? toEmployeeSession(employee) : undefined, [employee]);
  // Gates across the admin UI check draft-vocabulary permission names, so
  // they must read the projected session (where IAM grants are expanded
  // into those names), not the raw identity profile permissions.
  const hasPermission = useCallback((permission: string) => Boolean(session?.permissions.includes(permission as EmployeePermission)), [session]);
  const hasRole = useCallback((roleCode: string) => Boolean(employee?.roles.some((role) => role.roleCode === roleCode)), [employee]);
  const value = useMemo(() => employee && session ? { employee, session, hasPermission, hasRole, logout: auth.logout } : null, [auth.logout, employee, hasPermission, hasRole, session]);
  return <EmployeeSessionContext.Provider value={value}>{children}</EmployeeSessionContext.Provider>;
}

export function useEmployeeSession() {
  const value = useContext(EmployeeSessionContext);
  if (!value) throw new Error("useEmployeeSession must be used inside an authenticated EmployeeSessionProvider");
  return value;
}
