---
name: Huế Heritage Confectionery
colors:
  surface: '#fcf9f4'
  surface-dim: '#dcdad5'
  surface-bright: '#fcf9f4'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f6f3ee'
  surface-container: '#f0ede9'
  surface-container-high: '#ebe8e3'
  surface-container-highest: '#e5e2dd'
  on-surface: '#1c1c19'
  on-surface-variant: '#50453e'
  inverse-surface: '#31302d'
  inverse-on-surface: '#f3f0eb'
  outline: '#82756d'
  outline-variant: '#d3c3ba'
  surface-tint: '#775841'
  primary: '#563a26'
  on-primary: '#ffffff'
  primary-container: '#70513b'
  on-primary-container: '#f0c6aa'
  inverse-primary: '#e8bea3'
  secondary: '#7e5700'
  on-secondary: '#ffffff'
  secondary-container: '#ffc96f'
  on-secondary-container: '#785300'
  tertiary: '#782210'
  on-tertiary: '#ffffff'
  tertiary-container: '#973925'
  on-tertiary-container: '#ffbfb1'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#ffdcc5'
  primary-fixed-dim: '#e8bea3'
  on-primary-fixed: '#2c1605'
  on-primary-fixed-variant: '#5d402c'
  secondary-fixed: '#ffdeac'
  secondary-fixed-dim: '#f3be65'
  on-secondary-fixed: '#281900'
  on-secondary-fixed-variant: '#604100'
  tertiary-fixed: '#ffdad3'
  tertiary-fixed-dim: '#ffb4a4'
  on-tertiary-fixed: '#3e0500'
  on-tertiary-fixed-variant: '#812816'
  background: '#fcf9f4'
  on-background: '#1c1c19'
  surface-variant: '#e5e2dd'
typography:
  display:
    fontFamily: Noto Serif
    fontSize: 48px
    fontWeight: '700'
    lineHeight: 56px
  display-mobile:
    fontFamily: Noto Serif
    fontSize: 32px
    fontWeight: '700'
    lineHeight: 40px
  headline-lg:
    fontFamily: Noto Serif
    fontSize: 36px
    fontWeight: '600'
    lineHeight: 44px
  headline-lg-mobile:
    fontFamily: Noto Serif
    fontSize: 26px
    fontWeight: '600'
    lineHeight: 34px
  headline-md:
    fontFamily: Noto Serif
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
  headline-sm:
    fontFamily: Noto Serif
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
  title:
    fontFamily: Noto Serif
    fontSize: 18px
    fontWeight: '500'
    lineHeight: 26px
  body-lg:
    fontFamily: Be Vietnam Pro
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-md:
    fontFamily: Be Vietnam Pro
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 22px
  body-sm:
    fontFamily: Be Vietnam Pro
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 18px
  label-lg:
    fontFamily: Be Vietnam Pro
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
  label-md:
    fontFamily: Be Vietnam Pro
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
  label-sm:
    fontFamily: Be Vietnam Pro
    fontSize: 10px
    fontWeight: '600'
    lineHeight: 14px
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  gutter: 1.5rem
  gutter-mobile: 1rem
  margin: 3rem
  margin-mobile: 1rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 1rem
  space-lg: 1.5rem
  space-xl: 2rem
---

## Brand & Style

This design system expresses the poetic nobility, architectural symmetry, and warmth of ancient Huế imperial tea-and-confectionery culture, paired with contemporary digital commerce usability. The experience evokes the quiet indulgence of tasting handcrafted sesame candies (mè xửng), ginger lotus seed paste, and imperial roasted pork sweet soup within a serene garden manor. 

Visual design is rooted in **Warm Editorial Elegance**:
- **Tonal Organic Harmony:** Grounded entirely in mineral and organic tones—lacquer wood, warm honey, terracotta brick, and unbleached silk rice paper.
- **Poetic Precision:** Clear typographical hierarchy marrying royal editorial serifs with crisp, accessible humanist sans-serifs tailored natively for the Vietnamese diacritical writing system.
- **Tactile Softness:** Subtly rounded geometry, paper-like matte surfaces, and gentle warm ambient shadows that emulate sunlight through wooden louvers, rejecting stark electronic whites and cold grey drop shadows.

