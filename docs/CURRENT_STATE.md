# Customer Mobile — current handoff

Updated 2026-10-31. Migration IN PROGRESS. Branch feature/mobile-ui; M3 checkpoint parent357252745cac8dbcd0a77cd78fa54452aa5b64d1, new source identified by git log -1. Only Mobile source/tests/task docs changed; no merge/PR/dev/Web writes.

## Authority and current phase

Read-only dev7f22226c7b2f4363d3ba376b715feb0e63f69355 public MVP integration target; historical ZIP provenance and Webb1ff2918 visual foundations unchanged. Full-cart accepted; no geography supplier chosen. Master docs/plans/MASTER_PLAN.md; detailed completed-local slice docs/plans/active/M3_CHECKOUT_OPERATIONS.md. User requests autonomous continuation through remaining approved work, not a pause at checkpoints.

## Completed

M2-A/B typed cart/address/quote and local review retained. M3 now connects explicit quote confirmation to accepted operation and committed order identity. App-scoped MvpCheckoutCubit tracks identity, persists exact input/key before POST, serializes duplicate taps, rejects stale/mismatched responses, restores prior attempts and preserves PublicError operation_id. Never generates another key for uncertain retry. Failed storage blocks POST/new attempts. Owner/scope isolation and account changes are tested. Secure store and REST operation adapter are prepared; local runtime intentionally uses memory storage with process-local backend fixtures.

Main local cart has full-cart totals/CTA and no checkboxes; pending requests offer continuation even with an empty cart. Old /checkout deep link redirects to quote in the new cohort. Legacy flow is still a separate, explicitly tested cohort via localMvpCheckout:false; remote runtime remains ZIP-based/unactivated. Quote review offers confirmation only while usable and no unresolved previous request exists. Operation screen supports working/unknown/retry/manual-review/terminal states and bounded foreground GET polling. Success reloads cart and never consumes/clears it on the client. No actual payment/QR/refund is implied.

Modified areas: app dependencies/providers/router, CartPage, quote issuance/page, new checkout DTO/repository/store/REST/local adapters/Cubit/operation page, Failure/PublicError recovery metadata, new contract/state/navigation fixtures/tests. M3 phase plan describes exact files and boundaries. Separate untracked docs/MOBILE_MVP_DISCUSSION_REFERENCE.md belongs to Issue38; preserve and do not stage.

## Validation

M3 adds9 state/transport/storage tests +2 integrated navigation tests. Focused17/17 with existing legacy/quote navigation PASS; operation navigation2/2 PASS. Full172/172 PASS37s; analyze No issues18.7s; diff check PASS. Existing tests retained; two legacy/preview fixtures explicitly select their prior cohort. New dependencies NONE. Current source APK/native NOT RUN yet; prior APK4389c06 contains only M2-B. Build once after M4 integration. Real backend/provider acceptance NOT VERIFIED. Local backend/cart/attempt recovery is process-local; secure-store reconstruction tests do not prove local backend survives app kill.

## Important limits

Unknown HTTP outcomes remain unresolved until an authoritative operation is recovered; no error-status-only inference of cancellation or order failure. Manual review/410/irrecoverable metadata uses retry/support, never a new key. Saved-address create/edit/geography source, cursor input parameters/global order search and typed shipping timeline are open contract decisions. No fabricated endpoints, production activation or cleanup acceptance.

## Exact next action

Continue M4 now: write detailed entry plan from pinned public Order/payment-instructions/payment/cancel schemas and current pages. Add separate typed MVP order/payment projections, read-only payment instructions/status and versioned cancellation/recovery; wire committed M3 order to those pages. Honor allowed_actions, independent lifecycle/payment/compensation and server-time payment deadlines. No client SePay call, pretend-paid button or sample real bank/QR. Keep incomplete cursor/tracking capabilities gated. Then focused/full/analyze/APK validation and one bounded device check if available, update evidence and push Mobile-only checkpoint without awaiting another approval.
