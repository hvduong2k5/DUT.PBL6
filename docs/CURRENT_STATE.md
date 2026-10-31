# Customer Mobile — current handoff

Updated2026-10-31. Branch feature/mobile-ui. M3 source4e6d76d854a03bab3ce57b1cb722b694d902c373 and M4 source7e00825f5f819fa02d51bac900a56e76a856c420 pushed and remote hashes verified. Delivery checkpoint74ba01a; latest documentation checkpoint identifiable with git log -1. Latest user steering: provide a new login/test it, then propose the next implementation for approval. Follow-up local native QA completed; next auth implementation NOT APPROVED yet. No merge/PR/dev/Web edits.

## Authority / current phase

Read-only dev7f22226c7b2f4363d3ba376b715feb0e63f69355 public MVP integration target; historical ZIP provenance and Webb1ff2918 visual foundations preserved. Full-cart accepted; geography supplier not selected. Master docs/plans/MASTER_PLAN.md, current detailed docs/plans/active/M4_ORDERS_PAYMENT.md. M1 analysis, M2 quote, M3 operations and M4 local order/payment/cancel implementation complete; automated/APK/bounded native evidence below. Whole production migration remains unaccepted; provider activation NOT APPLIED.

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

M3 checkpoint4e6d76d pushed/verified172 tests/analyze. M4 adds12 order/payment/cancel state/transport/storage tests +2 navigation cases (360px, scale1/1.6), and one malformed-quote regression. Initial full run exposed two local legacy route regressions; fixed with a LOCAL-ONLY legacy ID boundary, both focused cases PASS. Final full187/187 PASS46s, analyze No issues14.5s; diff check PASS. No disabled assertions/lints.

Final debug APK PASS94.6s on7e00825 (--no-pub; process-local Kotlin workaround). Artifact apps/mobile-app/build/app/outputs/flutter-apk/app-debug.apk,198276012bytes (189.1MiB), built2026-10-31 11:57:48+07:00. SHA256 C6FE8E23E27AA5DA999EC88F2E984A644CF510C0F3744EB12E40DC7712DD81E0. ZIP kernel inspection confirms mvp_operation_page/mvp_payment_cubit/mvp_orders_page present. Build output ignored, not committed.

Initial standard-APK smoke: ADB install -r Success; cold start com.example.mobile_app/.MainActivity Status:ok,6116ms. LG LMV409N70de1ce3 Android10. Home → product → explicit SKU → cart → quote account gate. Process15246 log:163 app-specific lines, zero crash/Flutter/overflow matches. This initial smoke did not establish post-login native acceptance; follow-up below does so only for the explicit local fixture. No real financial action/device data clear/uninstall/system setting changes.

## Follow-up account/native QA (2026-10-31)

Read current MockAuthRepository: register only sets currentUser; login accepts nonempty input except its negative fixture and maps ordinary accounts to shared CUST-MOCK. No account registry, password verification or durable session exists. Therefore no durable/backend account was created. Synthetic local login details were supplied to the user in chat, never committed. New auth persistence/isolation is a proposed next scope, not silently implemented.

Used an ignored temporary native entrypoint calling existing AuthCubit.register → logout → login, checked synthetic phone identity at each step, then seeded two local cart lines and launched the unchanged OmaApp quote route. Native log marker PBL6_NATIVE_QA_REGISTER_LOGIN_OK_LOCAL_ONLY confirmed success. This tests native registration/login API execution, NOT registration form submission or real server authentication.

Scrcpy verified full local post-login path: two-line quote subtotal157000 + shipping15000 =172000VND → explicit confirmation → one committed operation/order → LOCAL-1 detail → unpaid payment screen (allocated/refunded0, no real QR/beneficiary) → manual status refresh remains unpaid → cancel dialog empty reason disables submit → synthetic reason enables submit → authoritative cancelled detail with payment still unrecorded → history shows cancelled LOCAL-1/172000VND. Process15958:118 app-PID log lines, registration/login marker present, zero severity-E/F Flutter/AndroidRuntime/FATAL/Unhandled/RenderFlex matches. One SDK informational engine-flag notice is not an app error.

QA harness build59.8s/install10.1s; no production source/dependency edits. Afterwards removed credential-bearing temporary file, restored exact standard APK SHA256 above, adb install -r Success, cold launch5880ms Status:ok. Phone now runs standard guest/local app again; ephemeral QA orders/session intentionally do not survive this restore. Backend account/payment/provider acceptance remains NOT VERIFIED. Baseline187 tests/analyze still apply: only temporary setup and documentation changed in this follow-up, no broad suite rerun needed.

## Open external boundaries

Provider/Identity/BFF runtime readiness NOT VERIFIED; no real checkout/payment/refund calls. Canonical geography create/edit source, history cursor inputs and typed tracking remain open contract/team decisions. Care cancellation review requires verified approval and is not activated. Local commands/backend are process-local. New local auth persistence/password verification/unique account IDs is an implementation gap now proposed for user approval. These are explicit limits, not fabricated features or a claim that all production migration criteria passed.

## Exact next action

1. Wait for user approval of the proposed next auth-local scope; latest request explicitly requires approval after testing. Do not implement account persistence/isolation or activate real providers yet.
2. If approved, write the next detailed plan before auth changes: inspect ZIP/MVP auth authority and existing fixtures/callers; persistent local account/session boundaries, real per-account IDs, correct credential validation, duplicate registration, logout/restart and two-account data isolation. Retain explicit B2B/legacy fixtures without turning a phone into production authority; no credentials in Git or client production password store. Analyze dependencies/validation before choosing storage/security details.
3. Verify source7e00825 and latest docs checkpoint on feature/mobile-ui; preserve untracked reference MD. Standard APK restored; no need to redo current native/full-suite checks without a new change. Real BFF/Identity/Order/payment integration and geography/history/tracking/Care gaps remain separately gated.
