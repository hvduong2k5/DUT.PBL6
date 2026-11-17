"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import { BrandMark } from "./brand-mark";
import { COLLECTIONS } from "./collections";
import { MobileNav } from "./mobile-nav";
import { useAuth } from "@/components/auth/auth-provider";

function CollectionsMenu() {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onPointer = (event: PointerEvent) => { if (!root.current?.contains(event.target as Node)) setOpen(false); };
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") setOpen(false); };
    document.addEventListener("pointerdown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("pointerdown", onPointer); document.removeEventListener("keydown", onKey); };
  }, [open]);

  return <div className="gh-menu" ref={root}>
    <button type="button" className="gh-nav-link gh-menu__trigger" aria-expanded={open} aria-controls="gh-collections-panel" onClick={() => setOpen((value) => !value)}>
      Bộ sưu tập <span aria-hidden="true">﹀</span>
    </button>
    {open ? <div id="gh-collections-panel" className="gh-menu__panel" onClick={() => setOpen(false)}>
      <ul>
        {COLLECTIONS.map((item) => <li key={item.href}><Link href={item.href}><strong>{item.name}</strong><small>{item.note}</small></Link></li>)}
      </ul>
      <Link className="gh-menu__all" href="/products">Xem tất cả sản phẩm →</Link>
    </div> : null}
  </div>;
}

export function SiteHeader() {
  const auth = useAuth();
  const { status, customer, principal, customerAccess, hasCapability } = useCustomerSession();
  const { itemCount } = useCartSummary();
  const authenticated = auth.isAuthenticated;
  const displayName = customer?.displayName ?? principal?.displayName ?? auth.user?.displayName ?? "Tài khoản";
  const initial = authenticated ? displayName.trim().charAt(0).toLocaleUpperCase("vi") || "T" : "♙";
  const ordersHref = authenticated && customerAccess ? "/account/orders" : MVP_FEATURE_GATES.guestOrder ? "/track-order" : "/login?returnUrl=%2Faccount%2Forders";
  const ordersLabel = authenticated && customerAccess ? "Đơn hàng" : MVP_FEATURE_GATES.guestOrder ? "Tra cứu đơn" : "Đăng nhập xem đơn";
  const accountHref = authenticated ? (auth.isEmployee ? "/admin" : "/account/profile") : "/login?returnUrl=%2Faccount%2Fprofile";

  return (
    <>
      <header className="gh-header">
        <div className="gh-utility">
          <div className="gh-wrap gh-utility__inner">
            <span>🚚 Giao hàng toàn quốc</span>
            <span>✺ Đặc sản &amp; quà biếu xứ Huế</span>
            <span className="gh-utility__push">⌖ 128 Lê Lợi, TP. Huế</span>
            <strong>☎ 1900 68 Hue</strong>
          </div>
        </div>
        <div className="gh-wrap gh-header__main">
          <BrandMark />
          <nav className="gh-nav" aria-label="Điều hướng chính">
            <Link className="gh-nav-link" href="/">Trang chủ</Link>
            <CollectionsMenu />
            {auth.isCustomer && hasCapability("B2B_QUOTE_VIEW") ? <Link className="gh-nav-link" href="/b2b">Báo giá sỉ B2B</Link> : null}
            <Link className="gh-nav-link" href="/#heritage-story">Chuyện Huế</Link>
            <Link className="gh-nav-link" href="/#ocop-traceability">Truy xuất nguồn gốc</Link>
            <Link className="gh-nav-link" href="/promotions">Ưu đãi</Link>
          </nav>
          <div className="gh-header__tools">
            <form className="gh-search" action="/products" method="get" role="search">
              <label className="sr-only" htmlFor="header-search">Tìm sản phẩm</label>
              <svg className="gh-search__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m16 16 4.5 4.5" strokeLinecap="round" /></svg>
              <input id="header-search" name="q" minLength={2} placeholder="Tìm sản phẩm…" />
            </form>
            {auth.isEmployee
              ? <Link className="gh-track-link" href="/admin">🛠 Kênh quản trị</Link>
              : <Link className="gh-track-link" href={ordersHref}>{ordersLabel}</Link>}
            {!auth.isEmployee ? <Link className="gh-cart-button" href="/cart" aria-label={itemCount === null ? "Mở giỏ hàng" : `Mở giỏ hàng, ${itemCount} sản phẩm`}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true"><path d="M4 8h16l-1.2 12H5.2z" /><path d="M8 8V6a4 4 0 0 1 8 0v2" /></svg>
              <span>Giỏ hàng</span>
              {itemCount !== null ? <b>{itemCount > 99 ? "99+" : itemCount}</b> : null}
            </Link> : null}
            {authenticated
              ? <details className="gh-user">
                <summary className="gh-account gh-account--active" aria-label={`Mở menu tài khoản của ${displayName}`} title={displayName}>{auth.isLoading || status === "loading" ? "…" : initial}</summary>
                <div className="gh-user__menu">
                  <strong>{displayName}</strong><small>{auth.user?.email}</small>
                  {auth.isEmployee ? <Link href="/admin">Kênh quản trị</Link> : <><Link href="/account/profile">Hồ sơ cá nhân</Link><Link href="/account/orders">Đơn hàng của tôi</Link></>}
                  <Link href="/settings/password">Đổi mật khẩu</Link>
                  <button type="button" onClick={() => void auth.logout()}>Đăng xuất</button>
                </div>
              </details>
              : <Link className="gh-account" href="/login?returnUrl=%2Faccount%2Fprofile" aria-label="Đăng nhập tài khoản" title="Đăng nhập">{auth.isLoading || status === "loading" ? "…" : initial}</Link>}
          </div>
        </div>
      </header>
      <MobileNav
        itemCount={itemCount}
        showCart={!auth.isEmployee}
        accountHref={accountHref}
        accountLabel={authenticated ? "Tài khoản" : "Đăng nhập"}
      />
    </>
  );
}