## Colors

The palette directly honors traditional Vietnamese crafts: roasted malt, woven bamboo, aged temple tiles, and imperial brass.

### Palette Roles
- **Brand Brown (`#70513B`):** The primary identity color. Governs brand presence, primary section headers, primary interactive states, and solid authoritative containers.
- **Honey Gold (`#D4A24C`):** The secondary highlight tone. Drives high-conversion call-to-actions, badges, rating stars, loyalty elements, and royal seals.
- **Brick Red (`#B8513B`):** The tertiary accent tone. Applied purposefully for promotional labels, product discounts, live stock alerts, and primary checkout purchase flows.
- **Ivory Cream (`#FAF7F2`):** The master canvas foundation. Replaces synthetic white to yield an organic, warm paper texture that minimizes eye fatigue.

### Supporting & Neutral Tokens
- **Surface Soft (`#F5EFE8`):** Elevated cards, input field fills, alternating table row backgrounds, and subtle secondary containers.
- **Surface Crisp (`#FFFFFF`):** High-contrast product showcase cards, interactive dropdown modals, and active cart drawers.
- **Text Primary (`#2F241C`):** Deep roasted espresso tint ensuring high-contrast readability (AAA) for body copy, microcopy, and pricing decimals.
- **Text Secondary (`#6B5A4A`):** Earthy mid-tone for product subtitles, metadata, breadcrumbs, and disabled states.
- **Border Natural (`#E7DED4`):** Soft, structural separator line tint mimicking unbleached linen fiber.

## Typography

The typographical pairing bridges imperial heritage and functional digital commerce. 

- **Headlines (`Noto Serif`):** Conveys traditional authority and literary craftsmanship. Carefully balanced optical sizing preserves grace across long multi-word Vietnamese phrases with tonal markers.
- **Interface & Body (`Be Vietnam Pro`):** Purpose-designed for Vietnamese diacritics, preventing accent collision and baseline shifts across critical e-commerce touchpoints, inventory tables, and numeric price tables.

### Rules & Guidelines
- Headings use optical sizing with relaxed line heights to accommodate stacked tone accents (`ấ`, `ễ`, `ở`, `ự`).
- E-commerce currency displays use `Be Vietnam Pro` with tabular lining figures (`font-variant-numeric: tabular-nums`) to prevent jitter across quantity recalculations and totals.
- All uppercase button labels maintain slight tracking (`letter-spacing: +0.02em`) to ensure legibility when set against rich background colors.

## Layout & Spacing

The design system operates on a base 4px metric system (`4px`, `8px`, `12px`, `16px`, `20px`, `24px`, `32px`, `40px`, `48px`, `64px`, `96px`).

### Responsive Layout Strategy
- **Web D2C Desktop (1200px+):** 12-column fluid grid, max-width `1280px` centered, `24px` gutter, `48px` outer page margins. Product catalog layouts favor generous white space and rhythmic asymmetry to celebrate product artistry.
- **Web Admin Portal (1024px+):** 12-column fixed sidebar layout with high-density spacing. Fluid workspace utilizing `16px` gutters and `24px` page margins.
- **Flutter Mobile & Web Mobile (<768px):** 4-column layout, dynamic safe-area gutters (`16px`), and `16px` outer margins. Single column vertical flow for checkout and product detail views; 2-column modular grid for product listing cards.

### Spacing Application Rules
- **`space-xs` (4px):** Form icon-to-label gaps, badge internal vertical padding.
- **`space-sm` (8px):** Chip internal padding, horizontal icon pairings, list item micro-spacing.
- **`space-md` (16px):** Standard card padding, form input heights, item grid gaps.
- **`space-lg` (24px):** Desktop card internal padding, layout card separation, modal dialog padding.
- **`space-xl` (32px):** Page section intervals on mobile, administrative card group spacing.

## Elevation & Depth

