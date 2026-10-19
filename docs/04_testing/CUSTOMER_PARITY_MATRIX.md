# Customer Web / Mobile parity matrix

Reference commit: `origin/feature/ui-web` `b1ff2918`. Mobile starting commit: `8e1994a7`. Statuses describe verified implementation, not intended completion. All Web paths below are under `apps/web-user/src`; source is read using `git show`, never edited. Common acceptance: loading/empty/error/retry, pending/disabled, keyboard/back/scroll and readable phone layout.

| Group / Web source | Mobile baseline | Gap / next verification | Phase |
|---|---|---|---|
| `app/globals.css`, shared layout/auth controls | Shared theme/widgets | Primary CTA gold instead of brown; ink/muted/border/error values differ. Match shared tokens first. | P1 |
| `app/login/login-form.tsx`, `app/register/registration-flow.tsx` | AuthPage + AuthCubit | Missing recovery link/return destination/terms/password checklist; weak registration validation. Login REST sends phone_number, ZIP requires identifier. ZIP requires registration email although Web labels optional; obey ZIP and document difference. | P1 |
| `app/account-recovery/recovery-flow.tsx` | Missing | Email neutral request, link verification, restricted proof, reset/success/relogin; expired/429/error states. | P1 |
| `app/home-experience.tsx`, `app/products/catalog-flow.tsx` | Home/Catalog/CatalogCubit | Compare category/search/filter/sort entrypoints, context preservation and recovery states; content sections still to inspect in detail. | P2 |
| `app/products/[slug]/product-detail-flow.tsx`, `components/product/product-reviews.tsx` | ProductDetail/variant sheet | Compare explicit SKU selection, unavailable combinations, review eligibility and related product links. Detailed audit pending. | P2 |
| `app/cart/cart-flow.tsx` | CartPage/CartCubit | Web selects available lines; Mobile initially checks out whole cart. Reconcile selection, unavailable/price-change state, quantity limits, confirmation and selected-line checkout. | P2 |
| `app/promotions/promotions-flow.tsx` | Missing | Voucher list/details/validation; ZIP supports list+validate. Checkout voucher submission is absent from ZIP; user clarification pending. | P2 |
| `app/checkout/checkout-flow.tsx`, `services/checkout-service.ts` | CheckoutPage/Cubit/REST | Mobile still requires district; ZIP shipping is province→ward only. Compare shipping/method/context/selection and error handling. | P2 |
| `app/payment/[orderId]/payment-flow.tsx`, confirmation flow | Confirmation only | Missing payment screen; preserve actual pending/payment states and expiry, no fake paid success. Detailed audit pending. | P2 |
| `app/account/profile/profile-flow.tsx`, account shell | AccountPage | No editable profile; inspect permission/capability and profile API projection. | P3 |
| `app/account/addresses/address-book-flow.tsx` | Address list/form | Compare two-level location and default/delete/validation semantics. | P3 |
| `app/account/loyalty/loyalty-flow.tsx` | Static account points | Missing loyalty detail; transactions are planned, overview stable. | P3 |
| `app/account/orders/orders-flow.tsx`, order-detail flow | Orders/OrderDetail | Compare filters, cancellation, payment/packing/shipping separation and reorder. | P3 |
| `app/track-order/page.tsx` and order services | Missing guest ownership lookup | Challenge/OTP/grant exact-order scope; no direct unrestricted order ID access. | P3 |
| after-sales request/case flows | Missing | Create/detail stable; eligibility/list/supplement drafts. Inspect approved Web interaction and keep lifecycle explicit. | P3 |
| support request/ticket flows | Missing | Context/create/detail/reply stable; ticket list draft. Preserve public/internal separation. | P3 |
| `app/trace/[qrCode]/page.tsx` | Missing | QR/manual code entry and public trace detail, invalid/not found/error recovery. | P3 |
| `app/b2b/b2b-request-flow.tsx`, quote flow | Missing | Verified B2B capabilities/context/request/released quote/accept; no customer draft/internal notes. | P3 |
| notifications (no matching Web customer route in inventory) | Local notifications | Preserve native feature; ZIP notifications planned, do not claim live integration. | P4 |

## Contract discrepancies to preserve explicitly

1. ZIP `POST /auth/login` requires `identifier`; existing Mobile used `phone_number`.
2. ZIP `POST /auth/register` requires email; Web treats email optional. Mobile will require it, per user's API authority.
3. ZIP Checkout shipping forbids `district`; native form/REST must adapt to two-level locations.
4. ZIP Checkout has no voucher property. Validate/list can be implemented, but applying a financial reduction to a submitted order needs an approved contract. No fabricated final total.
5. Password policy is present in Web (15–128 Unicode code points, uppercase/lowercase/digit/punctuation or symbol); ZIP is a string without tighter constraints. Reuse Web UX rules for registration/reset; login must accept existing shorter passwords.

## Validation evidence

Baseline analyze PASS, tests 60/60 PASS. P1 analyze PASS, focused tests 30/30 PASS, recovery navigation/keyboard 3/3 PASS. Native baseline: NOT RUN. Each phase records focused/automated/native results before its rows become DONE.
