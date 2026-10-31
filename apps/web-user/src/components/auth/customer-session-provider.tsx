"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import type { CustomerSummary, PrincipalSummary } from "@/lib/auth/types";
import type { CustomerActor, CustomerCapability } from "@/lib/auth/capabilities";
import { AuthApiError } from "@/lib/auth/types";
import { authService } from "@/services/auth-service";
import { CapabilityApiError, getCustomerCapabilities } from "@/services/capability-service";

type SessionStatus = "loading" | "authenticated" | "guest" | "forbidden" | "error";
type CapabilityStatus = "loading" | "ready" | "forbidden" | "error";

interface CustomerSessionValue {
  status: SessionStatus;
  customer: CustomerSummary | null;
  principal: PrincipalSummary | null;
  customerAccess: boolean;
  isEmployee: boolean;
  actor: CustomerActor | null;
  capabilityStatus: CapabilityStatus;
  capabilities: readonly CustomerCapability[];
  hasCapability: (capability: CustomerCapability) => boolean;
  hasRole: (roleCode: string) => boolean;
  hasPermission: (permissionCode: string) => boolean;
  reload: () => Promise<void>;
  logout: () => Promise<string | undefined>;
}

const CustomerSessionContext = createContext<CustomerSessionValue | null>(null);

export function CustomerSessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>("loading");
  const [customer, setCustomer] = useState<CustomerSummary | null>(null);
  const [principal, setPrincipal] = useState<PrincipalSummary | null>(null);
  const [customerAccess, setCustomerAccess] = useState(false);
  const [actor, setActor] = useState<CustomerActor | null>(null);
  const [capabilityStatus, setCapabilityStatus] = useState<CapabilityStatus>("loading");
  const [capabilities, setCapabilities] = useState<CustomerCapability[]>([]);
  const [sessionExpiresAt, setSessionExpiresAt] = useState<string | undefined>();

  const loadCapabilities = useCallback(async (allowed: boolean) => {
    if (!allowed) {
      setActor(null);
      setCapabilities([]);
      setCapabilityStatus("ready");
      return;
    }
    setCapabilityStatus("loading");
    try {
      const projection = await getCustomerCapabilities();
      setActor(projection.actor);
      setCapabilities(projection.capabilities);
      setCapabilityStatus("ready");
    } catch (cause) {
      setActor(null);
      setCapabilities([]);
      if (cause instanceof CapabilityApiError && cause.status === 401) throw cause;
      setCapabilityStatus(cause instanceof CapabilityApiError && cause.status === 403 ? "forbidden" : "error");
    }
  }, []);

  const reload = useCallback(async () => {
    try {
      const session = await authService.me();
      setCustomer(session.customer);
      setPrincipal(session.principal);
      setCustomerAccess(session.customerAccess);
      setStatus("authenticated");
      setSessionExpiresAt(session.sessionExpiresAt);
      await loadCapabilities(session.customerAccess);
    } catch (cause) {
      setCustomer(null);
      setSessionExpiresAt(undefined);
      setPrincipal(null);
      setCustomerAccess(false);
      setActor(null);
      setCapabilities([]);
      setCapabilityStatus("ready");
      const unauthorized = cause instanceof AuthApiError && cause.status === 401 || cause instanceof CapabilityApiError && cause.status === 401;
      const forbidden = cause instanceof AuthApiError && cause.status === 403 || cause instanceof CapabilityApiError && cause.status === 403;
      setStatus(unauthorized ? "guest" : forbidden ? "forbidden" : "error");
    }
  }, [loadCapabilities]);

  useEffect(() => { void Promise.resolve().then(reload); }, [reload]);
  useEffect(() => {
    if (status !== "authenticated" || !sessionExpiresAt) return;
    const expiry = Date.parse(sessionExpiresAt);
    if (!Number.isFinite(expiry)) return;
    const timer = setTimeout(() => { void reload(); }, Math.max(1000, expiry - Date.now() - 25_000));
    return () => clearTimeout(timer);
  }, [status, sessionExpiresAt, reload]);

  const logout = useCallback(async () => {
    try { return (await authService.logout())?.logoutUrl; }
    finally {
      setCustomer(null);
      setPrincipal(null);
      setCustomerAccess(false);
      setStatus("guest");
      setSessionExpiresAt(undefined);
      await loadCapabilities(false);
    }
  }, [loadCapabilities]);

  const hasCapability = useCallback((capability: CustomerCapability) => capabilityStatus === "ready" && capabilities.includes(capability), [capabilities, capabilityStatus]);
  const hasRole = useCallback((roleCode: string) => Boolean(principal?.roles.some((role) => role.roleCode === roleCode)), [principal]);
  const hasPermission = useCallback((permissionCode: string) => Boolean(principal?.permissions.includes(permissionCode)), [principal]);
  const isEmployee = principal?.principalType === "EMPLOYEE";
  const value = useMemo(() => ({ status, customer, principal, customerAccess, isEmployee, actor, capabilityStatus, capabilities, hasCapability, hasRole, hasPermission, reload, logout }), [actor, capabilities, capabilityStatus, customer, customerAccess, hasCapability, hasPermission, hasRole, isEmployee, logout, principal, reload, status]);
  return <CustomerSessionContext.Provider value={value}>{children}</CustomerSessionContext.Provider>;
}

export function useCustomerSession() {
  const value = useContext(CustomerSessionContext);
  if (!value) throw new Error("useCustomerSession must be used inside CustomerSessionProvider");
  return value;
}
