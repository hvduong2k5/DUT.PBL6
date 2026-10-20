# P4 — Customer parity validation and delivery

## Goal and dependencies

Verify implemented P1/P2/P3 against read-only Customer Web and unchanged ZIP on Android. P2 reviews and cart clear implemented (focused reviews 4/4 and cart/purchase 8/8 PASS); native and regression remain open. Production voucher application and returns line-ID context are external contract gaps.

## Scope and steps

1. Audit session expiry, keyboard, large text, back, loading/empty/error and order states.
2. Verify local Android recovery/login, catalog/filter/detail/review, cart/checkout/payment, profile/address/loyalty, history/tracking, support/reply, trace, returns and verified B2B.
3. Run analyze, full test suite and APK build with stable source; distinguish native from automated and production evidence.
4. Reconcile matrix/plans/CURRENT_STATE and Mobile-only checkpoint/push. No merges/PRs or uncontracted endpoints.

## Verified issue and design before modification

Protected API 401 currently becomes a screen error without revalidating global identity. Restore/refresh works only at startup. Add an optional session-recovery callback at transport; RestAuthRepository reuses serialized restore/refresh and reports identity loss through an optional domain observer; AuthCubit clears stale authenticated UI. Exclude auth endpoints to avoid recursion. No automatic write replay: screen retry preserves existing idempotency. Concurrent failures share recovery; explicit auth actions stay serialized. Retain tokens offline. No new provider/dependency/endpoint.

Expected files: core/network/api_client, auth domain/REST/Cubit, focused session tests. Preserve existing auth payloads and 401/403 distinctions.

## Acceptance / validation

Protected 401 rechecks once without auth recursion; valid refresh retains identity, invalid refresh removes actor/capabilities; no duplicate writes; 403/network errors never trigger refresh; concurrency, explicit-auth races and offline token retention tested. Auth regressions pass. Native fixture tests cannot prove backend/email/SMS/payment.

## Status / next

Session recovery and cart confirmation implemented. Auth focused 21/21 PASS, cart/purchase 8/8 PASS, analyze PASS and full 139/139 PASS. APK build running with source held stable. Logistics known codes translated, unknown values preserved. Local created-order snapshots now retain exact SKU.

Product audit slice implemented: gallery preserves all images; certificate number/variant weight/original price retained; explicit SKU choice even for one available variant; food-information unavailable state and manual trace recovery. ZIP provides these product fields but no food fields: no fabricated ingredients or inferred batch/sample certificate association. Focused catalog/reviews/viewport 32/32 PASS; new explicit-selection widget passed, projection assertion corrected to compare Money BigInt. Rerun final full/analyze/build then native. Whole task remains ACTIVE.

Final source validation: analyze PASS; full 141/141 PASS; APK build PASS 35.3s. Install/native next.
