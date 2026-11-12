import { usesIdentityBff } from "./auth/oidc-config";

/**
 * Upstream topology for the customer storefront (ADR-002).
 *
 * Two different mounts hang off the public API gateway:
 * - the gateway root (…/api/v1) routes to catalog / order / profile and to
 *   identity's public endpoints — products, cart, checkout, orders and
 *   public/customer-registrations live here;
 * - the identity customer-api mount (…/api/v1/customer-api), configured
 *   through CUSTOMER_CORE_UPSTREAM_URL, serves only identity's own
 *   customer surface (session, profile, addresses).
 *
 * The commerce handlers (catalog, cart, checkout, orders, payments,
 * returns) build root-relative paths, so they must resolve the gateway
 * root. Reading the customer-api mount instead answers 401 from identity
 * for paths it does not own — that mis-wiring broke registration and the
 * whole shop in the OIDC deployment (findings register F-12).
 *
 * Resolution order:
 * 1. API_GATEWAY_UPSTREAM_URL — an explicit gateway root wins everywhere.
 * 2. Identity-BFF mode: IDENTITY_UPSTREAM_URL already names the gateway
 *    root in that deployment profile (the catalog and registration routes
 *    have always used it there).
 * 3. CUSTOMER_CORE_UPSTREAM_URL with a trailing /customer-api removed —
 *    the customer-api mount always hangs below the same gateway root, and
 *    local development points CUSTOMER_CORE_UPSTREAM_URL straight at the
 *    root-shaped mock, which passes through unchanged.
 * 4. The local mock default the handlers used before.
 */
const DEFAULT_UPSTREAM_URL = "http://127.0.0.1:4010/api/v1";

function clean(value: string | undefined): string | undefined {
  const trimmed = value?.trim().replace(/\/+$/u, "");
  return trimmed || undefined;
}

export function gatewayUpstreamUrl(): string {
  const explicit = clean(process.env.API_GATEWAY_UPSTREAM_URL);
  if (explicit) return explicit;
  if (usesIdentityBff()) {
    const identity = clean(process.env.IDENTITY_UPSTREAM_URL);
    if (identity) return identity;
  }
  const customerCore = clean(process.env.CUSTOMER_CORE_UPSTREAM_URL);
  if (customerCore) return customerCore.replace(/\/customer-api$/u, "");
  return DEFAULT_UPSTREAM_URL;
}
