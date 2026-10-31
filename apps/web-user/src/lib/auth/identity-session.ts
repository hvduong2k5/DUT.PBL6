import { mapCustomerCoreSession, type CustomerCoreUser } from "./customer-core";
import { verifiedAccess } from "./oidc";

export class IdentitySessionError extends Error {
  constructor(readonly status: 401 | 403 | 503, readonly code: string) { super(code); }
}

export async function identitySession(accessToken: string) {
  const claims = await verifiedAccess(accessToken);
  const base = process.env.CUSTOMER_CORE_UPSTREAM_URL?.replace(/\/$/u, "");
  if (!base) throw new IdentitySessionError(503, "IDENTITY_UNAVAILABLE");
  const read = async (path: string) => fetch(`${base}${path}`, { headers: { Accept: "application/json", Authorization: `Bearer ${accessToken}` }, cache: "no-store", redirect: "error", signal: AbortSignal.timeout(3000) });
  const response = await read("/users/me");
  if (response.status === 401) throw new IdentitySessionError(401, "AUTH_REQUIRED");
  if (response.status === 403 || response.status === 404) throw new IdentitySessionError(403, "PRINCIPAL_FORBIDDEN");
  if (!response.ok) throw new IdentitySessionError(503, "IDENTITY_UNAVAILABLE");
  const envelope = await response.json();
  const principal = envelope?.data;
  if (envelope?.success !== true || principal?.principalId !== claims.user_id || principal?.idpSubject !== claims.sub || !["CUSTOMER", "EMPLOYEE"].includes(principal?.principalType)) throw new IdentitySessionError(503, "IDENTITY_INVALID");
  if (principal.provisioningStatus !== "ACTIVE") throw new IdentitySessionError(403, "PRINCIPAL_FORBIDDEN");
  const user: CustomerCoreUser = { principal_id: principal.principalId, principal_type: principal.principalType, email: principal.email, roles: principal.roles, permissions: principal.permissions };
  const realmAccess = claims.realm_access as { roles?: unknown } | undefined;
  const customerRole = Array.isArray(realmAccess?.roles) && realmAccess.roles.includes("CUSTOMER");
  if (!customerRole) {
    if (principal.principalType !== "EMPLOYEE") throw new IdentitySessionError(403, "PRINCIPAL_FORBIDDEN");
    return { ...mapCustomerCoreSession(user)!, sessionExpiresAt: new Date(claims.exp! * 1000).toISOString() };
  }
  const profileResponse = await read("/profile");
  if (profileResponse.status === 401) throw new IdentitySessionError(401, "AUTH_REQUIRED");
  if (profileResponse.status !== 404) {
    if (!profileResponse.ok) throw new IdentitySessionError(profileResponse.status === 403 ? 403 : 503, "CUSTOMER_MAPPING_UNAVAILABLE");
    const profile = await profileResponse.json();
    if (profile.user_id !== principal.principalId || typeof profile.id !== "string" || !/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/u.test(profile.id) || profile.id === "00000000-0000-0000-0000-000000000000") throw new IdentitySessionError(503, "CUSTOMER_MAPPING_INVALID");
    if (profile.status !== "ACTIVE") throw new IdentitySessionError(403, "CUSTOMER_FORBIDDEN");
    Object.assign(user, { customer_id: profile.id, full_name: profile.full_name, phone_number: profile.phone_number, avatar_url: profile.avatar_url });
  }
  const session = mapCustomerCoreSession(user);
  if (!session) throw new IdentitySessionError(503, "IDENTITY_INVALID");
  return { ...session, sessionExpiresAt: new Date(claims.exp! * 1000).toISOString() };
}
