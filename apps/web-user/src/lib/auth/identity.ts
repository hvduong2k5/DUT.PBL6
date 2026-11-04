import type { PrincipalRole, PrincipalType } from "./types";

export interface IdentityProfile {
  customerId?: string;
  customerStatus?: "ACTIVE" | "PENDING_VERIFICATION";
  principalId: string;
  principalType: PrincipalType;
  email: string;
  displayName: string;
  provisioningStatus: string;
  roles: PrincipalRole[];
  permissions: string[];
}

export interface IdentitySessionProbe {
  authenticated: boolean;
  principalId?: string;
  email?: string;
  expiresAt?: string;
}

function record(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : undefined;
}

function text(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

export function unwrapIdentityData(value: unknown): unknown {
  const source = record(value);
  return source && "data" in source ? source.data : value;
}

export function parseIdentitySession(value: unknown): IdentitySessionProbe | undefined {
  const source = record(unwrapIdentityData(value));
  const authenticated = source?.authenticated ?? source?.isAuthenticated;
  if (!source || typeof authenticated !== "boolean") return undefined;
  return {
    authenticated,
    ...(text(source.principalId) ? { principalId: text(source.principalId) } : {}),
    ...(text(source.email) ? { email: text(source.email) } : {}),
    ...(text(source.expiresAt) ? { expiresAt: text(source.expiresAt) } : {})
  };
}

export function parseIdentityProfile(value: unknown): IdentityProfile | undefined {
  const source = record(unwrapIdentityData(value));
  const principalId = text(source?.principalId) ?? text(source?.principal_id);
  const principalType = source?.principalType ?? source?.principal_type;
  if (!source || !principalId || (principalType !== "CUSTOMER" && principalType !== "EMPLOYEE")) return undefined;

  const roles = Array.isArray(source.roles) ? source.roles.flatMap((item): PrincipalRole[] => {
    if (typeof item === "string" && item.trim()) return [{ roleCode: item.trim() }];
    const role = record(item);
    const roleCode = text(role?.roleCode);
    if (!role || !roleCode) return [];
    const scopeType = text(role.scopeType);
    return [{
      roleCode,
      ...(text(role.roleName) ? { roleName: text(role.roleName) } : {}),
      ...(scopeType ? { scopeType: scopeType as PrincipalRole["scopeType"] } : {}),
      ...(role.scopeId === null || text(role.scopeId) ? { scopeId: role.scopeId === null ? null : text(role.scopeId) } : {})
    }];
  }) : [];

  const email = text(source.email) ?? "";
  const displayName = text(source.displayName)
    ?? [text(source.firstName), text(source.lastName)].filter(Boolean).join(" ")
    ?? email
    ?? principalId;

  return {
    principalId,
    principalType,
    email,
    displayName: displayName || email || principalId,
    provisioningStatus: text(source.provisioningStatus) ?? "ACTIVE",
    roles,
    permissions: Array.isArray(source.permissions)
      ? [...new Set(source.permissions.map(text).filter((item): item is string => Boolean(item)))]
      : []
  };
}
