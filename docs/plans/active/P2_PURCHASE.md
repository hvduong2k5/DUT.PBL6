# P2 — Customer purchase flow parity

## Goal / why

Match the Web sequence: choose available cart lines → recipient/two-level address → verify voucher → place one order → actual order/payment confirmation. Keep financial state derived from documented backend responses.

## Dependencies / current state

P1 automated checks passed; Android build/device QA pending. Independent P2 planning proceeds while P1 native build runs; do not mark P1 complete yet. Web cart/checkout/payment/promotions and BFF mapping read at `b1ff2918`. Mobile currently checks out/clears entire cart, requires district, sends extra fields and lacks promotions/payment screens.

## Scope / non-goals

In: cart selection/availability/quantity/error states; selected-line checkout; strict ZIP request, two-level locations; voucher list/validation; actual confirmation/QR/expiry; discovery/product differences refined before editing those screens. Out: uncontracted shipping-quote/payment endpoints, fake paid button, backend finance logic, Web edits. User decided to keep ZIP: voucher application to an order waits for contract extension.

## Contracts / decisions

- ZIP POST `/checkout`: only channel, shipping_address, payment_method, items; address excludes district; items only sku_code/quantity. Correct header is `Idempotency-Key` (existing Mobile used `X-Idempotency-Key`). Backend recomputes price.
- Web BFF still sends legacy district, item price and optional voucher_code; these conflict with supplied ZIP. Reuse UI intent, not those unsupported payload fields.
- GET `/locations/checkout`: version + provinces with codes/names and wards; no district.
- GET `/promotions/vouchers`: available vouchers + min/max/expiry; POST `/promotions/validate`: voucher_code + subtotal_amount → is_valid/discount_amount/message. No speculative production discount application.
- Checkout result provides authoritative totals, vietqr_url, payment_expires_at. Order read endpoint supplies status. Do not invent Payment API paths absent from ZIP.

## Steps / files

- [x] Extend CartState with selected available IDs and selected Cart; preserve selection after mutations; prevent concurrent duplicate mutation and surface recoverable errors.
- [x] Cart line selector/all/clear, real available product thumbnails/fallback, quantity bounds, unavailable state and selected subtotal; mobile-friendly layout.
- [x] Checkout uses selected Cart snapshot, preserves unselected lines after success, two-level locations and explicit missing-selection state.
- [x] Correct REST checkout payload/header; maintain idempotent retry per identical payload and new key after edits; map field/business errors.
- [x] Add promotions domain/REST/mock/Cubit/list/validation UI, reachable from drawer/account and Checkout; stale validation discarded when code/subtotal changes.
- [x] Add confirmation/payment presentation using backend fields; expiry and retry/read status; no automatic paid claim.
- [ ] Compare Home/Catalog/Product/reviews, implement remaining gaps in a focused follow-on slice.
- [x] Catalog supported search/pageSize/pagination with loaded-scope category/price/OCOP/stock facets, sorting and filter sheet; stale/no-progress/next-page errors; unknown Inventory and Home search/category continuity. Unsupported global advanced parameters remain a contract limit.
- [x] Reviews: public aggregates/list/paging, owned delivered order and exact SKU selection, text submission, server-only verified badge, capability/pending/error/idempotent retry.
- [ ] Cart clear confirmation; product-detail final parity audit/native discovery QA.
- [ ] Tests, analysis, full regression, Android build/scrcpy QA and milestone checkpoint.

## UI / security / risks

Cover empty/no selected/all/partial/unavailable cart; pending quantity/removal; API error+retry without losing selected cart; location load/empty/error/dependent ward reset; coupon empty/expired/invalid/pending/error; double submit; changed order totals and pending/expired QR. Mobile safe areas, keyboard, narrow/large text and back navigation are required.

Never clear unselected lines, trust client price or show a preview discount as a final order total. If order creation succeeded but cart cleanup fails, preserve confirmation and report cleanup separately; never retry creating a second order for cleanup failure.

## Validation / DoD

Unit tests: selection/reconciliation/availability, retry/idempotency, payload exact allowed keys, location dependency, voucher stale-response handling, partial cart consumption. Widgets: cart→checkout, voucher invalid/valid, location, empty/no-selection and payment states. Run analyze/tests/build and native scrcpy. DONE only when matrix rows and evidence updated; unresolved API limits explicit.

## Handoff

Status: planned, source implementation not started. Next: finish P1 native build/QA, then implement cart selection and strict checkout payload first. User confirmed no contract extension in this task; mark actual Checkout voucher application pending.

2026-10-20 current status: purchase core implemented, latest overall full suite 124/124 PASS at B2B checkpoint. Current task is discovery/reviews/history pagination, not the original cart baseline. Preserve completed transaction contracts.

## Current discovery task and API boundary

Reviews milestone: analyze PASS and focused 4/4 PASS (strict REST body/server aggregates, purchase/duplicate/replay checks, stale order selection, revoked capability and retry keys, narrow-phone form→verified public list). Added explicit local OM-8860 delivered canonical SKU fixture; existing returns fixtures preserved. No product_id in POST, invented eligibility API or fake media upload. Native reviews pending. Exact next: cart clear confirmation, then P4 full/native validation.

Web Catalog/reviews and supplied product/list/search/order/review stable schemas read. Add a small shared typed page result and paged repository methods while retaining existing convenience methods/callers. Product/order query uses q/status/page/pageSize only; response metadata uses page_size. Never send legacy category_id/page_size/minPrice/weights/sort fields absent from ZIP.

Catalog server search/pagination is authoritative. Category-name/price/OCOP/confirmed-stock facets and price sorting may use only loaded results, with visible loaded/total scope until every page is loaded; never call them complete global matches prematurely or silently fetch an unbounded catalog. Weight/product-type fields are absent from stable list projection, so do not infer them from names. Preserve loading/empty/error/retry and next-page errors without discarding loaded cards, ignore stale results, deduplicate IDs and detect no-progress pages. Add a mobile filter sheet reusing tokens and keyboard-safe scrolling. Unknown Inventory is updating, not sold out, and detail entry remains available.

Order history adds server q/status/page/pageSize, read-only snapshot cards and explicit load more. No unsupported year query; narrow order-code local search does not imply product-name index absent from contract. Preserve precise raw status labels.

Review ZIP POST requires order_id/sku_code/rating/comment/media_urls, not Web's product_id input. Use an owned delivered order and exact SKU from its snapshot/current product variants; server determines verified purchase. No guessed order ID/eligibility endpoint or fake verified badge. Review list uses server average/count, pending/empty/error and public media/seller replies; strict retry idempotency, capability and validation. Read relevant source before editing; cart clear confirmation is still open.

Catalog checkpoint: analyze PASS, catalog focused 8/8 PASS; earlier catalog/purchase/viewport regression 26/26 PASS before final paging-sheet tests. No fake global filters. New PageResult shared primitive ready for history/review paging. Exact next: paged history/search/status then owned-order SKU review flow and clear-cart confirmation.

First slice evidence: full Flutter suite 80/80 PASS; analyze PASS before final product padding/lifecycle edits. P2 native QA pending. User confirmed actual Checkout voucher application waits for contract extension; ZIP unchanged. P1 native build/install and recovery validation/sent/reset keyboard/expired states observed via scrcpy.

Payment/order slice: invalid IDs no longer create fake confirmations, native QR/expiry/status view added. Populated purchase widget test passes and unselected lines remain. Native product/SKU/add/cart/checkout visuals observed. Full native order/payment workflow remains P4 work. Quantity limit verified as 20 from Web source.
