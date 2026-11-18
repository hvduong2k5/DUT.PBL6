# Goi Hue heritage UI: web and mobile parity

Single checklist so `apps/web-user` (Next.js) and `apps/mobile-app` (Flutter) stay in step.
Source of truth for design: the Goi Hue kit (`mon-02/bo-ui-v01`). Web tokens: `apps/web-user/src/styles/heritage.css`.
Mobile tokens: `apps/mobile-app/lib/core/theme/heritage_tokens.dart`. Change a token on both sides in the same PR series.

## Tokens (same hex on both sides)
| Token | Hex | Web | Mobile (`HeritageColors`) |
|---|---|---|---|
| kem | `#FCF9F4` | `--kem` | `kem` |
| kem-am | `#F6EFE2` | `--kem-am` | `kemAm` |
| kem-dam | `#EDE0CC` | `--kem-dam` | `kemDam` |
| nau-dam | `#563A26` | `--nau-dam` | `nauDam` |
| nau-cung-dinh | `#352014` | `--nau-cung-dinh` | `nauCungDinh` |
| nau | `#70513B` | `--nau` | `nau` |
| mat | `#D4A24C` | `--mat` | `mat` |
| do-chu-sa | `#8B261D` | `--do-chu-sa` | `doChuSa` (`AppColors.brickRed`) |
| xanh-ngoc | `#2D5A3C` | `--xanh-ngoc` | `xanhNgoc` (`AppColors.success`) |
| muc / phu / vien / loi | `#2F241C` / `#655241` / `#DFD1BE` / `#8F2817` | same names | `muc`, `phu`, `vien`, `loi` |

Fonts: headings Playfair Display (lining figures on mobile so prices read `170.000đ`), body Be Vietnam Pro.
Radii: 2px buttons/inputs, 4px large blocks, 20px bottom sheet. Minimum text 13px, tap targets >= 44.

## Shared components
| Kit element | Web | Mobile |
|---|---|---|
| Frame levels (imperial, plaque, card, functional) | `gh-frame-*` | `HeritageFrame` + `HeritageFrameLevel` |
| Button (gold, brown, paper, ghost) | `gh-btn*` | `HeritageButton` (`AppPrimaryButton`/`AppSecondaryButton` wrap it) |
| Section header (kicker, rule, divider) | `gh-section-header` | `HeritageSectionHeader` |
| Hero banner (3s loop, dots, blur arrows) | `HeritageBanner` | `HeritageBanner` |
| Paper / parchment / imperial surface | `gh-surface-*` | `HeritageSurface` |
| Motif (pointer-events none) | `Motif` | `HeritageMotif` |
| Emblem (logo, only decorative mark) | `.gh-emblem` | `HeritageEmblem` |
| Shell icons (search, cart, home, grid, gift only) | `icons.tsx` | `HeritageIcon` |
| Bottom sheet | `gh-sheet` | `showHeritageSheet` |

Assets are copied, never redrawn: web `public/images/heritage/`, mobile `assets/heritage/`.

## Flow parity
- Navigation differs on purpose (2026-11-18 decision): native mobile follows the Shopee/Lazada pattern.
  - Mobile tabs: Trang chu, Danh muc (own page: category rail + product grid), Gio hang, Thong bao, Tai khoan. Tabs only navigate; no side menu (its links live in Tai khoan).
  - Mobile search: search bar on Home and Danh muc opens a separate search screen; the 5 latest queries are stored on the device.
  - Mobile Uu dai: Home block + Tai khoan link (not a tab).
  - Web (responsive) keeps: Trang chu, Danh muc (sheet), Uu dai, Tim kiem (sheet), Gio hang. Consider aligning web later.
- Home order: hero, featured products, collections (real categories), story, trace search, account CTA.
- Header: imperial brown, gold 3px rule, emblem + "Goi Hue".
- No API, contract or backend change. Where data is missing the block stays empty; no mock in main code.

## Per-screen checklist (mobile)
| Screen | Status |
|---|---|
| Shell (search header, 5-tab nav, Account hub) | done |
| Search screen, Categories tab | done |
| Home | done |
| Catalog, product detail | done (reviews section re-skinned through theme only) |
| Cart | done |
| Onboarding | done |
| Promotions, trace | done |
| Account tab, sign-in | done (imperial header, kit emblem, sentence-case titles) |
| Quote review | done (functional totals frame, one primary action) |
| Checkout operation, payment, order detail, support, returns, B2B, notifications | paper surface + themed controls; not yet reviewed on device |

Verification per change: `flutter analyze`, `flutter test`, a real-device screenshot compared with the web at 375px.
