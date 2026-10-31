# Customer Mobile — current handoff

Updated 2026-10-31. Migration IN PROGRESS. Current slice: M2-B visible full-cart quote review. Branch feature/mobile-ui; checkpoint parent 0574916d4864cb8cf1575cacbe4853f8d1044c99. Identify the new checkpoint with git log -1; no merge/PR/dev/Web edits.

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

Focused MVP 17/17 PASS. Full regression 161/161 PASS (~30s); flutter analyze --no-pub PASS (No issues,12.0s); git diff --check PASS. No tests/lints disabled. New debug APK build PENDING at this source checkpoint; exact next action below. Previous APK PASS275.1s was built on cf7948c and does not contain the new quote screen. Device install/native/provider acceptance NOT RUN for M2-B.

## Decisions and remaining work

M2-B is a labelled local quote preview. Keep new quote/revision away from legacy direct checkout; preserve that flow until M3 has async operation identity, retry keys and recovery. Entire cart is reviewed regardless of legacy cart checkboxes. Saved addresses are read-only; create/edit/geography source unresolved. No claimed cart-cleanup semantics, production payment readiness, cursor search or invented shipment timeline. Backend/provider compatibility still needs integration verification.

Untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md is the separate reference artifact published at https://github.com/hvduong2k5/DUT.PBL6/issues/38. Preserve and do not stage automatically. No Issue/comment changes in this slice.

## Exact next action

1. Build current M2-B source from apps/mobile-app using flutter build apk --debug --no-pub with process-local GRADLE_OPTS Kotlin workaround; record outcome/artifact and push a documentation checkpoint on feature/mobile-ui only.
2. At M3 entry, inspect pinned public CheckoutOperation/create/poll/idempotency/error contracts and existing auth/storage ownership. Write its detailed plan before edits. Implement pending/unknown outcome recovery and durable retry identity; then migrate the main cart CTA to full-cart purchase and remove selection only for the new flow. Do not connect to legacy direct order/payment DTOs.
3. Native testing: one targeted integrated check after meaningful operation wiring; avoid broad computer-use loops. M4 separately migrates payment/order authority. Geography/history/tracking gaps remain explicit.
