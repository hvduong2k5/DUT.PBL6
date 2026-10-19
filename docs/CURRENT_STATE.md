# Customer Mobile parity handoff

Updated 2026-10-19. Goal ACTIVE, full parity NOT COMPLETE.

## Branch / checkpoints / reference

- Branch `feature/mobile-ui`; HEAD before current purchase checkpoint `82fd35890eba8355dba5e3b1251212779f3ee4af`, pushed and remote SHA verified. This handoff is part of the next purchase milestone; resolve its resulting SHA with git log.
- Read-only Web `origin/feature/ui-web` at `b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e`. No checkout, edits or merge of Web. ZIP Customer contracts preserved in docs/03_api_specs/frontend-contracts with hashes and -text attributes.
- Plans: docs/plans/MASTER_PLAN.md; P0 inventory, P1 auth and P2 purchase active plans. Matrix docs/04_testing/CUSTOMER_PARITY_MATRIX.md. Decisions docs/DECISIONS.md.

## Completed slices

- P1: Web-aligned tokens and brown primary CTA; login identifier payload; registration required email/terms/Unicode password checklist; safe return-location through login/register/recovery; recovery request/neutral sent/link proof/reset/relogin, 410/429 and structured Error metadata. Scoped Android recovery custom URI.
- P2 first slice: available-line cart selection/reconciliation; pending/error mutation guards; only selected lines passed to Checkout; unselected lines preserved after order creation. Strict ZIP Checkout payload/header, no district/is_gift/item price; two-level location selection with dependent ward reset; idempotency renewed after failed payload edits. Added voucher list/condition validation with REST/mock/UI entrypoints and stale-response protection. Product list actions now open explicit SKU detail; no invented SKU from product ID; explicit availability/stock-unknown state, actual image/fallback renderer and success only after successful add.
- User decision received: keep ZIP unchanged; actual voucher application to submitted Checkout is waiting for contract extension. Do not send voucher_code or subtract a preview discount from total.

## Validation / native

- Baseline analyze PASS, 60/60 tests PASS.
- P1 focused 30/30, recovery navigation/keyboard 3/3 PASS; Android debug build PASS (321.5s), APK installed by update preserving app data.
- P1 scrcpy observed: empty-email validation, email entry+keyboard, neutral sent response, mock link to reset, live Unicode password checklist, scrolling with keyboard to confirmation/submit, expired proof safely returns to request with explanatory error. Native successful-reset submit/relogin not yet exercised; automated widget flow passes. No real email/provider/backend validation claimed.
- Current full Flutter suite 80/80 PASS; most recent analyze PASS before two tiny product padding/didUpdateWidget changes (covered by suite; final analyze still required).
- P2 native build/install/QA NOT RUN yet. Existing scrcpy selected via @oai/sky in node_repl; window id 4130356, process E:/scrcpy-win64-v4.1/scrcpy.exe. Refresh window/state before inputs. Phone is showing recovery expired state of P1 APK. ADB authorized device exists; do not hard-code it into tracked docs/source.

## Modified scope / remaining

- Intentional areas: apps/mobile-app Android ignore; app DI/router; catalog/cart/checkout/promotions and shared image/navigation; account promotions link; targeted tests; parity docs. Generated desktop registrants have line-ending-only tool noise (not staged); android/.kotlin ignored. Do not commit caches/build outputs/screenshots.
- P2 still OPEN: confirmation/payment screen must use actual response QR/expiry/status without fake success; address/location edge cases and cart-cleanup warning; meaningful populated-cart/checkout/promotion widget tests; Home/catalog filter/sort/asset alignment; reviews. No dedicated Payment endpoints should be invented.
- P3 still OPEN: profile/address/loyalty/order/guest ownership/after-sales/support/trace/B2B parity. Read corresponding Web source and ZIP schemas before each slice. Guest session/grant integration and current mock order fallback require inspection. No policy decisions for draft/planned APIs should be silently promoted.
- Exact next action: create/verify purchase milestone checkpoint on feature/mobile-ui, then implement confirmation/payment presentation from stable Checkout fields and Order read, add populated flow tests, rerun focused checks/build and use scrcpy. Then continue remaining roadmap; do not mark goal complete at checkpoint.
- No external blocker to independent work. Voucher application is an explicit accepted contract limitation. Push only feature/mobile-ui; no merge/PR.
