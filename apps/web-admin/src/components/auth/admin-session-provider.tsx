"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import type { AdminPermission, AdminSession } from "@/lib/auth/types";
import { getAdminSession } from "@/services/admin-session-service";

interface AdminSessionContextValue {
  status: "loading" | "authenticated" | "error";
  session: AdminSession | null;
  hasPermission: (permission: AdminPermission) => boolean;
  reload: () => Promise<void>;
}

const AdminSessionContext = createContext<AdminSessionContextValue | null>(null);

export function AdminSessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AdminSessionContextValue["status"]>("loading");
  const [session, setSession] = useState<AdminSession | null>(null);
  const reload = useCallback(async () => {
    setStatus("loading");
    try { setSession(await getAdminSession()); setStatus("authenticated"); }
    catch { setSession(null); setStatus("error"); }
  }, []);
  useEffect(() => { void Promise.resolve().then(reload); }, [reload]);
  const hasPermission = useCallback((permission: AdminPermission) => Boolean(session?.permissions.includes(permission)), [session]);
  const value = useMemo(() => ({ status, session, hasPermission, reload }), [hasPermission, reload, session, status]);
  return <AdminSessionContext.Provider value={value}>{children}</AdminSessionContext.Provider>;
}

export function useAdminSession() {
  const value = useContext(AdminSessionContext);
  if (!value) throw new Error("useAdminSession must be used inside AdminSessionProvider");
  return value;
}
