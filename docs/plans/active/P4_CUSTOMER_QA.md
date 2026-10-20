# P4 — Customer parity validation and delivery

Updated 2026-10-20. ACTIVE; checkpoint requested by user for later continuation.

## Goal, dependencies and scope

Verify P1/P2/P3 implementations against read-only Customer Web and unchanged API ZIP on Android. Implementation slices exist; native cancellation and final acceptance reconciliation remain open. Production voucher application and returns line-ID context depend on external contract extensions. Only Mobile source/tests/task documents may change; push feature/mobile-ui only, no merge/PR.

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

Source f4e0a085b55b0ed2ac97d59bb3531bfd0b898512: analyze PASS, full142/142 PASS, debug APK PASS37.4s. Review focused4/4 PASS; earlier session focused21/21 and cart/purchase8/8 PASS. No source change after this validation; documentation checkpoint requires no repeated expensive build.

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
| Saved address/VietQR | Address autofill; OM-82047142 pending85,000→countdown/explicit missing QR→Check status clicked | Result needs fresh observation; mock has no QR URL; native cancel NOT RUN |

Localization APK718159e installed update without wipe; review fix hot-reloaded on physical device. Current f4e0a08 APK built but not installed after hot reload. No claim of fresh-install verification or real provider processing.

## Checkpoint / exact next step

Source f4e0a08 already pushed. Documentation checkpoint records new trace/discovery/history/returns/Support/VietQR evidence, updates matrix and roadmap, and preserves unfinished state. See CURRENT_STATE for runtime hints. Do not repeat completed flows without a reason.

Next: rediscover device/runtime; freshly observe Check status on OM-82047142, open detail, test local cancellation reason/confirmation and returned status. Then reconcile P1–P3 acceptance/matrix and full Definition of Done. Keep contract blockers explicit. If source changes, rerun focused checks plus appropriate final analyze/tests/build. Update handoff and verify Mobile-only remote push before final delivery.
