import type { AuthenticatedSession, PrincipalRole, PrincipalType } from "./types";

export interface CustomerCoreUser {
  customer_id?: string;
  customerId?: string;
  principal_id?: string;
  principalId?: string;
  principal_type?: PrincipalType;
  principalType?: PrincipalType;
  phone_number?: string;
  full_name?: string;
  displayName?: string;
  email?: string;
  avatar_url?: string;
  membership_tier?: "BRONZE" | "SILVER" | "GOLD" | "DIAMOND";
  loyalty_points?: number;
  roles?: Array<string | { roleCode?: string; roleName?: string; scopeType?: string; scopeId?: string | null }>;
  permissions?: string[];
}

export interface CustomerCoreTokens {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

export interface CustomerCoreAuthResponse {
  user: CustomerCoreUser;
  tokens: CustomerCoreTokens;
}

const SCOPE_TYPES = new Set(["GLOBAL", "STORE", "WAREHOUSE", "REGION", "DEPARTMENT", "BRANCH", "BUSINESS_UNIT"]);

function clean(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

function mapRoles(value: CustomerCoreUser["roles"], principalType: PrincipalType): PrincipalRole[] {
  const roles: PrincipalRole[] = [];
  for (const item of value ?? []) {
    if (typeof item === "string") {
      const roleCode = clean(item);
      if (roleCode) roles.push({ roleCode });
      continue;
    }
    const roleCode = clean(item.roleCode);
    if (!roleCode) continue;
    const scopeType = clean(item.scopeType);
    roles.push({
      roleCode,
      ...(clean(item.roleName) ? { roleName: clean(item.roleName) } : {}),
      ...(scopeType && SCOPE_TYPES.has(scopeType) ? { scopeType: scopeType as PrincipalRole["scopeType"] } : {}),
      ...(item.scopeId === null || clean(item.scopeId) ? { scopeId: item.scopeId === null ? null : clean(item.scopeId) } : {})
    });
  }
  if (principalType === "CUSTOMER" && !roles.some((role) => role.roleCode === "CUSTOMER")) roles.unshift({ roleCode: "CUSTOMER" });
  return roles.filter((role, index) => roles.findIndex((candidate) => candidate.roleCode === role.roleCode && candidate.scopeType === role.scopeType && candidate.scopeId === role.scopeId) === index);
}

export function mapCustomerCoreSession(user: CustomerCoreUser): AuthenticatedSession | undefined {
  const customerId = clean(user.customer_id) ?? clean(user.customerId);
  const explicitType = user.principal_type ?? user.principalType;
  const principalType: PrincipalType | undefined = explicitType === "CUSTOMER" || explicitType === "EMPLOYEE"
    ? explicitType
    : customerId
      ? "CUSTOMER"
      : undefined;
  const principalId = clean(user.principal_id) ?? clean(user.principalId) ?? customerId;
  if (!principalType || !principalId) return undefined;

  const displayName = clean(user.full_name) ?? clean(user.displayName) ?? clean(user.email) ?? principalId;
  const roles = mapRoles(user.roles, principalType);
  const permissions = [...new Set((user.permissions ?? []).map(clean).filter((value): value is string => Boolean(value)))];
  const customerAccess = Boolean(customerId);
  return {
    authenticated: true,
    principal: {
      principalId,
      principalType,
      displayName,
      ...(clean(user.email) ? { email: clean(user.email) } : {}),
      roles,
      permissions,
      ...(customerId ? { customerId } : {})
    },
    customerAccess,
    customer: customerId ? {
      id: customerId,
      displayName,
      status: "ACTIVE",
      ...(clean(user.phone_number) ? { phoneNumber: clean(user.phone_number) } : {}),
      ...(clean(user.email) ? { email: clean(user.email) } : {}),
      ...(user.membership_tier ? { membershipTier: user.membership_tier } : {}),
      ...(Number.isFinite(user.loyalty_points) ? { loyaltyPoints: user.loyalty_points } : {})
    } : null
  };
}

export function toCustomerCoreLogin(input: { phoneNumber: string; password: string }) {
  return { phone_number: input.phoneNumber, password: input.password };
}

export function toCustomerCoreRegistration(input: { fullName: string; phoneNumber: string; email?: string; password: string }) {
  return {
    phone_number: input.phoneNumber,
    password: input.password,
    full_name: input.fullName,
    ...(input.email ? { email: input.email } : {})
  };
}
