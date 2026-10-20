# Customer Mobile parity decisions

## 2026-10-19 — Baseline and authority

Approved Customer Web at `b1ff2918` is presentation/flow reference; supplied ZIP is API authority. Keep the current Flutter architecture and isolated `feature/mobile-ui`. Web is inspected through Git objects, so its checkout and files are not modified. Direct implementation/testing is authorized by the current request and replaces historical Stitch prompt review for this task.

## 2026-10-19 — Auth contract and native adaptation

Use ZIP login `identifier`, registration required email, and existing snake_case user/token projection. Web's optional email conflicts with required ZIP schema; require email in Mobile. Registration/reset reuse Web 15–128 Unicode code point password policy; login continues accepting existing nonempty passwords. Recovery follows email/link/restricted-proof/reset/relogin rather than inventing OTP endpoints.

Add only a scoped Android custom URI `oma://customer/account-recovery?recoveryToken=...` to enter the recovery route. The server must verify any token; this is not a verified HTTPS app-link claim or provider configuration. Email delivery/link generation is external to this UI task. Local mock has an explicitly labelled non-email test link. Browser HttpOnly cookie text is not copied into native UI.

## Open contract issue — Voucher at Checkout

Stable ZIP exposes voucher list and validation but Checkout forbids extra properties and has no voucher code. Customer Web validates a voucher in its BFF flow. Do not claim an applied discount in real submitted totals without an approved contract. User confirmed: keep ZIP unchanged; mark voucher application at Checkout as waiting for a contract extension. Implement discovery/validation only, with no submitted reduction.

## 2026-10-19 — Native guest cookie transport

ZIP guest_session/guest_order_grant schemes are server-issued cookies. Native Dio had no cookie capability; X-Session-Id is not a substitute for a signed ownership grant. Add `dio_cookie_manager` + `cookie_jar` at the existing ApiClient boundary, memory-only with normal expiry/domain/path restrictions and a web-safe conditional stub. No new provider or disk persistence. Clear cookies on logout/reset. Do not fabricate grants or assume the backend issues them until integration is verified. Bearer authentication remains supported. Native JSON API redirects are not followed automatically.

Primary package docs checked: https://pub.dev/packages/dio_cookie_manager and https://pub.dev/packages/cookie_jar. No equivalent dependency was present.
## Support parity and stable contract boundary (2026-10-19)

Web support request collects guest contact and attachment metadata. Stable ZIP POST /support/tickets forbids a contact property; no stable upload/download contract exists. Mobile submits only documented fields (including empty linkedOrderId/attachments when unused), retains server session authority, displays public attachment metadata without inventing download URLs, and does not collect unsendable guest contact. Upload/contact extension remains a contract gap. Ticket list is draft; expose create/detail/public reply without claiming an authenticated ticket inbox. Do not copy Web placeholder hotline or static response-time claims; show server context/servicePromise.

## Session and capability boundary (2026-10-19)

Restore identity using GET auth/me before showing protected account data. On 401 only, retry through documented refresh and then verify identity again. Keep tokens on network failure; clear expired credentials and clear local session even if server logout is unreachable. Serialize explicit REST auth actions after pending restore. Reload server capabilities on auth transitions; discard stale responses and show loading/error/forbidden states. Never derive production B2B authority from phone, display name or a client toggle. Mock users remain REGISTERED until an explicit verified fixture is introduced.

## Opening public media links (2026-10-19)

Trace Web exposes a server-provided crafting video and released B2B quotes may link public documents. No external-link capability exists in Mobile. Add Flutter-published url_launcher 6.3.3, verified against https://pub.dev/packages/url_launcher, behind one shared HTTPS-only button with launch failure recovery. Do not pass arbitrary schemes, credentials, auth tokens or private URLs to another app. Use direct launch and handle failure; no browser-state access or automatic media navigation.

## Orders and cancellation authority (2026-10-20)

Stable ZIP supplies raw SKU/quantity/price without product names or unit-versus-line price semantics. Display exact SKU/raw price and authoritative subtotal/discount/shipping/final totals; do not multiply ambiguous REST price. Preserve unknown statuses instead of calling them pending. Actual carrier/checkpoints/seal/video come from the stable tracking endpoint; derived phase progress is labelled separately. Cancel POST permits only reason plus the idempotency header: use a reason input, preserve retry keys, and leave eligibility to the server. UI offers the request for known pending/unpaid statuses and registered order capability; Guest Web remains view-only. Do not copy Web's unsupported cancellation policy/refund fields or claim refunded after cancellation. Current referenced Web orders list/detail expose no reorder action; do not add an uncontracted reorder endpoint.

## After-sales context gap (2026-10-20)

Stable POST /returns requires authoritative line_id; stable GET Order exposes only sku_code/quantity/price, and draft eligibility defines lines as unspecified objects. Production cannot safely construct a return form's selectable lines. Keep a clear unavailable/Support recovery state and never call that draft endpoint implicitly. Implement stable detail and exact create adapter independently; default local fixture repository may issue its own LOCAL-LINE identifiers and exercise the approved form with one line, quantity bound, reason/details, HTTPS evidence URLs and bank details. Mock retains only case projection, not bank/evidence inputs. Approval/requested refund value never means money transferred. List/supplements/pickup remain draft or absent. This is an open production contract limitation, not a claim that backend creation was tested.
