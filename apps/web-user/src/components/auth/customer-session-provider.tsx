"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import type { CustomerSummary, PrincipalSummary } from "@/lib/auth/types";
import type { CustomerActor, CustomerCapability } from "@/lib/auth/capabilities";
import { CapabilityApiError, getCustomerCapabilities } from "@/services/capability-service";

type SessionStatus = "loading" | "authenticated" | "guest" | "forbidden" | "error";
type CapabilityStatus = "loading" | "ready" | "forbidden" | "error";

interface CustomerSessionValue {
  status: SessionStatus;
  customer: CustomerSummary | null;
  principal: PrincipalSummary | null;
  customerAccess: boolean;
  actor: CustomerActor | null;
  capabilityStatus: CapabilityStatus;
  capabilities: readonly CustomerCapability[];
  hasCapability: (capability: CustomerCapability) => boolean;
  hasRole: (roleCode: string) => boolean;
  hasPermission: (permissionCode: string) => boolean;
  reload: () => Promise<void>;
  logout: () => Promise<void>;
}

const CustomerSessionContext = createContext<CustomerSessionValue | null>(null);

export function CustomerSessionProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const [actor, setActor] = useState<CustomerActor | null>(null);
  const [capabilityStatus, setCapabilityStatus] = useState<CapabilityStatus>("loading");
  const [capabilities, setCapabilities] = useState<CustomerCapability[]>([]);

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
      setCapabilityStatus(cause instanceof CapabilityApiError && (cause.status === 401 || cause.status === 403) ? "forbidden" : "error");
    }
  }, []);

  const customerIdentity = auth.profile?.principalType === "CUSTOMER" ? auth.profile : null;
  const status: SessionStatus = auth.status === "loading"
    ? "loading"
    : auth.status === "guest"
      ? "guest"
      : auth.status === "error"
        ? "error"
        : customerIdentity
          ? "authenticated"
          : "forbidden";
  const customerAccess = Boolean(customerIdentity);
  const principal = useMemo<PrincipalSummary | null>(() => customerIdentity ? {
    principalId: customerIdentity.principalId,
    principalType: "CUSTOMER",
    displayName: customerIdentity.displayName,
    email: customerIdentity.email,
    roles: customerIdentity.roles,
    permissions: customerIdentity.permissions,
    customerId: customerIdentity.principalId
  } : null, [customerIdentity]);
  const customer = useMemo<CustomerSummary | null>(() => customerIdentity ? {
    id: customerIdentity.principalId,
    displayName: customerIdentity.displayName,
    email: customerIdentity.email,
    status: "ACTIVE"
  } : null, [customerIdentity]);

  useEffect(() => {
    void Promise.resolve().then(() => loadCapabilities(customerAccess));
  }, [customerAccess, loadCapabilities]);

  const hasCapability = useCallback((capability: CustomerCapability) => capabilityStatus === "ready" && capabilities.includes(capability), [capabilities, capabilityStatus]);
  const hasRole = useCallback((roleCode: string) => Boolean(principal?.roles.some((role) => role.roleCode === roleCode)), [principal]);
  const hasPermission = useCallback((permissionCode: string) => Boolean(principal?.permissions.includes(permissionCode)), [principal]);
  const value = useMemo(() => ({ status, customer, principal, customerAccess, actor, capabilityStatus, capabilities, hasCapability, hasRole, hasPermission, reload: auth.reload, logout: auth.logout }), [actor, auth.logout, auth.reload, capabilities, capabilityStatus, customer, customerAccess, hasCapability, hasPermission, hasRole, principal, status]);
  return <CustomerSessionContext.Provider value={value}>{children}</CustomerSessionContext.Provider>;
}

export function useCustomerSession() {
  const value = useContext(CustomerSessionContext);
  if (!value) throw new Error("useCustomerSession must be used inside CustomerSessionProvider");
  return value;
}
