"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import type { CustomerSummary } from "@/lib/auth/types";
import type { CustomerActor, CustomerCapability } from "@/lib/auth/capabilities";
import { AuthApiError } from "@/lib/auth/types";
import { authService } from "@/services/auth-service";
import { getCustomerCapabilities } from "@/services/capability-service";

type SessionStatus = "loading" | "authenticated" | "guest" | "error";
type CapabilityStatus = "loading" | "ready" | "error";

interface CustomerSessionValue {
  status: SessionStatus;
  customer: CustomerSummary | null;
  actor: CustomerActor | null;
  capabilityStatus: CapabilityStatus;
  capabilities: readonly CustomerCapability[];
  hasCapability: (capability: CustomerCapability) => boolean;
  reload: () => Promise<void>;
  logout: () => Promise<void>;
}

const CustomerSessionContext = createContext<CustomerSessionValue | null>(null);

export function CustomerSessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>("loading");
  const [customer, setCustomer] = useState<CustomerSummary | null>(null);
  const [actor, setActor] = useState<CustomerActor | null>(null);
  const [capabilityStatus, setCapabilityStatus] = useState<CapabilityStatus>("loading");
  const [capabilities, setCapabilities] = useState<CustomerCapability[]>([]);

  const loadCapabilities = useCallback(async () => {
    setCapabilityStatus("loading");
    try {
      const projection = await getCustomerCapabilities();
      setActor(projection.actor);
      setCapabilities(projection.capabilities);
      setCapabilityStatus("ready");
    } catch {
      setActor(null);
      setCapabilities([]);
      setCapabilityStatus("error");
    }
  }, []);

  const reload = useCallback(async () => {
    try {
      const session = await authService.me();
      setCustomer(session.customer);
      setStatus("authenticated");
    } catch (cause) {
      setCustomer(null);
      setStatus(cause instanceof AuthApiError && cause.status === 401 ? "guest" : "error");
    }
    await loadCapabilities();
  }, [loadCapabilities]);

  useEffect(() => { void Promise.resolve().then(reload); }, [reload]);

  const logout = useCallback(async () => {
    try { await authService.logout(); }
    finally {
      setCustomer(null);
      setStatus("guest");
      await loadCapabilities();
    }
  }, [loadCapabilities]);

  const hasCapability = useCallback((capability: CustomerCapability) => capabilityStatus === "ready" && capabilities.includes(capability), [capabilities, capabilityStatus]);
  const value = useMemo(() => ({ status, customer, actor, capabilityStatus, capabilities, hasCapability, reload, logout }), [actor, capabilities, capabilityStatus, customer, hasCapability, logout, reload, status]);
  return <CustomerSessionContext.Provider value={value}>{children}</CustomerSessionContext.Provider>;
}

export function useCustomerSession() {
  const value = useContext(CustomerSessionContext);
  if (!value) throw new Error("useCustomerSession must be used inside CustomerSessionProvider");
  return value;
}
