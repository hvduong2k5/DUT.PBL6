# Customer Mobile parity decisions

## 2026-10-19 — Baseline and authority

Approved Customer Web at `b1ff2918` is presentation/flow reference; supplied ZIP is API authority. Keep the current Flutter architecture and isolated `feature/mobile-ui`. Web is inspected through Git objects, so its checkout and files are not modified. Direct implementation/testing is authorized by the current request and replaces historical Stitch prompt review for this task.

## 2026-10-19 — Auth contract and native adaptation

Use ZIP login `identifier`, registration required email, and existing snake_case user/token projection. Web's optional email conflicts with required ZIP schema; require email in Mobile. Registration/reset reuse Web 15–128 Unicode code point password policy; login continues accepting existing nonempty passwords. Recovery follows email/link/restricted-proof/reset/relogin rather than inventing OTP endpoints.

Add only a scoped Android custom URI `oma://customer/account-recovery?recoveryToken=...` to enter the recovery route. The server must verify any token; this is not a verified HTTPS app-link claim or provider configuration. Email delivery/link generation is external to this UI task. Local mock has an explicitly labelled non-email test link. Browser HttpOnly cookie text is not copied into native UI.

## Open contract issue — Voucher at Checkout

Stable ZIP exposes voucher list and validation but Checkout forbids extra properties and has no voucher code. Customer Web validates a voucher in its BFF flow. Do not claim an applied discount in real submitted totals without an approved contract. User clarification requested; independent implementation continues.
