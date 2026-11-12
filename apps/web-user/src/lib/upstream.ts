import { usesIdentityBff } from "./auth/oidc-config";

/**
 * Upstream topology for the customer storefront (ADR-002).
 *
 * Two different mounts hang off the public API gateway:
 * - the gateway root (…/api/v1) routes to catalog / order / profile and to
 *   identity's public endpoints — products, cart, checkout, orders and
 *   public/customer-registrations live here;
 * - the identity customer-api mount (…/api/v1/customer-api), configured
 *   through CUSTOMER_CORE_UPSTREAM_URL. What it serves depends on the
 *   auth mode: in OIDC mode only identity's own customer surface
 *   (session, profile, addresses); in identity-BFF mode it is also the
 *   translating proxy for the credentialed commerce routes (see
 *   customerCommerceUpstreamUrl below).
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

/**
 * Base for the credentialed commerce routes (cart, checkout, orders,
 * payments, returns). The modes differ in kind, not in prefix: in
 * identity-BFF mode the credential is an opaque session id that only
 * identity's customer BFF can translate into a principal, and the
 * customer-api mount proxies these routes onward — so the commerce
 * base stays CUSTOMER_CORE_UPSTREAM_URL there. In OIDC mode the access
 * token is a self-contained JWT that each service verifies itself, so
 * the calls go to the gateway root like the anonymous routes.
 *
 * The first version of the F-12 fix used the gateway root in both
 * modes and the web-identity fixture caught it: its cross-owner cart
 * check failed because the order service cannot resolve a raw session
 * id. This resolver is the correction.
 */
export function customerCommerceUpstreamUrl(): string {
  if (usesIdentityBff()) {
    const customerCore = clean(process.env.CUSTOMER_CORE_UPSTREAM_URL);
    if (customerCore) return customerCore;
  }
  return gatewayUpstreamUrl();
}
