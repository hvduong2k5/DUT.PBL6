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

Trace Web exposes a server-provided crafting video; shipment tracking also has packing video. No external-link capability exists in Mobile. Add Flutter-published url_launcher 6.3.3, verified against https://pub.dev/packages/url_launcher, behind one shared HTTPS-only button with launch failure recovery. Do not pass arbitrary schemes, credentials, auth tokens or private URLs to another app. Use direct launch and handle failure; no browser-state access or automatic media navigation. Quote document downloads are not in the stable schema and are not invented.

## Orders and cancellation authority (2026-10-20)

Stable ZIP supplies raw SKU/quantity/price without product names or unit-versus-line price semantics. Display exact SKU/raw price and authoritative subtotal/discount/shipping/final totals; do not multiply ambiguous REST price. Preserve unknown statuses instead of calling them pending. Actual carrier/checkpoints/seal/video come from the stable tracking endpoint; derived phase progress is labelled separately. Cancel POST permits only reason plus the idempotency header: use a reason input, preserve retry keys, and leave eligibility to the server. UI offers the request for known pending/unpaid statuses and registered order capability; Guest Web remains view-only. Do not copy Web's unsupported cancellation policy/refund fields or claim refunded after cancellation. Current referenced Web orders list/detail expose no reorder action; do not add an uncontracted reorder endpoint.

## After-sales context gap (2026-10-20)

Stable POST /returns requires authoritative line_id; stable GET Order exposes only sku_code/quantity/price, and draft eligibility defines lines as unspecified objects. Production cannot safely construct a return form's selectable lines. Keep a clear unavailable/Support recovery state and never call that draft endpoint implicitly. Implement stable detail and exact create adapter independently; default local fixture repository may issue its own LOCAL-LINE identifiers and exercise the approved form with one line, quantity bound, reason/details, HTTPS evidence URLs and bank details. Mock retains only case projection, not bank/evidence inputs. Approval/requested refund value never means money transferred. List/supplements/pickup remain draft or absent. This is an open production contract limitation, not a claim that backend creation was tested.

## B2B contract and uncertain acceptance (2026-10-20)

Require server actor B2B, feature-specific capabilities and VERIFIED organization context. Mock-only login fixture 0905999888 returns a dedicated verified identity; default registered/guest users have no B2B authority. Production never derives roles from phone/name. No unsupported request notes, fake upload, hard-coded specialist contact, quote document or internal draft data.

Stable acceptance body allows only confirmation/idempotencyKey. Re-fetch quote before POST, compare version and complete terms/items/totals/payment/expiry fingerprint, clear consent if changed and recheck capability. This precheck is not atomic version enforcement; server must uphold the stable summary. Do not invent quoteVersion/If-Match. After an uncertain POST, retain its key and replay to recover the existing result even if the same quote is now accepted; status alone is not changed commercial content. Rejected/expired/changed quotes are not newly accepted. Acceptance creates one actual mock order reference in PENDING_PAYMENT and never marks it paid.

## Catalog discovery scope (2026-10-20)

Stable list/search queries expose q/status/page/pageSize, without Web's advanced filter parameters. Use supported server search and pagination, retaining legacy convenience methods. Apply category-name/price/OCOP/confirmed-stock facets and price order to loaded results, show loaded/total scope until complete, and let users load more explicitly. No unbounded fetch, inferred weights/product types, unsupported query fields or premature global-match claims. Price presets follow Web customer-core mapping. Unknown Inventory is updating, never falsely sold out; detail entry stays available and actual SKU/cart guards remain intact. Home search opens Catalog; category selection passes through View more.

## Order snapshot labels and timeline correction (2026-10-20)

History now uses stable server q/status/page/pageSize and total/page_size metadata. Do not send Web-only year. Preserve exact payment/packing labels from raw status. Earlier generic progress mapped PAID to confirmed and synthesized future phases; stable Order does not supply those business events. Replace that presentation with actual recorded checkpoints or creation time only, leaving current logistics events to Tracking. This corrects an evidence gap without changing the server order contract.

## 2026-10-20 — Protected request expiry and product disclosure

A protected 401 triggers serialized existing auth/me + refresh verification through an optional transport callback; auth endpoints are excluded. RestAuthRepository signals recovered/lost identity through optional domain lifecycle observer. AuthCubit removes stale authenticated actor after loss. Do not automatically replay writes; screen retry retains operation idempotency and avoids duplicate financial actions. Network failures retain stored tokens and preserve original actionable failure.

Product detail requires explicit SKU choice even for one option, matching Web. Preserve all stable images, certificate number, variant weight/original price. Never infer food/allergen/storage/batch dates absent from ZIP; expose honest unavailable information and manual trace entry. Do not attach a sample batch to a purchased product as if verified.

## 2026-10-20 — Vietnamese native controls
Native B2B calendar showed English despite Vietnamese customer UI. Add Flutter SDK flutter_localizations and Vietnamese Material/Cupertino/Widgets delegates via GlobalMaterialLocalizations.delegates. Local SDK pins intl0.20.2; required lock downgrade from0.20.3 verified, no other existing dependency changed. Calendar suffix owns persistent FocusNode so dialog focus returns to calendar action rather than reopening date keyboard. Focused localized picker/request/acceptance3/3 PASS; full validation/build running.

## 2026-10-31 — Plan MVP contract migration from dev7f22226

User approved a four-step master roadmap, detailed analysis at each step entry, and M1 now. Public mvp-customer.openapi.yaml at7f22226c7b2f4363d3ba376b715feb0e63f69355 is the migration planning authority on overlapping routes; original ZIP snapshots remain unchanged historical provenance. Current runtime is still ZIP-based until explicit implementation slices. Trade-off: existing144-test local parity is a baseline, not proof of new contract compatibility; old QA-01 must migrate with payment contract.

Retain UI primitives/Cubit/adapters/mock separation. No merge, dev/Web writes, provider activation or dependency update. Proposed full-cart, canonical geography source/saved-only fallback, cursor inputs and typed tracking are **not approved architectural/product decisions**: G1–G3 in M1 record evidence and team decisions needed. No source behavior changed. M2–M4 get detailed plans at entry, after reviewing relevant dependency gates.
