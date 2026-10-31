# M4 — Committed orders, payment authority and cancellation

Follow-up2026-10-31 COMPLETED: user requested a new login and post-login testing, then a proposal for approval before further implementation. APK is mock: no persisted account/credentials, password authentication or unique ordinary user ID. Did not create/claim a durable/backend account. Ignored temporary QA entrypoint executed AuthCubit register/logout/login with a synthetic identity on device; same session reviewed two-line172000VND quote → committed operation → detail → unpaid payment/no real QR → status refresh → reasoned cancellation → cancelled history. Process15958 log118 lines, registration/login marker present, no actual Flutter/AndroidRuntime/FATAL/overflow exceptions. Registration form submission and backend auth NOT VERIFIED. No production source/dependency/backend changes. Restored/reinstalled standard APK exact SHA256, cold start5880ms PASS; removed temporary credential-bearing file. Never commit login/password. Next local account persistence/ID isolation/credential validation proposal is WAITING USER APPROVAL.

2026-10-31. LARGE, continuation explicitly authorized. M3 source4e6d76d pushed/verified,172 tests/analyze PASS. M2 quote and M3 committed identity are dependencies; never send those IDs to legacy DTO parsers. Branch feature/mobile-ui only; no provider activation/dev/Web/contract edits.

## Goal and contract evidence

Complete the new local Customer purchase path through order detail/history, read-only payment status/instructions and versioned unpaid cancellation. Prepare strict REST boundaries for later integration. Public dev7f22226 spec: orders lines443–603 and OrderOrder2640–2798; cancel604–710/2799–2841; payment1055–1238/2923–3140. Independent lifecycle/payment/compensation and server allowed_actions govern UI.

Order list supports items/next_cursor but public input cursor/limit/status/q is missing (M1 G3). Fetch first page without invented parameters, no global search/year/legacy page totals; show incomplete-list state if next_cursor exists. Do not enable uncontracted pagination. Tracking remains an untyped object: show only actual available identifiers/status, no fabricated timeline/carrier/media. Paid cancellation endpoint is verified-care-approval gated: offer support instead of activating that mutation.

Payment instructions GET is read-only, persisted SEPAY/VIETQR intent, can_pay:true, immutable amount/receiver/code/QR/expiry, strictly >30s before expiry. Status GET has can_pay/display_state/server_time/version/next_poll_after_seconds and independent expected/allocated/refunded amounts. VERIFYING/VERIFICATION_DELAYED/FINALIZING/reconciliation/expired/cancelled must not expose fresh transfer instructions or claim paid. No SePay calls, webhook inference, fake paid controls or client-generated bank/QR. Local fixture exposes unpaid status and explicitly unavailable real instructions.

Cancel POST only reason+active:true, Idempotency-Key and decimal If-Match expected version.200/202 response is OrderCancellation, not an Order or boolean; cancellation and compensation separate. Persist reason/version/key before POST and preserve after uncertainty across page recreation. Never call cancelled/refunded on timeout/acceptance. Refresh authoritative Order after terminal decision;409 does not justify automatic re-key or changing payload. Local fixture may model PENDING→SUCCEEDED for unpaid cancellation, never money transfer.

## Architecture / expected files

New orders/domain/mvp_order_models.dart and mvp_orders_repository.dart; data/rest_mvp_orders_repository.dart, local_mvp_orders_repository.dart and mvp_cancellation_store.dart. New focused Cubits/pages for first-page history, detail/cancel, payment. Reuse checkout UUID/timestamp validation, Money, shared theme/header/buttons/error/empty and existing secure storage/uuid. Optional MvpOrdersRuntime in app dependencies/provider; local checkout onPlaced records a new MVP order snapshot. Default remote ZIP cohort remains untouched. Existing legacy order navigation tests explicitly select their cohort, assertions retained.

