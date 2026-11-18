"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import { BrandMark } from "./brand-mark";
import { COLLECTIONS } from "./collections";
import { HeaderDropdown } from "./header-dropdown";
import { CartIcon, SearchIcon } from "./icons";
import { MobileNav } from "./mobile-nav";
import { useAuth } from "@/components/auth/auth-provider";

function SearchForm({ id }: { id: string }) {
  return <form className="gh-search" action="/products" method="get" role="search">
    <label className="sr-only" htmlFor={id}>Tìm sản phẩm</label>
    <span className="gh-search__icon"><SearchIcon /></span>
    <input id={id} name="q" type="search" minLength={2} placeholder="Tìm mè xửng, trà sen…" />
  </form>;
}

export function SiteHeader() {
  const pathname = usePathname();
  const auth = useAuth();
  const { status, customer, principal, customerAccess, hasCapability } = useCustomerSession();
  const { itemCount } = useCartSummary();
  const authenticated = auth.isAuthenticated;
  const loading = auth.isLoading || status === "loading";
  const displayName = customer?.displayName ?? principal?.displayName ?? auth.user?.displayName ?? "Tài khoản";
  const initial = displayName.trim().charAt(0).toLocaleUpperCase("vi") || "T";
  const guestOrderHref = MVP_FEATURE_GATES.guestOrder ? "/track-order" : "/login?returnUrl=%2Faccount%2Forders";
  const showB2B = auth.isCustomer && hasCapability("B2B_QUOTE_VIEW");
  const current = (href: string) => (href === "/" ? pathname === "/" : pathname?.startsWith(href)) ? "page" : undefined;

  const accountMenu = authenticated
    ? <HeaderDropdown align="end" ariaLabel={`Mở menu tài khoản của ${displayName}`} triggerClassName="gh-account-trigger"
      label={<><span className="gh-avatar" aria-hidden="true">{loading ? "…" : initial}</span><span className="gh-account-trigger__name">{displayName}</span><span className="gh-chevron" aria-hidden="true">﹀</span></>}>
      <div className="gh-account-menu__who"><strong>{displayName}</strong>{auth.user?.email ? <small>{auth.user.email}</small> : null}</div>
      <ul className="gh-account-menu">
        {auth.isEmployee
          ? <li><Link href="/admin">Kênh quản trị</Link></li>
          : <>
            <li><Link href="/account/profile">Hồ sơ cá nhân</Link></li>
            <li><Link href={customerAccess ? "/account/orders" : guestOrderHref}>Đơn hàng của tôi</Link></li>
            {showB2B ? <li><Link href="/b2b">Báo giá sỉ B2B</Link></li> : null}
          </>}
        <li><Link href="/settings/password">Đổi mật khẩu</Link></li>
        <li><button type="button" onClick={() => void auth.logout()}>Đăng xuất</button></li>
      </ul>
    </HeaderDropdown>
    : <HeaderDropdown align="end" ariaLabel="Mở menu tài khoản" triggerClassName="gh-account-trigger gh-account-trigger--guest"
      label={<><span className="gh-avatar gh-avatar--guest gh-emblem" aria-hidden="true" /><span className="gh-account-trigger__name">Tài khoản</span><span className="gh-chevron" aria-hidden="true">﹀</span></>}>
      <div className="gh-account-menu__who"><strong>Chào mừng đến Gói Huế</strong><small>Đăng nhập để theo dõi đơn và nhận ưu đãi.</small></div>
      <div className="gh-account-menu__actions">
        <Link className="gh-btn gh-btn--brown" href="/login?returnUrl=%2Faccount%2Fprofile">Đăng nhập</Link>
        <Link className="gh-btn gh-btn--paper" href="/register">Đăng ký tài khoản</Link>
      </div>
      <ul className="gh-account-menu">
        <li><Link href={guestOrderHref}>{MVP_FEATURE_GATES.guestOrder ? "Tra cứu đơn hàng" : "Xem đơn hàng"}</Link></li>
        <li><Link href="/support/request">Hỗ trợ khách hàng</Link></li>
      </ul>
    </HeaderDropdown>;

  return (
    <>
      <header className="gh-header gh-surface-imperial">
        <div className="gh-wrap gh-header__main">
          <BrandMark />
          <nav className="gh-nav" aria-label="Điều hướng chính">
            <Link className="gh-nav-link" href="/" aria-current={current("/")}>Trang chủ</Link>
            <HeaderDropdown triggerClassName="gh-nav-link" panelClassName="gh-collections-panel"
              label={<>Bộ sưu tập <span className="gh-chevron" aria-hidden="true">﹀</span></>}>
              <ul>{COLLECTIONS.map((item) => <li key={item.href}><Link href={item.href}><strong>{item.name}</strong><small>{item.note}</small></Link></li>)}</ul>
              <Link className="gh-collections-panel__all" href="/products">Xem tất cả sản phẩm →</Link>
            </HeaderDropdown>
            <Link className="gh-nav-link" href="/#heritage-story">Chuyện Huế</Link>
            <Link className="gh-nav-link" href="/#ocop-traceability" aria-current={current("/trace")}>Truy xuất nguồn gốc</Link>
            <Link className="gh-nav-link" href="/promotions" aria-current={current("/promotions")}>Ưu đãi</Link>
            {showB2B ? <Link className="gh-nav-link" href="/b2b" aria-current={current("/b2b")}>B2B</Link> : null}
          </nav>
          <div className="gh-header__tools">
            <div className="gh-search-wide"><SearchForm id="header-search" /></div>
            <div className="gh-search-compact">
              <HeaderDropdown align="end" ariaLabel="Tìm sản phẩm" triggerClassName="gh-icon-button" panelClassName="gh-search-panel" label={<SearchIcon />}>
                <SearchForm id="header-search-compact" />
              </HeaderDropdown>
            </div>
            {!auth.isEmployee ? <Link className="gh-icon-button gh-cart" href="/cart" title="Giỏ hàng" aria-current={current("/cart")}
              aria-label={itemCount === null ? "Mở giỏ hàng" : `Mở giỏ hàng, ${itemCount} sản phẩm`}>
              <CartIcon />
              {itemCount ? <b className="gh-cart__count">{itemCount > 99 ? "99+" : itemCount}</b> : null}
            </Link> : null}
            {accountMenu}
          </div>
        </div>
      </header>
      <MobileNav itemCount={itemCount} showCart={!auth.isEmployee} />
    </>
  );
}
