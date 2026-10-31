# M2 — Cart/address models and server quote review

2026-10-31: user authorized implementation of the agreed Mobile subset. G1 full-cart direction is accepted for this MVP slice; partial selection remains deferred. Geography source, history queries and typed tracking remain unresolved. Read-only dev7f22226 is the target; only feature/mobile-ui may change.

## Current slice M2-A

Goal: add a tested MVP purchase foundation using full server cart revision and canonical saved addresses. Existing runtime stays usable until coherent UI/async-operation wiring is ready; never route a new quote into legacy direct checkout.

Dependencies: M1 evidence, full-cart user steering, public OrderCart/OrderItem/OrderQuote/ProfileShippingAddress at7f22226. No approval of backend cleanup semantics or provider readiness inferred from screenshots/Issue38. Untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md belongs to the separate reference publication task and is not staged here.

In scope: isolated MVP typed models, GET cart, array GET saved addresses, strict quote body, quote state with loading/empty/unavailable/error/retry/stale guards. Default VIETQR only for this preparation; COD/voucher/provider gates are not opened. Preserve integer VND, optional line prices, canonical codes, quote snapshot and expiry. Reuse ApiClient auth/error handling.

Out of scope: cart writes/cleanup, production geography/inline address creation, history/tracking, backend/contract edits, dependencies, existing partial-cart runtime removal, async checkout operation/payment, UI route activation or any live financial request. No scrcpy needed for foundation tests.

Expected files: checkout/domain/mvp_purchase_models.dart, mvp_purchase_repository.dart; checkout/data/rest_mvp_purchase_repository.dart; checkout/presentation/mvp_quote_cubit.dart; core/network/api_client.dart for strict array read; focused MVP tests.

## Steps / acceptance

- [x] Models parse MVP integers/canonical codes/revision, retain missing price as unavailable and copy collections immutably.
- [x] Adapter reads array addresses, sends quote only cart_revision/payment_method/shipping_address_id; no selected items/voucher/client prices.
- [x] Quote state reads cart/addresses, blocks empty cart/no valid address; revision is server-owned; changing address invalidates quote; pending duplicate/stale response/dispose safe.
- [x] Expired quote is unusable; retry rereads authority, no local expiry extension/price calculation. Conflicts remain visible.
- [x] Focused12/12, analyze PASS, full156/156; build/native NOT RUN for unactivated foundation. Diff/handoff prepared; checkpoint Mobile only.

## Next slices (refine at entry)

M2-B must analyze coherent route/repository/mock activation, canonical saved-address preview and full-cart UI. Do not delete legacy selection or enable production merely because M2-A parses DTOs. M2 complete only after visible quote-review UX and validation; M2-A is not the whole migration.

## Checkpoint / validation

M2-A foundation implemented. New DTOs/repository/Cubit are **PREPARED / NOT WIRED TO RUNTIME / NATIVE NOT RUN**. No visible Mobile screen behavior changed yet. ApiClient adds a strict array read sharing existing auth/error recovery; existing object methods are unchanged.

Focused12/12 PASS and full156/156 PASS. First analyze reported one private mock type exposed by a public test helper; renamed helper private; final analyze PASS (No issues,17.1s). No test/lint disabled. Build/native NOT RUN: no route/provider activation, dependencies or Android code changed; full widget suite compiles/checks shared transport regression. No live backend requests.

Next exact slice M2-B: inspect quote review screen/route and coherent cohort/mock wiring, enforce saved-address availability and full-cart intent, timers/requote UX and explicit preview-only boundary until M3 operations exist. Never connect quote to legacy direct submit. Before editing UI, read/apply existing design skill/primitives and plan the actual route states. Geography/history/tracking and cleanup semantics remain separately gated.
