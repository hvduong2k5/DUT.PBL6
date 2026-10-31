# Customer Mobile — current handoff

Updated2026-10-31. Branch feature/mobile-ui, M4 source checkpoint parent4e6d76d854a03bab3ce57b1cb722b694d902c373. New source commit identifiable with git log -1. User explicitly requested autonomous continuation; do not pause at implementation checkpoints. No merge/PR/dev/Web edits.

## Authority / current phase

Read-only dev7f22226c7b2f4363d3ba376b715feb0e63f69355 public MVP integration target; historical ZIP provenance and Webb1ff2918 visual foundations preserved. Full-cart accepted; geography supplier not selected. Master docs/plans/MASTER_PLAN.md, current detailed docs/plans/active/M4_ORDERS_PAYMENT.md. M1 analysis, M2 quote, M3 operations and M4 local order/payment/cancel implementation complete automatically; APK/native evidence pending below. Production/provider activation remains NOT APPLIED.

## Completed behavior

- New local main cart buys entire cart, no checkboxes. Quote review uses saved canonical fixture address, returned totals/snapshot/expiry and explicit confirmation. New quote never enters legacy direct checkout.
- App-scoped operation keeps exact owner/backend/key/body; save before POST, recover unknown result with same identity, preserve PublicError operation_id, reject duplicate/stale/account responses. Foreground GET polling bounded; order identity only after SUCCEEDED. Success reloads cart, never clears/consumes it on client.
- Placed MVP fixture order is recorded into new history/detail/payment routes. First-page list sends no invented q/status/cursor/page parameters; nonempty next_cursor shows incomplete history. Detail uses server totals/snapshot/actions/version and independent lifecycle/payment/cancellation/compensation. No synthetic shipping timeline.
- Payment reads status/instructions separately; immutable server beneficiary/code/HTTPS QR only when allowed PAY and current can_pay admit them. Monotonic elapsed + server time/round-trip allowance enforces >30s margin. Refresh/error/verification/delay/finalization/expiry/cancel hides instructions; next_poll_after_seconds respected; no client SePay calls or pretend-paid button. Local fixture intentionally has no real receiver/QR/payment.
- Unpaid cancellation persists key/reason/original If-Match version before POST, handles200/202 as cancellation DTO, preserves unknown command on re-entry, blocks altered reason/new key, then refreshes actual Order. Cancellation never implies refund. Paid review endpoint remains feature-gated; UI offers support.
- Default remote factory remains historical ZIP-based, without activating MVP providers. REST/secure stores for new flows are prepared and tested. Local MVP backend/commands are process-local; secure-store recreation tests are separate from local app-kill recovery.
- Legacy regression cohort via localMvpCheckout:false retained. Existing local guest/B2B non-UUID fixture IDs retain their own legacy ownership/grant checks; all real/remote MVP requests stay on the new parser with no legacy downgrade. New MVP history is separate from historical sample orders.

## Modified areas

App dependency/provider/router, checkout onPlaced/operation/quote guards and new payment Cubit/page; orders MVP models/repository/REST/local/cancellation storage/history/detail Cubits/pages; focused MVP fixtures/tests and explicit legacy cohort fixtures. New invalid-quote guard prevents malformed ID escaping through UI. No dependencies/Android native files or historical contracts changed. Separate untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md belongs to Issue38; preserve, never stage automatically.

## Validation / current evidence

M3 checkpoint4e6d76d pushed/verified172 tests/analyze. M4 adds12 order/payment/cancel state/transport/storage tests +2 navigation cases (360px, scale1/1.6), and one malformed-quote regression. Initial full run exposed two local legacy route regressions; fixed with a LOCAL-ONLY legacy ID boundary, both focused cases PASS. Final full187/187 PASS46s, analyze No issues14.5s; diff check PASS. No disabled assertions/lints. Current M3+M4 APK/native NOT RUN yet; previous M2-B APK4389c06 is outdated. Device LMV409N70de1ce3 connected via ADB; final build/install and one bounded scrcpy smoke remain.

## Open external boundaries

Provider/Identity/BFF runtime readiness NOT VERIFIED; no real checkout/payment/refund calls. Canonical geography create/edit source, history cursor inputs and typed tracking remain open contract/team decisions. Care cancellation review requires verified approval and is not activated. Local commands/backend are process-local. These are explicit external limits, not fabricated features or a claim that all production migration criteria passed.

## Exact next action

1. Checkpoint/push current tested M4 source on feature/mobile-ui only. Build flutter build apk --debug --no-pub using process-local GRADLE_OPTS workaround; inspect artifact/version and install with adb install -r (do not uninstall/clear data).
2. Launch com.example.mobile_app; inspect only its scrcpy surface using computer-use skill, one bounded launch/cart check and app-specific crash output. No broad computer-use loop, credentials, provider operations or real financial action. Widget tests already exercise complete new flow; native smoke is separate evidence.
3. Record source commit, APK outcome/artifact and exact native scope in this file/active plan/master, push and verify final documentation checkpoint. Remaining work requires external contracts/readiness above; do not automatically activate remote providers or merge.