This design system replaces synthetic cold shadows with warm, filtered light and tonal surfaces, evoking paper-over-wood architecture:

- **Level 0 (Flat / Canvas):** Ivory Cream (`#FAF7F2`) base surface. No elevation.
- **Level 1 (Card & Sub-container):** Surface Crisp (`#FFFFFF`) or Soft Surface (`#F5EFE8`) paired with a single pixel border in Border Natural (`#E7DED4`). No shadow on mobile; optional ambient diffusion on desktop: `0px 2px 8px rgba(112, 81, 59, 0.04)`.
- **Level 2 (Hover & Raised Panels):** Used for product card interactions, sticky navigation bars, and cart summaries. Elevation: `0px 8px 24px rgba(112, 81, 59, 0.08)`.
- **Level 3 (Modals, Overlays & Drawers):** Used for quick-view overlays, mobile checkout sheets, and filter drawers. Elevation: `0px 16px 40px rgba(47, 36, 28, 0.14)`. Backdrop scrim is tinted at `rgba(47, 36, 28, 0.4)` with a `4px` blur.

## Shapes

The shape vocabulary balances structured heritage geometry with modern mobile ergonomics.

- **Small Corner (`8px`):** Input fields, inline status chips, price tags, search bars, and table cell buttons.
- **Medium Corner (`12px`):** Standard product cards, modal containers, notification banners, and primary CTA buttons.
- **Large Corner (`16px`):** Featured editorial hero cards, review showcase cards, and web admin metric panels.
- **Extra Large Corner (`20px`):** Mobile bottom action sheets, sliding cart drawers, and promotional gift sets.
- **Full / Circular:** Notification badges, profile avatars, floating quantity toggles, and icon-only tooltips.

## Components

### Buttons
- **Primary CTA:** Background Brand Brown (`#70513B`), Text White (`#FFFFFF`). Hover state transitions to Honey Gold (`#D4A24C`). Corner radius `12px`. Height: `44px` on desktop, `48px` on mobile for ergonomic touch targets.
- **Urgent / Buy Now:** Background Brick Red (`#B8513B`), Text White (`#FFFFFF`). Hover brightness boosted by 5%. Used strictly for flash sales and instant checkout.
- **Secondary / Ghost:** Transparent background, `1px` border in Border Natural (`#E7DED4`), text in Brand Brown (`#70513B`). Hover background shifts to Soft Surface (`#F5EFE8`).

### Input Fields & Controls
- **Text Inputs:** Height `48px`. Background Soft Surface (`#F5EFE8`), border `1px` solid Border Natural (`#E7DED4`). Active focus shifts border to Brand Brown (`#70513B`) with a `2px` focus ring in `rgba(112, 81, 59, 0.15)`.
- **Checkboxes & Radios:** Size `20px`. Border in Brand Brown (`#70513B`). Checked state uses Brand Brown background with a White glyph.
- **Quantity Selector:** Stepper with a combined pill container. `+` and `-` buttons in Soft Surface (`#F5EFE8`) flanking tabular-numeric central text in roasted primary typography (`#2F241C`).

### Cards & Merchandising
- **Product Listing Card:** White background, `1px` border in Border Natural (`#E7DED4`), `12px` radius. Image container features a subtle 1:1 square ratio with an Ivory Cream background. Price display leads with Brick Red (`#B8513B`) for discounted prices, accompanied by secondary strike-through text in Secondary Text (`#6B5A4A`).
- **Hue Specialty Badge (Chip):** Small pills (`8px` radius) with Honey Gold background (`#D4A24C`), text White (`#FFFFFF`), or bordered outline version for product categories (e.g., "Tiến Vua", "Truyền Thống").

### Lists & Tables (Web Admin)
- Row striping alternated between White (`#FFFFFF`) and Soft Surface (`#F5EFE8`).
- Header labels in `Be Vietnam Pro` Bold (`label-md`), Text Secondary (`#6B5A4A`), bottom border `1px` in Border Natural (`#E7DED4`).
- Interactive action icons color-coded: Edit in Brand Brown, Delete in Brick Red, Details in Honey Gold.