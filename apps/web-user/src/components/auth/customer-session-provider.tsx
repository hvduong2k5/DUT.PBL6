"use client";

import { createContext, useCallback, useContext, useMemo, type ReactNode } from "react";
import { useAuth } from "@/context/AuthContext";
import { CUSTOMER_CAPABILITIES, type CustomerActor, type CustomerCapability } from "@/lib/auth/capabilities";
import type { CustomerSummary, PrincipalRole, PrincipalSummary } from "@/lib/auth/types";

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
  logout: () => Promise<void>;
}

const CustomerSessionContext = createContext<CustomerSessionValue | null>(null);
const STANDARD_CAPABILITIES: CustomerCapability[] = [
  "CATALOG_VIEW", "CART_MANAGE", "CHECKOUT_CREATE", "GUEST_ORDER_TRACK", "SUPPORT_CREATE",
  "PROFILE_MANAGE", "ADDRESS_MANAGE", "ORDER_HISTORY_VIEW", "LOYALTY_VIEW", "REVIEW_CREATE"
];
const SCOPE_TYPES = new Set<NonNullable<PrincipalRole["scopeType"]>>(["GLOBAL", "STORE", "WAREHOUSE", "REGION", "DEPARTMENT", "BRANCH", "BUSINESS_UNIT"]);

export function CustomerSessionProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const status: SessionStatus = auth.isLoading ? "loading" : auth.isAuthenticated && auth.user ? "authenticated" : "guest";
  const isEmployee = auth.user?.principalType === "EMPLOYEE";
  // Product rule: an Employee account also receives the Customer experience.
  const customerAccess = status === "authenticated";
  const displayName = auth.user?.displayName || [auth.user?.firstName, auth.user?.lastName].filter(Boolean).join(" ") || auth.user?.email || "Tài khoản O Mạ";

  const principal = useMemo<PrincipalSummary | null>(() => auth.user ? {
    principalId: auth.user.principalId,
    principalType: auth.user.principalType,
    displayName,
    email: auth.user.email,
    roles: auth.user.roles.map((role): PrincipalRole => ({
      roleCode: role.roleCode,
      ...(role.roleName ? { roleName: role.roleName } : {}),
      ...(role.scopeType && SCOPE_TYPES.has(role.scopeType as NonNullable<PrincipalRole["scopeType"]>) ? { scopeType: role.scopeType as NonNullable<PrincipalRole["scopeType"]> } : {}),
      scopeId: role.scopeId
    })),
    permissions: auth.user.permissions,
    customerId: auth.user.principalId
  } : null, [auth.user, displayName]);

  const customer = useMemo<CustomerSummary | null>(() => auth.user ? {
    id: auth.user.principalId,
    displayName,
    status: "ACTIVE",
    email: auth.user.email
  } : null, [auth.user, displayName]);

  const actor: CustomerActor | null = auth.user ? auth.user.roles.some((role) => role.roleCode.toUpperCase().includes("B2B")) ? "B2B" : "REGISTERED" : null;
  const capabilities = useMemo<CustomerCapability[]>(() => {
    if (!auth.user) return [];
    const identityCapabilities = auth.user.permissions.filter((permission): permission is CustomerCapability => CUSTOMER_CAPABILITIES.includes(permission as CustomerCapability));
    const base = identityCapabilities.length ? identityCapabilities : STANDARD_CAPABILITIES;
    return actor === "B2B" ? [...new Set([...base, "B2B_COMPANY_VIEW", "B2B_QUOTE_CREATE", "B2B_QUOTE_VIEW", "B2B_QUOTE_ACCEPT"] as CustomerCapability[])] : base;
  }, [actor, auth.user]);

  const hasCapability = useCallback((capability: CustomerCapability) => capabilities.includes(capability), [capabilities]);
  const hasRole = useCallback((roleCode: string) => Boolean(auth.user?.roles.some((role) => role.roleCode === roleCode)), [auth.user]);
  const hasPermission = useCallback((permissionCode: string) => Boolean(auth.user?.permissions.includes(permissionCode)), [auth.user]);
  const value = useMemo<CustomerSessionValue>(() => ({
    status,
    customer,
    principal,
    customerAccess,
    isEmployee,
    actor,
    capabilityStatus: auth.isLoading ? "loading" : "ready",
    capabilities,
    hasCapability,
    hasRole,
    hasPermission,
    reload: auth.checkSession,
    logout: auth.logout
  }), [actor, auth.checkSession, auth.isLoading, auth.logout, capabilities, customer, customerAccess, hasCapability, hasPermission, hasRole, isEmployee, principal, status]);

  return <CustomerSessionContext.Provider value={value}>{children}</CustomerSessionContext.Provider>;
}

export function useCustomerSession() {
  const value = useContext(CustomerSessionContext);
  if (!value) throw new Error("useCustomerSession must be used inside CustomerSessionProvider");
  return value;
}
