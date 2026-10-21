# P4 — Customer parity validation and delivery

Updated 2026-10-21. ACTIVE; native cancellation and P1–P3 evidence reconciliation complete; QA-01/QA-02 remain explicit.

## Goal, dependencies and scope

Verify P1/P2/P3 implementations against read-only Customer Web and unchanged API ZIP on Android. Implementation slices and native cancellation are recorded; P1–P3 acceptance reconciliation is complete with targeted evidence gaps QA-01/QA-02 still open. Production voucher application and returns line-ID context depend on external contract extensions. Only Mobile source/tests/task documents may change; push feature/mobile-ui only, no merge/PR.

## Steps and acceptance criteria

1. Audit session expiry, keyboard, large text, back navigation, loading/empty/error/retry and order states.
2. Validate recovery/login, discovery/filter/detail/review, cart/checkout/payment, account/addresses/loyalty, history/tracking/cancel, Support/reply, trace, returns and verified B2B on local Android.
3. Analyze, test and build stable source; record physical, automated and production evidence separately.
4. Resolve every parity row or document a precise platform/contract limitation; reconcile phase acceptance, matrix and handoff.
5. Commit/push only intentional Mobile work and task docs; verify remote hash. Checkpoint alone does not satisfy the whole goal.

## Implemented fixes and decisions

- Protected401 transport callback reuses serialized auth restore/refresh and optional domain observer; invalid refresh clears stale actor/capabilities, offline retains tokens. Auth endpoints excluded; concurrent recovery shared. No automatic mutation replay; explicit screen retry preserves idempotency.
- Product gallery/certificate/variant weight/original price retained; explicit SKU even for one option; unavailable food fields and manual trace instead of inferred batch association.
- Shared input outside-tap unfocus and flexible subtotal fix native keyboard refocus/27px overflow.
- Flutter SDK localization sets Vietnamese Material/Cupertino/Widgets. SDK requires intl0.20.2 instead of previous0.20.3; no other dependency upgrade. Persistent calendar action FocusNode prevents keyboard restoration.
- Review keyboard inset moved outside ScrollView so viewport shrinks and caret reveals automatically. Regression asserts visibility with400px keyboard; physical hot reload confirms.

## Current automated evidence

Cancellation slice: analyze PASS110.5s, full144/144 PASS, debug APK PASS166.0s, focused order actions/history10/10 PASS. Two regressions prove stale badge and stale pending-filter result before fix, pass after awaiting detail return/reloading. Search/filter stay selected. Initial generic pageBack harness corrected to actual custom-header tooltip; no assertion disabled.

## Physical Android evidence (local mocks only)

| Flow | Observed | Remaining limit |
|---|---|---|
| Recovery/auth | Neutral/invalid/expired; valid mock link→reset checklist/keyboard→success→required login→account | No backend password verification/revocation or email proof |
| Guest tracking | Challenge/contact/OTP→OM-8921 detail→tracking | No real SMS/cookie lifecycle proof |
| Purchase/cart | Explicit SKU→add/cart; clear cancel preserves line; two-level guest address→COD85,000→unpaid payment→same detail | Other error states covered automatically |
| Account | Profile save, loyalty→promotions, address default/edit/save | No draft history/redemption integration |
| B2B | Local verified org, MOQ20/date→YC-LOCAL-2; released quote/consent→accept→LOCAL-B2B-001 pending4,180,000; Vietnamese calendar/no keyboard return | No real commercial transaction; atomic quote version remains server responsibility |
| Reviews | Owned OM-8860/exact SKU→comment→submit→5/5/count1/verified badge; caret auto-reveal after fix | No production posting/upload |
| Trace/discovery | Empty validation→labelled sample detail/disclaimer→exact search→empty→clear; under200k filter returns3 | Specimen differs from catalog name; no invented association |
| History/returns | Status empty→clear4orders; OM-8874 exact line/max2→quality/details/example URL/fake bank→LOCAL-RET REQUESTED/refund0 | Production stable line-ID context unavailable; no money transferred |
| Support | Owned order/type→local ticket→public reply visible/composer cleared | No real CSKH transmission/upload |
| Saved address/VietQR | Address autofill; OM-82047142 pending85,000→countdown/explicit missing QR→Check status clicked | Historical Check status result unverified; mock has no QR URL. OM-82047142 cancellation now observed before fixture reset |

Current cancellation APK installed update Success without uninstall/wipe. Relaunch reset memory-only fixtures. Prepared one replacement LOCAL-B2B-001 through existing released mock quote. Physical pending filter→detail/reason/confirm→cancelled/action hidden→back preserves pending filter0/0empty→clear shows5cards with cancelled replacement. No real financial/provider operation; no clean-install claim.

## Checkpoint / exact next step

### Bounded step 2026-10-21: native cancellation

Resume d6acb9a verified locally/remotely; worktree clean and ADB connected. Scope is only native cancellation and a directly observed defect, not final acceptance reconciliation. Physical OM-82047142 still exists: empty reason blocked, keyboard/input/confirmation usable, cancellation returns Đã hủy and hides action. Returning to history leaves stale Chờ thanh toán. Verified cause: history card pushes detail but never reloads OrdersCubit when detail returns. Minimal fix implemented: await navigation, then reload while mounted using current query/status; no API/contract change. Regressions cover all-orders and pending-filter return, including empty recovery after cancellation. Automated and installed-APK physical checks PASS as recorded above. Other P4 work stays OPEN.

### Evidence reconciliation 2026-10-21

Read branch/HEAD/status/remote, P1–P3/MASTER/CURRENT/matrix, Web24page inventory/read-only globals.css, Mobile routes/theme/CTA and relevant source/tests. ZIP+3snapshot hashes match SOURCE. Historical P1–P3 pending/source-not-started notes replaced with current evidence/limitations; no source changes or computer-use calls. Source e86f82b retains prior144/144/analyze/APK PASS; no redundant test/build rerun for docs only.

Result: all24Web destinations have routed counterparts; P1 verified locally; P2/P3 main flows implemented and locally exercised. Full acceptance remains conditional because QA-01 payment status interaction is not covered by the expired-QR-only widget, and QA-02 external-media OS launch remains unverified. See matrix for precise evidence and external-contract limits. Do not classify a missing test as a proven source defect.

## Exact next task — QA-01 payment status, no broad device loop

- Goal: prove PaymentPage Check status derives pending/paid/cancelled from repository, shows loading, blocks duplicate reads while pending and recovers from a read failure.
- Files: test/features/checkout/purchase_navigation_test.dart (or a focused neighboring payment test); PaymentPage source only if a failure demonstrates a defect. Reuse existing mocktail/repository/provider test setup; no new dependency/API/provider/QR service.
- Cases: pending response never claims paid; PAID response shows recorded payment and hides QR; cancelled response shows cancelled and hides QR; error visible then retry succeeds; two taps while unresolved issue one read; dispose safely before response.
- Validate focused test first; appropriate analyze/full/build only if implementation changes justify it. Unit/widget evidence cannot establish real provider verification. Avoid network QR/image loads with controlled fixture or expired context.
- QA-02 stays separate: one targeted native handoff only if approved/testable media exists; no repeated completed trace/purchase flows.
- After this bounded test slice, update matrix/handoff and Mobile-only checkpoint. No merge/PR. Whole goal stays ACTIVE until remaining acceptance/disposition is recorded.