State ownership: pages are account-gated and keyed to principal/order; Cubits reject stale/disposed/account responses. Stored cancel command scoped by principal/backend/order, no tokens/contact/financial credentials. Failed storage blocks mutation. Cancellation result kept separate from Order snapshot; pending/unknown command blocks a new reason/key. Reads can recover safely; only explicit retry replays an uncertain command.

Payment time: use server_time plus monotonic elapsed/round-trip allowance, never local wall clock to extend payment window. Hide instructions during refresh/error and when margin <=30s. Match order/intent/expected amount/expiry across status and instructions; reject regressive versions/identities. Bounded foreground polling follows next_poll_after_seconds;0 stops. Back/account transition stops timers. Order allowed PAY and payment can_pay must both admit instructions.

## UX and steps

- [x] Strict integer VND/status/Order/action/payment/cancellation models; missing snapshot/price means unavailable, never zero/free.
- [x] Exact REST reads and versioned cancel command; first-page-only list and honest tracking limits.
- [x] Local placed-order repository tied to real M3 fixture identity; no real bank/QR/paid data.
- [x] Recoverable cancellation store/state and order refresh; readonly payment state/time/polling guards.
- [x] Customer routes use new history/detail/payment for this cohort; operation success links to actual Order. No legacy return/care/payment mutation for MVP IDs.
- [x] Empty/loading/error/retry/pending/expired/verification/reconciliation/manual-review/cancel states, back/login, large text and narrow screens.
- [x] Focused transport/state/widget tests, full regression/analyze, final APK; one bounded native check if available.
- [x] Update handoff/master/decisions/evidence, verify Mobile-only source push; final evidence checkpoint follows, external contract/provider gaps clearly separated.

## Acceptance and checkpoint

User can locally add full cart → quote → confirm → observe one operation/order → view new history/detail → read payment availability → request unpaid cancellation and refresh actual result. No legacy direct order creation, client cart clear, fake paid/refund/timeline or unsupported history query. REST transport, concurrency, idempotency, server-time expiry, action gates and cancellation retries verified separately from local/native/provider acceptance.

Implementation complete for local cohort. New Order/history/detail/payment UI consumes actual committed M3 identity. REST adapters and secure cancellation storage are PREPARED, not production-activated. Local fixtures contain unpaid projections with can_pay:false and no real receiver/QR. Server-time/monotonic expiry, version/action/identity guards, same-command cancel recovery and independent cancellation/compensation are implemented. A malformed quote ID now fails visibly before any create request, without redirecting to an older attempt's result.

Validation:12 new order/payment/cancel tests +2 360px navigation cases (text scale1/1.6) PASS, one added M3 invalid-ID regression PASS. Initial full run found two local legacy route regressions; repaired local-only non-UUID fixture routing with original auth/grant checks, then focused4/4 PASS. Final full187/187 PASS46s; analyze No issues14.5s. No assertions/lints disabled. Existing guest/B2B local fixtures and explicit ZIP cohort retained; no remote downgrade. Source7e00825 pushed/verified. Debug APK PASS94.6s,198276012bytes, build/app/outputs/flutter-apk/app-debug.apk; SHA256 and exact native evidence in CURRENT_STATE. Installed with -r; cold launch6116ms PASS. LG Android10 scrcpy smoke: product/SKU/add/cart/no-checkbox/quote account gate.163 app-PID log lines, zero crash/Flutter/overflow matches. At that delivery post-login native flow was NOT VERIFIED; follow-up above now verifies the local fixture path. Providers remain NOT VERIFIED. No dependencies/Android code/provider calls changed.

Local implementation and automated/build/native-fixture validation complete; follow-up above extends initial pre-login smoke to the full post-login local path. Next implementation now waits for user approval of auth-local persistence/ID isolation/credential validation, as explicitly requested after testing. Do not claim durable/backend registration or production acceptance. Real integration still needs owning-team evidence/new contracts and next-phase analysis before REST activation. External gates remain provider/Identity/BFF readiness, canonical geography create/edit, cursor inputs, tracking DTO and Care approval.
