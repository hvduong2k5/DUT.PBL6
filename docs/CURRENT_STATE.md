# Customer Mobile — current handoff

Updated 2026-10-31. Migration IN PROGRESS. Current slice: M2-B visible full-cart quote review. Branch feature/mobile-ui; source checkpoint 4389c063e542fc88bb8790479a548e4c882b4684 pushed and remote hash verified. Latest commit adds only build/handoff evidence; identify it with git log -1. No merge/PR/dev/Web edits.

## Authority and plans

Read-only target dev7f22226c7b2f4363d3ba376b715feb0e63f69355 public MVP Customer contract. Historical ZIP snapshots unchanged; Web visual baseline b1ff2918 unchanged. These are integration targets, not deployed provider proof. Full-cart MVP accepted; no geography supplier or partial checkout approved.

Master: docs/plans/MASTER_PLAN.md. Current detailed plan: docs/plans/active/M2_MVP_PURCHASE.md. M2-A foundation complete; M2-B quote UI implemented and verified automatically in local mode. Main checkout migration, M3 operations and M4 payment remain unfinished. Do not describe the app as fully migrated.

## Completed work and modified areas

- M2-A retained: strict MVP models, REST cart/address-array/quote adapter, quote Cubit, ApiClient array reader and 12 contract/state tests.
- AppDependencies/app/router/routes/auth return: optional MVP purchase repository, /checkout/quote gated by account identity with login return. Local factory shares the actual in-memory cart with LocalMvpPurchaseRepository. Remote factory deliberately does not activate the new purchase flow.
- CartPage: extra local entry “Xem báo giá cả giỏ”. Existing selected-lines checkout remains separate until M3 can replace it coherently; checkbox removal/main CTA migration is NOT DONE.
- MvpQuotePage: all cart lines, saved canonical address, returned snapshot/totals, earliest of three expiries, expired notice and one-tap requote that reloads cart/addresses first. Names label snapshot geography only when codes match. No place-order action, fake order, QR or payment result.
- Local adapter: explicit sample address and quote/fee fixture; reads shared cart, checks revision/stock/auth and never clears it. No remote-error fallback or provider calls.
- New 5 tests: shared full-cart/revision/logout guards; login return; 360px + text scale 1.6 + expiry/requote; error/address recovery/pending controls; empty cart. Native visual acceptance remains separate.

## Validation

Focused MVP 17/17 PASS. Full regression 161/161 PASS (~30s); flutter analyze --no-pub PASS (No issues,12.0s); git diff --check PASS. No tests/lints disabled. New debug APK PASS38.2s on source4389c06, using process-local Kotlin workaround and --no-pub. Artifact apps/mobile-app/build/app/outputs/flutter-apk/app-debug.apk,198276012bytes (189.1MiB), contains the new local quote screen. Device install/native/provider acceptance NOT RUN for M2-B; build output is ignored and not committed.

## Decisions and remaining work

M2-B is a labelled local quote preview. Keep new quote/revision away from legacy direct checkout; preserve that flow until M3 has async operation identity, retry keys and recovery. Entire cart is reviewed regardless of legacy cart checkboxes. Saved addresses are read-only; create/edit/geography source unresolved. No claimed cart-cleanup semantics, production payment readiness, cursor search or invented shipment timeline. Backend/provider compatibility still needs integration verification.

Untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md is the separate reference artifact published at https://github.com/hvduong2k5/DUT.PBL6/issues/38. Preserve and do not stage automatically. No Issue/comment changes in this slice.

## Exact next action

1. Resume AGENTS/master/current handoff, verify branch/HEAD/status/remote; start M3 analysis and a detailed phase plan before source edits. M2-B local review is complete; do not repeat its build/test suite unless source changes.
2. Read pinned CheckoutOperation/create/poll/idempotency/error contracts and existing auth/storage ownership. Initial evidence: public spec lines229–449 and2411–2505; POST202 means accepted operation with null order_id, replay201 can return SUCCEEDED with committed order_id, transport uncertainty requires same-key/body replay. Existing uuid and flutter_secure_storage can be reused without dependency upgrades. Plan identity-scoped durable recovery before enabling create. Then migrate the main cart CTA to full-cart purchase and remove selection only for the new flow. Do not connect to legacy direct order/payment DTOs.
3. Native testing: one targeted integrated check after meaningful operation wiring; avoid broad computer-use loops. M4 separately migrates payment/order authority. Geography/history/tracking gaps remain explicit. To inspect current local UI manually: add cart items → Xem báo giá cả giỏ → local login if prompted → Lấy báo giá cả giỏ; this action creates no order.
