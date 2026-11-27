# Catalogue-change runbook (administrative provinces/wards)

The administrative catalogue is owned by **profile-service** (seeded from its
migrations) and mirrored into **order-service** as the embedded canonical
dataset (`services/order-service/internal/simulator/administrative_units.json`).
The two copies must always describe the same set:

- the Dev Gate enforces *candidate consistency* — the contract test
  `TestAdministrativeDatasetMatchesProfileSeed` fails the build if the
  embedded dataset diverges from profile's full-replace migration;
- this runbook covers *runtime consistency* — deploying a catalogue change
  without checkout taking orders against a half-updated catalogue.

## When a maintenance window is required

Any catalogue change that **removes, renames or re-parents** a code is
breaking and MUST go through the window below. A purely additive change
(new codes only, no meaning change to existing codes) may skip the window
only when all four conditions hold (address-seam design, GPT review
2026-11-25): downstream accepts the new codes before profile publishes
them; the business accepts orders using codes profile has not published
yet; a failed profile deploy leaves orders placed with new codes safe to
fulfil or recover; no existing code changes meaning. When in doubt, use
the window.

## The barrier

`catalogue_maintenance` (the simulator's provider database; created by
the simulator's bootstrap schema, applied at every simulator start) is a
persistent flag. While it is on, the checkout provider paths (address validate,
terms/quote, revalidate) answer **503 CATALOGUE_MAINTENANCE**. The flag
survives restarts, redeploys and rollbacks, and nothing clears it
automatically — a failed deploy leaves checkout closed, never half-open.
The check fails closed: if the flag cannot be read, checkout is treated
as under maintenance.

Internal endpoints (simulator listener, `X-Internal-Token` required):

- `GET /catalogue/status` → `{dataset_version, dataset_checksum, maintenance}`
- `POST /catalogue/maintenance` with `{"enabled": true|false, "reason": "..."}`

Controlled smoke access while the barrier is on: requests to the provider
paths carrying the internal token in the `X-Catalogue-Bypass` header are
let through. Storefront traffic never carries that header.

## Window procedure

1. Confirm the candidate SHA passed the Dev Gate (contract test green).
2. **Enable the barrier** (`POST /catalogue/maintenance`, `enabled: true`,
   reason = the change ticket). Verify with `GET /catalogue/status`
   (`maintenance: true`) and one public guest quote — it must answer
   503 CATALOGUE_MAINTENANCE (`scripts/catalogue_smoke.py
   --expect-maintenance` checks exactly this).
3. Let in-flight checkouts drain. Quotes live at most 15 minutes; after
   the barrier is on, no new quotes are issued, and accepts are refused,
   so any checkout still in flight fails safe and the customer re-quotes
   after the window.
4. Deploy profile and order from `cd-candidate` through the normal CD
   flow.
5. **While still locked**, verify runtime consistency:
   - `scripts/catalogue_smoke.py --internal-base <simulator> --token <internal>`
     — the checksum order-service actually loaded equals the repo dataset;
   - the public locations checksum (step 1 of the public smoke) already
     matches the repo dataset.
6. **Disable the barrier** (`enabled: false`). Run the full public smoke:
   `scripts/catalogue_smoke.py` — served-catalogue checksum equals the
   repo dataset; a valid pair quotes 200; a retired pair and a cross
   pair answer 422 INVALID_ADMINISTRATIVE_UNIT.
7. Announce the window closed.

If anything fails mid-window: leave the barrier **on**, fix forward or
roll the deploy back, and only disable the barrier after step 5 passes.
Never disable the barrier "just to test".

## Data warnings

- A full-replace catalogue migration (e.g. 000005) deletes saved
  addresses and street→ward mappings that reference retired codes —
  their province/ward no longer exists. Orders are not touched: they
  keep the address name snapshots taken at quote time.
- Rolling the migration back (down file) restores the previous code set
  but cannot restore deleted saved addresses.

## Dataset updates

The canonical dataset is generated, never hand-edited: regenerate the
profile full-replace migration and `administrative_units.json` from the
same source (National Statistics Office post-2025 divisions, via the
`vietnam-provinces` data package) in one commit, bump the dataset
`version`, and let the contract test prove both copies identical.
