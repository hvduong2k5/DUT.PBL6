"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import type { CustomerSummary, PrincipalSummary } from "@/lib/auth/types";
import type { CustomerActor, CustomerCapability } from "@/lib/auth/capabilities";
import { CapabilityApiError, capabilityFailureAction, getCustomerCapabilities } from "@/services/capability-service";

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
  const sessionRecheckKey = useRef<string | null>(null);
  const authReload = useRef(auth.reload);
  useEffect(() => { authReload.current = auth.reload; }, [auth.reload]);

  const loadCapabilities = useCallback(async (allowed: boolean, identityKey: string) => {
    if (!allowed) {
      setActor(null);
      setCapabilities([]);
      setCapabilityStatus("ready");
      return;
    }
    setCapabilityStatus("loading");
    try {
      const projection = await getCustomerCapabilities();
      sessionRecheckKey.current = null;
      setActor(projection.actor);
      setCapabilities(projection.capabilities);
      setCapabilityStatus("ready");
    } catch (cause) {
      setActor(null);
      setCapabilities([]);
      const action = cause instanceof CapabilityApiError ? capabilityFailureAction(cause.status, cause.code) : "error";
      if (action === "forbidden") {
        setCapabilityStatus("forbidden");
        return;
      }
      if (action === "session" && sessionRecheckKey.current !== identityKey) {
        // The credential layer rejected the session while the auth probe
        // still shows the user as signed in. Re-probe the session once: if
        // it really ended, the auth layer flips to guest and the account
        // guard redirects to the login page. The error state below keeps a
        // manual retry available in the meantime, and a failed capability
        // check alone never clears the authenticated state.
        sessionRecheckKey.current = identityKey;
        void authReload.current();
      }
      if (action === "error" && cause instanceof CapabilityApiError && cause.status === 401) {
        console.warn(`capabilities: unexpected 401 error code "${cause.code}"`);
      }
      setCapabilityStatus("error");
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
  const customerAccess = Boolean(customerIdentity?.customerId);
  const principal = useMemo<PrincipalSummary | null>(() => customerIdentity ? {
    principalId: customerIdentity.principalId,
    principalType: "CUSTOMER",
    displayName: customerIdentity.displayName,
    email: customerIdentity.email,
    roles: customerIdentity.roles,
    permissions: customerIdentity.permissions,
    customerId: customerIdentity.customerId
  } : null, [customerIdentity]);
  const customer = useMemo<CustomerSummary | null>(() => customerIdentity?.customerId ? {
    id: customerIdentity.customerId,
    displayName: customerIdentity.displayName,
    email: customerIdentity.email,
    status: customerIdentity.customerStatus ?? "ACTIVE"
  } : null, [customerIdentity]);

  const identityKey = customerIdentity?.principalId ?? "anonymous";

  useEffect(() => {
    void Promise.resolve().then(() => loadCapabilities(customerAccess, identityKey));
  }, [customerAccess, identityKey, loadCapabilities]);

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
