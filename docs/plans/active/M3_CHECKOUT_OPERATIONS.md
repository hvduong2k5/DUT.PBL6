# M3 — Async checkout and recovery

2026-10-31. Classification LARGE within approved Mobile migration. User explicitly requests autonomous continuation through remaining steps; no approval pause between coherent slices. Only feature/mobile-ui may change/push. Dev7f22226 and Web remain read-only.

## Goal and dependencies

Replace the new cohort's selected-line/direct checkout with full-cart quote → explicit confirmation → accepted operation → authoritative outcome. M2-A/B complete (source4389c06,161 tests/analyze/APK PASS). Existing quote/canonical saved-address contracts are sufficient; no geography lookup or provider activation is inferred. Current entry source3572527, separate reference MD remains untracked.

Evidence: pinned public mvp-customer.openapi.yaml lines229–443 and2411–2505. POST /api/v1/checkout requires only quote_id/cart_revision/payment_method, visible ASCII Idempotency-Key16–128. New acceptance202 has null order_id; successful replay201 returns original operation/order. GET /checkout-operations/{id} is owner-authorized. States ACCEPTED/PROCESSING/WAITING_RETRY/COMPENSATING/SUCCEEDED/FAILED/MANUAL_REVIEW; SUCCEEDED requires order_id, FAILED requires error_code. Poll operations before placement. Unknown transport outcome requires same-key/body replay; keys never become reusable automatically.

## Architecture and scope

Keep new operation DTOs/repository separate from ZIP checkout. Add app-scoped Cubit observing AuthCubit, an attempt-store abstraction and secure-storage implementation. Persist only principal/scope, exact body/key and operation projection; no tokens, contact/address/bank payloads. Storage must succeed before POST. Distinguish local memory store from durable prepared REST integration: local backend/cart fixtures are process-local, so do not claim process-restart backend recovery in local mode. Secure store recovery will be tested with fresh instances.

An attempt belongs to authenticated principal + backend scope; logout/account change invalidates in-flight UI work without discarding unresolved ownership records. Never automatically create a new attempt after unknown/410/manual-review/storage failure. Replay the original body even if quote is now expired. Known terminal operation permits a later, explicitly confirmed new quote; same quote continues to show original outcome. Storage corruption blocks creation until resolved; do not silently erase it.

REST adapter sends exact fields/header and parses operation invariants. No automatic write replay; HTTP errors/invalid responses retain uncertainty. Bounded foreground GET polling only when operation identity exists; no POST timer, no polling after dispose/background/manual review/terminal state. Manual refresh recovers pending/offline states. Keep original attempt if a poll returns an unexpected ID or regressive snapshot.

Local adapter validates previously issued quote, revision and owner, implements idempotent ACCEPTED → PROCESSING → SUCCEEDED fixtures. Does not mark payment paid or infer provider readiness. UI reloads cart after success; no consume/clear call. Local cart preservation is explicit, not a cleanup assertion.

## Expected files

checkout/domain/mvp_checkout_models.dart and mvp_checkout_repository.dart; data/rest_mvp_checkout_repository.dart, mvp_checkout_store.dart, local_mvp_checkout_repository.dart; presentation/mvp_checkout_cubit.dart and mvp_operation_page.dart. Extend local quote issuance, app providers/dependencies/routes/auth return, CartPage and MvpQuotePage. Existing legacy integration test remains an explicitly legacy cohort; no assertions disabled.

## UX / security / failure modes

- Main new cart has full-cart totals and one CTA, no checkbox; pending request offers continuation. Existing remote ZIP runtime remains unchanged until independently enabled.
- Quote confirmation disabled while loading/expired; retained attempt takes precedence over a new placement. Fresh quote confirmation is explicit.
- Operation screen: accepted/working/retrying/compensating, unknown/offline, definitive failure, manual review, committed order. No fabricated order/payment success; explain back does not cancel processing.
- Narrow/large text, empty context, failed storage, expired quote, double taps, re-entry, login return, account transitions and stale async callbacks must be covered.
- No cart deletion, payment creation/QR, provider calls, new dependency, source contract edits, order-list queries or tracking invention.

## Implementation and validation

- [x] Typed operation/attempt invariants and exact REST payloads; immutable retry identity.
- [x] Memory and secure store, principal/scope separation and restart reconstruction tests.
- [x] App-scoped state machine, serialized start/retry/poll, session/stale guards and fail-closed storage.
- [x] Local fixture backend + full-cart main flow + operation route and recovery UI.
- [x] Focused tests including transport uncertainty, failed persistence, duplicate taps, mismatched poll/account switch; regression/analyze then build.
- [x] Update current handoff/decisions/master and Mobile-only verified checkpoint; continue M4 entry analysis without approval pause.

Definition of Done: new local purchase route creates exactly one operation/order fixture per attempt, no direct legacy handoff or client cart cleanup; prepared REST recovery preserves keys/body across instance recreation; all applicable state/transport/widget checks pass; local/provider/native evidence distinguished. Production activation remains external readiness work.

## Current state / checkpoint

Implementation COMPLETE for local cohort; REST adapter and durable store PREPARED, remote activation NOT APPLIED. All implementation checklist items above are satisfied except the intentionally deferred APK/native check (build once after M4). Main local cart/deep links now enforce quote confirmation; pending continuation survives leaving the page, success reloads cart without cleanup. PublicError operation_id preserved as a recovery identity without inventing an operation snapshot. Timer polls GET only in foreground, up to20 attempts per page visit; unknown POST requires explicit same-key recovery. Two existing legacy/preview tests select localMvpCheckout:false rather than disabling assertions.

Validation:9 new domain/REST/storage/state tests +2 navigation tests PASS. Full172/172 PASS37s; analyze No issues18.7s; git diff --check PASS. No new dependencies/provider calls/native test. Secure-store reconstruction and owner/backend isolation tested; local default explicitly uses process-local memory and does not claim app-kill recovery. Current checkpoint source derived from parent3572527; verify git log/remote. Exact next action: continue M4 entry analysis/plan and implement order/payment/cancellation on actual committed identity. Canonical address create/edit/geography, cursor queries and typed shipment timeline remain separate contract decisions.
