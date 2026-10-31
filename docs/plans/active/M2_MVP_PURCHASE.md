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
- [x] Focused12/12, analyze PASS, full156/156; debug APK PASS275.1s on cf7948c; native NOT RUN for unactivated foundation. Diff/handoff prepared; checkpoint Mobile only.

## Next slices (refine at entry)

## M2-B entry analysis — 2026-10-31

User explicitly requested continuing beyond build. Add an actual quote-review route and a full-cart quote entry in the existing cart; keep legacy direct transaction path separate until M3 is coherent. AppDependencies optional MVP repository preserves legacy tests/custom dependency bundles. Local factory shares the real in-memory cart with a dedicated mock quote adapter; canonical saved-address fixtures labelled local, not inferred from old name-only address book. Remote adapter remains opt-in/not advertised as provider-ready. No new backend writes or geographic lookup.

Route /checkout/quote is account-gated and preserves login return. Screen uses existing header/CTA/error/empty/theme primitives: full-cart lines, saved-address picker, server quote snapshot/totals, absolute expiry with timer, expired/retry/requote/changed address, no-address/empty/offline and back-to-cart. M2-B has no fake place-order action; availability message explains the pending order step. M3 adds the actual operation flow after its own plan. Validate route from cart, login return, all-lines snapshot, selection invalidation, expiry, errors, narrow/large text, disposal; no arbitrary geography provider or payment integration.

M2-B must analyze coherent route/repository/mock activation, canonical saved-address preview and full-cart UI. Do not delete legacy selection or enable production merely because M2-A parses DTOs. M2 complete only after visible quote-review UX and validation; M2-A is not the whole migration.

## M2-B implementation and checkpoint

- [x] Optional repository/app provider and authenticated /checkout/quote route; login returns to review, unavailable configuration offers return to cart.
- [x] Local shared-cart quote adapter, explicitly labelled address/price/fee fixtures; no backend fallback, order creation or cart deletion.
- [x] Cart entry and full-cart review without selection controls; snapshot/address/totals/earliest expiry and one-tap re-read/requote. Expired totals remain identified as expired until refreshed; no enabled purchase action.
- [x] Empty/error/retry/no-address/pending/back states; timer disposal, 360px large text and login navigation verified. Remote factory remains unconfigured; legacy cart selection/direct checkout deliberately retained pending M3.
- [x] Focused MVP17/17, full regression161/161 PASS; analyze No issues12.0s; diff check PASS. Debug APK PASS38.2s on4389c06,198276012bytes (189.1MiB), build/app/outputs/flutter-apk/app-debug.apk. No native or provider calls; no generated build output committed.

Implementation areas: app dependencies/provider/routes/auth return, CartPage entry, LocalMvpPurchaseRepository, MvpQuotePage, Cubit.refreshQuote, MvpQuote.validUntil and mvp_quote_navigation_test.dart (5 new tests). Existing 12 model/REST/state tests retained. Requote re-reads server authority and discards an older load before issuing a replacement request.

M2 UI slice delivered as a local preview; whole migration remains incomplete. Main cart checkbox removal and default full-cart purchase transition are deferred until M3 async operation can accept the quote coherently. No premature production gate or legacy payment integration. Saved-address create/edit remains blocked on geography decisions rather than inferred from fixture codes.

Source checkpoint4389c063e542fc88bb8790479a548e4c882b4684 pushed and remote hash verified. Exact next action: analyze M3 create/poll/idempotency/ownership/persistence in a new detailed phase plan before implementing its operation flow. Do not mistake quote preview for successful order placement. Current manual local review path: cart → Xem báo giá cả giỏ → login if needed → Lấy báo giá cả giỏ. Device install/native acceptance NOT RUN; automated tests establish local behavior only.

## M2-A historical validation

M2-A foundation implemented. New DTOs/repository/Cubit are **PREPARED / NOT WIRED TO RUNTIME / NATIVE NOT RUN**. No visible Mobile screen behavior changed yet. ApiClient adds a strict array read sharing existing auth/error recovery; existing object methods are unchanged.

Focused12/12 PASS and full156/156 PASS. First analyze reported one private mock type exposed by a public test helper; renamed helper private; final analyze PASS (No issues,17.1s). No test/lint disabled. User-requested debug APK build PASS275.1s on source cf7948c (--no-pub; process-local Kotlin workaround), artifact build/app/outputs/flutter-apk/app-debug.apk. Native install/device test NOT RUN; no route/provider activation, dependencies or Android code changed. No live backend requests.

At the M2-A checkpoint the next slice was M2-B; that work is now completed above. Resume with M3 entry analysis rather than rebuilding M2-B. Never connect quote to legacy direct submit. Geography/history/tracking and cleanup semantics remain separately gated.
