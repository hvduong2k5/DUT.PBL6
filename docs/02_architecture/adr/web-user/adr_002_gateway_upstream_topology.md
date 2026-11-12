# ADR-002: Gateway upstream topology for the customer commerce routes

- Status: Accepted, 2026-11-12. Extends ADR-001 (OIDC customer auth).
- Evidence: deployment findings register F-12 (`docs/06_operations/deployment-findings-2026-11.md`).

## Context

Two different mounts hang off the public API gateway:

- the **gateway root** (`…/api/v1`) routes to catalog / order / profile and
  to identity's public endpoints — `products`, `cart`, `checkout`,
  `orders`, `public/customer-registrations` live here;
- the **identity customer-api mount** (`…/api/v1/customer-api`), configured
  through `CUSTOMER_CORE_UPSTREAM_URL`, serves only identity's own
  customer surface (session, profile, addresses, capability checks).

The commerce handlers (catalog, cart, checkout, orders, payments, returns)
and the registration helper build root-relative paths, but read their base
from `CUSTOMER_CORE_UPSTREAM_URL`. In local development that variable
points at the root-shaped mock, so the wiring looked right. In the OIDC
deployment it points at the customer-api mount, so every commerce call
landed on identity paths that do not exist there:

- registration posted to `…/customer-api/public/customer-registrations`
  and identity answered 401 — customers could not register;
- the catalog route fetched `…/customer-api/products/…` — product pages
  failed with `CATALOG_UPSTREAM_ERROR`;
- cart / checkout / orders / payments / returns were wired the same way.

## Decision

1. A single resolver, `gatewayUpstreamUrl()` in `src/lib/upstream.ts`,
   owns the gateway-root base for every root-relative customer route.
   Resolution order:
   1. `API_GATEWAY_UPSTREAM_URL` (new, optional) — an explicit gateway
      root wins everywhere;
   2. in identity-BFF mode, `IDENTITY_UPSTREAM_URL` — it already names
      the gateway root in that deployment profile, and the catalog and
      registration routes have always used it there;
   3. `CUSTOMER_CORE_UPSTREAM_URL` with a trailing `/customer-api`
      removed — the mount always hangs below the same gateway root, and
      the local root-shaped mock passes through unchanged;
   4. the local mock default (`http://127.0.0.1:4010/api/v1`).
2. The catalog route, the registration helper, and the cart / checkout /
   orders / payments / returns handlers build their upstream URLs from
   `gatewayUpstreamUrl()`. Identity's own customer surface (customer,
   customer-core, session and capability checks) keeps using
   `CUSTOMER_CORE_UPSTREAM_URL` unchanged.
3. Cart, checkout and orders join the ADR-001 credential boundary: their
   exported handlers resolve (and when needed refresh) the customer
   credential through `withCustomerAuth()` and forward that credential,
   exactly like payments and returns already did. Before this change they
   read the raw cookie token with no renewal path, so an OIDC session
   older than the 900-second access token failed commerce calls even
   where the upstream had been reachable.

## Consequences

- The deployed OIDC shape works with no configuration change (rule 3
  derives the root from the configured mount); setting
  `API_GATEWAY_UPSTREAM_URL` makes the topology explicit and is the
  recommended production configuration.
- The identity-BFF deployment profile keeps its existing behaviour
  (rule 2); the local mock topology is unchanged (rules 3–4).
- Support and B2B keep using `CUSTOMER_EXTENSIONS_UPSTREAM_URL`: that is
  a separate backend surface, not part of this topology.
- The employee/admin handlers are untouched; their upstream contract
  (`IDENTITY_UPSTREAM_URL`, `ADMIN_API_UPSTREAM_URL`) is separate, as
  recorded in ADR-001.
