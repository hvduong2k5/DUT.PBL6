# Customer Mobile — current handoff

Updated2026-10-31. Goal ACTIVE. Current implementation is M2-A MVP purchase foundation, not full MVP UI. User authorized implementing agreed portions after reference Issue38. Mobile only: feature/mobile-ui, parent c6a1c3b3c29a14a6fb2ae7d57f28ff64a3c76143; new checkpoint identifiable by git log -1. No merge/PR/dev/Web edits.

## Authority / current plans

Read-only target dev7f22226c7b2f4363d3ba376b715feb0e63f69355, public mvp-customer.openapi.yaml and MVP migration/registry. Historical ZIP snapshots unchanged; Web tokens b1ff2918 unchanged. Target is integration-required, not deployed provider proof. Full-cart MVP direction accepted by user; no partial selection or new geography source is approved. Prior M1 evidence remains active/M1_MVP_CONTRACT_GAPS.md.

Master: docs/plans/MASTER_PLAN.md. Current detailed plan: docs/plans/active/M2_MVP_PURCHASE.md. M2-A complete as PREPARED / NOT WIRED TO RUNTIME / NATIVE NOT RUN; M2 overall ACTIVE, M3/M4 NOT STARTED. Legacy runtime and local mocks still usable until coherent integration. Do not claim checkbox/UI migration from this foundation slice.

## Completed source slice

- checkout/domain/mvp_purchase_models.dart: server revision/full items, optional line integer-VND prices (missing is not free), immutable lists, canonical saved address codes/name/version, quote snapshot/totals/source version/expiry. No price calculation or local deadline extension.
- checkout/domain/mvp_purchase_repository.dart and data/rest_mvp_purchase_repository.dart: GET server cart, GET saved addresses array, POST quote with only cart_revision/payment_method VIETQR/shipping_address_id. Reject mismatched revision/method/source ID. No order creation/cart writes/cleanup/voucher/COD/inline address API.
- core/network/api_client.dart: strict JSON object-array read with existing auth/error recovery, no automatic replay.
- checkout/presentation/mvp_quote_cubit.dart: loading/empty/no usable address/editing/pending/ready/error; address change invalidates quote; stale responses ignored; expiry query; retry rereads server authority; duplicate pending tap/dispose safe.
- test/features/checkout/mvp_purchase_test.dart:12tests for DTOs/REST/state/races/expiry/auth transport. All fixtures local; no provider calls.

## Validation

Focused12/12 PASS; full156/156 PASS. Analyze initially reported one public test helper exposing private mock type; helper renamed private, final analyze PASS No issues17.1s. Focused rerun after naming-only fix recorded at checkpoint. No disabled assertions/lints. User-requested debug APK build PASS275.1s on cf7948c, output apps/mobile-app/build/app/outputs/flutter-apk/app-debug.apk. Native install/device test NOT RUN; no UI wiring/dependencies/Android changes. Baseline old144tests do not by themselves establish new DTO/provider compatibility.

## Boundaries / remaining work

Saved-address read only; canonical lookup/source and create/edit UI are unresolved. No backend cleanup assumptions implemented. Cursor input/typed tracking remain open contract gaps; no q/page invention. Provider readiness, authentication integration, operation recovery and payment acceptance still pending in later slices. QA-01 is re-scoped to new payment contract at M4; QA-02 external-media handoff remains unverified.

Untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md is the separate reference artifact published as https://github.com/hvduong2k5/DUT.PBL6/issues/38. Preserve it and do not stage automatically. No Issue/comments modified in this implementation turn.

## Exact next action

Resume AGENTS/master/current M2 plan; verify branch/HEAD/status/remote. Start M2-B by analyzing existing UI primitives/design skill, route and cohort/mock activation. Implement explicit quote-review UX for full-cart + canonical saved address, server totals/expiry/requote and empty/error/back. Never route a new quote into old direct checkout. Keep place-order unavailable until M3 operation integration is coherent, explaining availability in user-friendly terms. No broad scrcpy loop; deterministic widget tests first, one targeted native check only when integrated.
