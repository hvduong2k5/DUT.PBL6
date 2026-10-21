# P2 — Customer purchase flow parity

## Goal / why

Match the Web sequence: choose available cart lines → recipient/two-level address → verify voucher → place one order → actual order/payment confirmation. Keep financial state derived from documented backend responses.

## Dependencies / current state

P1 foundation/recovery implemented and locally verified. Web reference remains b1ff2918. Cart selection, two-level strict checkout, promotions/payment, discovery/product/reviews are implemented; remaining targeted payment verification is explicit below. Preserve completed transaction contracts.

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
- [x] Compare Home/Catalog/Product/reviews and implement the mapped differences below.
- [x] Catalog supported search/pageSize/pagination with loaded-scope category/price/OCOP/stock facets, sorting and filter sheet; stale/no-progress/next-page errors; unknown Inventory and Home search/category continuity. Unsupported global advanced parameters remain a contract limit.
- [x] Reviews: public aggregates/list/paging, owned delivered order and exact SKU selection, text submission, server-only verified badge, capability/pending/error/idempotent retry.
- [x] Cart clear confirmation matching Web; cancel retains lines, confirm clears through existing repository.
- [x] Product detail final source audit: explicit SKU choice, gallery/certificate/weight/original price, honest unavailable food information and trace entry. No synthetic food fields.
- [x] Recorded native discovery/filter/product/reviews main flows and keyboard fixes; current full suite/analyze/build PASS.
- [x] Tests, analysis, full regression, Android build, main-flow Android QA and Mobile checkpoint.
- [ ] QA-01: payment Check status result/loading/error/retry/paid/cancelled transitions lack direct widget/native verification; expired-QR widget alone does not cover this action.

## UI / security / risks

Cover empty/no selected/all/partial/unavailable cart; pending quantity/removal; API error+retry without losing selected cart; location load/empty/error/dependent ward reset; coupon empty/expired/invalid/pending/error; double submit; changed order totals and pending/expired QR. Mobile safe areas, keyboard, narrow/large text and back navigation are required.

Never clear unselected lines, trust client price or show a preview discount as a final order total. If order creation succeeded but cart cleanup fails, preserve confirmation and report cleanup separately; never retry creating a second order for cleanup failure.

## Validation / DoD

Unit tests: selection/reconciliation/availability, retry/idempotency, payload exact allowed keys, location dependency, voucher stale-response handling, partial cart consumption. Widgets: cart→checkout, voucher invalid/valid, location, empty/no-selection and payment states. Run analyze/tests/build and native scrcpy. DONE only when matrix rows and evidence updated; unresolved API limits explicit.

## Acceptance reconciliation — 2026-10-21

Status: IMPLEMENTED / VERIFIED LOCAL MAIN FLOWS; targeted payment QA-01 remains OPEN. No additional purchase source change made in this audit.

- Web8page group has Mobile counterparts: Home, Catalog, Product, Cart, Checkout, Confirmation, Payment, Promotions. Review lives in Product. Router mapping checked in app_router.dart/routes.dart; route coverage does not imply every state verified.
- REST checkout source matches supplied allowed keys/header, excludes district/client item prices/voucher_code. Tests: rest_checkout_repository_test.dart, checkout_cubit_test.dart, cart_selection_test.dart, purchase_navigation_test.dart, promotions_test.dart, cart_clear_navigation_test.dart. Catalog/product/review tests cover supported paging/loaded-scope filters, explicit SKU/projections and owned-order/SKU review submission.
- Recorded physical main flows: selected SKU/cart/clear-cancel, two-level checkout/COD unpaid detail, saved address/VietQR pending/missing QR, discovery filters/search recovery and review submission/caret. Current source e86f82b full144/144/analyze/debug APK PASS, installed update without data wipe.
- Audit found one PaymentPage widget test: expired QR hidden/no fake paid. No test presses Kiểm tra trạng thái. Handler calls OrdersRepository.order and exposes loading/error/response states, but source inspection is not interaction proof. Prior physical click has no newly confirmed result. Keep QA-01 open, prefer deterministic widget cases first; no reason for broad computer-use repetition.
- Checkout voucher application remains blocked by ZIP by user's explicit decision. Real QR/payment/provider status and payment reissue remain unverified/unsupported. No fabricated QR or final discount. Advanced global catalog filters remain a documented loaded-scope adaptation, not unsupported server requests.

Next P2 action: QA-01 focused tests for returned pending/paid/cancelled, network error/retry and duplicate pending taps. Only fix a reproduced defect. Actual provider QR acceptance requires backend evidence and cannot be inferred from fixtures.
