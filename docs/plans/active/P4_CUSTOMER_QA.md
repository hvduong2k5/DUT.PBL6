# P4 — Customer parity validation and delivery

## Goal and dependencies

Verify implemented P1/P2/P3 against read-only Customer Web and unchanged ZIP on Android. P2 reviews and cart clear implemented (focused reviews 4/4 and cart/purchase 8/8 PASS); native and regression remain open. Production voucher application and returns line-ID context are external contract gaps.

## Scope and steps

1. Audit session expiry, keyboard, large text, back, loading/empty/error and order states.
2. Verify local Android recovery/login, catalog/filter/detail/review, cart/checkout/payment, profile/address/loyalty, history/tracking, support/reply, trace, returns and verified B2B.
3. Run analyze, full test suite and APK build with stable source; distinguish native from automated and production evidence.
4. Reconcile matrix/plans/CURRENT_STATE and Mobile-only checkpoint/push. No merges/PRs or uncontracted endpoints.

## Verified issue and design before modification

Protected API 401 currently becomes a screen error without revalidating global identity. Restore/refresh works only at startup. Add an optional session-recovery callback at transport; RestAuthRepository reuses serialized restore/refresh and reports identity loss through an optional domain observer; AuthCubit clears stale authenticated UI. Exclude auth endpoints to avoid recursion. No automatic write replay: screen retry preserves existing idempotency. Concurrent failures share recovery; explicit auth actions stay serialized. Retain tokens offline. No new provider/dependency/endpoint.

Expected files: core/network/api_client, auth domain/REST/Cubit, focused session tests. Preserve existing auth payloads and 401/403 distinctions.

## Acceptance / validation

Protected 401 rechecks once without auth recursion; valid refresh retains identity, invalid refresh removes actor/capabilities; no duplicate writes; 403/network errors never trigger refresh; concurrency, explicit-auth races and offline token retention tested. Auth regressions pass. Native fixture tests cannot prove backend/email/SMS/payment.

## Status / next

Session recovery and cart confirmation implemented. Auth focused 21/21 PASS, cart/purchase 8/8 PASS, analyze PASS and full 139/139 PASS. APK build running with source held stable. Logistics known codes translated, unknown values preserved. Local created-order snapshots now retain exact SKU.

Product audit slice implemented: gallery preserves all images; certificate number/variant weight/original price retained; explicit SKU choice even for one available variant; food-information unavailable state and manual trace recovery. ZIP provides these product fields but no food fields: no fabricated ingredients or inferred batch/sample certificate association. Focused catalog/reviews/viewport 32/32 PASS; new explicit-selection widget passed, projection assertion corrected to compare Money BigInt. Rerun final full/analyze/build then native. Whole task remains ACTIVE.

Final source validation: analyze PASS; full 141/141 PASS; APK build PASS 35.3s. Install/native next.

Native b8ff216: explicit SKU disabled→selected→add badge/cart; clear dialog cancel preserves line; checkout keyboard +two-level Huế/Thuận Hòa→actual COD order85,000→payment explicitly unpaid→exact order detail all observed. Verified keyboard refocus after dropdown; shared AppTextField onTapOutside unfocus patch added, final checks/native confirmation pending. USB/scrcpy reconnected; active window9373924, attach97559. Local B2B login returned to account, profile save acknowledgement observed. No real order/payment/provider operation.

Keyboard/summary patch: focused checkout3/3 PASS including address focus assertion; final analyze PASS and full141/141 PASS. APK rebuild still running (session9810); source held stable. Native B2B verified organization/request layout/date picker observed. New concrete UI defect: Material date picker labels are English while app/Web is Vietnamese. Local installed Flutter SDK flutter_localizations/pubspec.yaml pins intl0.20.2; existing lock has0.20.3. After build ends, add SDK flutter_localizations and Vietnamese delegates/locale; required intl patch downgrade is justified by SDK constraint (no other upgrade). Add focused localized picker test, then validate/build again.

Localization implemented with exact SDK dependency/intl pin and calendar focus action; focused B2B3/3 PASS. Native local verified request MOQ20/date2026-10-27→YC-LOCAL-2 received observed. New request is not payment and not sent to a real specialist. Full tests29960/build41720 running with source stable; prior keyboard APK172.1s PASS. Current scrcpy68230 at existing quote link just clicked.

Localization milestone final: analyze PASS; full142/142 PASS; APK69.6s PASS. Native B2B full request→receipt plus existing released quote→consent→accept→reference order LOCAL-B2B-001 PENDING_PAYMENT total4,180,000 observed. No real transaction. Remaining native calendar language patch, recovery successful reset/relogin, QR, support reply, trace, returns, discovery/reviews/history. Goal ACTIVE.

Native new localization APK installed update Success (no wipe). Recovery custom URI with explicit mock-token→valid reset form→all checklist rules→keyboard scroll/confirmation→local completion requiring login, no auto-auth observed. This does not prove backend password/session revocation/email. Current scrcpy68230, recovery success return-login just clicked. Localizations APK code718159e; physical calendar confirmation pending.

Native718159e: APK update Success; mock recovery URI→valid form/checklist/keyboard/confirmation→success requiring login→relogin→account observed. Mock accepts nonempty password, so no claim of real credential verification. Vietnamese date labels/month/day/Huỷ and date selection OK with no keyboard restoration physically confirmed. Phone B2B form2026-10-27, signed-in local verified fixture. Remaining reviews/support/trace/returns/discovery/history/QR. Source unchanged since142/142/analyze/build69.6s PASS.

Native reviews718159e: product→write→owned OM-8860→exact Hộp500g→comment keyboard/manual scroll→submit→public aggregate5/5 count1 and verified purchase badge/content observed. New UX issue: keyboard padding is inside sheet ScrollView, so viewport still extends behind keyboard and focused comment needs manual scroll. Move keyboard inset outside scroll viewport under existing height constraint; add focus-visible narrow-phone regression, rerun focused/full/build before installing. Current phone product review success; no production review posted.

Review keyboard fix: outer keyboard inset now shrinks sheet scroll viewport; focused tests4/4 PASS and strengthened400px keyboard caret assertion PASS. Native hot reload26739 loaded1 library; open form→tap comment automatically reveals caret above keyboard without manual scroll, observed. Full75903/build16353 running; hold source stable.

Review keyboard milestone final: analyze PASS, full142/142 PASS, APK37.4s PASS; source stable. Physical auto-reveal caret observed after hot reload, no manual scroll. Reviews full native submission/count/verified badge already observed. Remaining trace/support/returns/discovery/history/QR. Phone product detail reviews success, trace buttony340, scrcpy68230, attach26739.
