"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useId, useRef, type KeyboardEvent } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import { useAuth } from "@/components/auth/auth-provider";
import { BrandMark } from "./brand-mark";
import { shopLinks, serviceLinks } from "./storefront-links";

const paths = {
  home: "m3 10 9-7 9 7v10h-6v-6H9v6H3z",
  menu: "M3 6h18M3 12h18M3 18h18",
  search: "M16 16l5 5M18 10a8 8 0 1 1-16 0 8 8 0 1 1 16 0",
  cart: "M2 3h3l3 12h11l3-9H6M10 20h.1M18 20h.1",
  user: "M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0M4 21v-2a8 8 0 0 1 16 0v2z",
  down: "m6 9 6 6 6-6"
};
function Icon({ name }: { name: keyof typeof paths }) {
  return <svg aria-hidden="true" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"><path d={paths[name]} /></svg>;
}

type NavLink = { name: string; href: string };
function Links({ links, close }: { links: NavLink[]; close?: () => void }) {
  return links.map(link => <Link href={link.href} key={link.href} onClick={close}>{link.name}</Link>);
}

function useAnchoredPopover() {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const panel = ref.current;
    const trigger = panel?.previousElementSibling;
    if (!panel || !trigger) return;
    function position() {
      if (!panel?.matches(":popover-open") || !trigger) return;
      const rect = trigger.getBoundingClientRect();
      panel.style.top = Math.round(rect.bottom + 4) + "px";
      panel.style.left = Math.max(8, Math.min(rect.left, document.documentElement.clientWidth - panel.offsetWidth - 8)) + "px";
    }
    panel.addEventListener("toggle", position);
    window.addEventListener("resize", position);
    window.addEventListener("scroll", position, { passive: true });
    return () => {
      panel.removeEventListener("toggle", position);
      window.removeEventListener("resize", position);
      window.removeEventListener("scroll", position);
    };
  });
  return ref;
}

function Dropdown({ label, links }: { label: string; links: NavLink[] }) {
  const id = useId();
  const ref = useAnchoredPopover();
  function move(event: KeyboardEvent<HTMLElement>) {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    event.preventDefault();
    ref.current?.showPopover();
    const items = Array.from(ref.current?.querySelectorAll("a") ?? []);
    const current = items.indexOf(document.activeElement as HTMLAnchorElement);
    const next = current < 0 ? (event.key === "ArrowDown" ? 0 : items.length - 1)
      : (current + (event.key === "ArrowDown" ? 1 : -1) + items.length) % items.length;
    items[next]?.focus();
  }
  return <div className="gh-group" onKeyDown={move}>
    <button popoverTarget={id}>{label}<Icon name="down" /></button>
    <div className="gh-popup" popover="auto" ref={ref} id={id}><Links links={links} close={() => ref.current?.hidePopover()} /></div>
  </div>;
}

function SearchForm() {
  return <form className="gh-search" action="/products" method="get" role="search">
    <input type="search" name="q" minLength={2} aria-label="Tìm sản phẩm" placeholder="Tìm sản phẩm…" />
    <button aria-label="Tìm kiếm"><Icon name="search" /></button>
  </form>;
}

function AccountLinks({ name, links, close, logout }: {
  name: string; links: NavLink[]; close: () => void; logout: () => void;
}) {
  return <>
    <strong>{name}</strong><Links links={[...links, { name: "Đổi mật khẩu", href: "/settings/password" }]} close={close} />
    <button onClick={() => { close(); logout(); }}>Đăng xuất</button>
  </>;
}

export function SiteHeader() {
  const auth = useAuth();
  const { customer, principal, customerAccess, hasCapability } = useCustomerSession();
  const { itemCount } = useCartSummary();
  const sheet = useRef<HTMLDialogElement>(null);
  const account = useAnchoredPopover();
  const path = usePathname();
  const displayName = customer?.displayName ?? principal?.displayName ?? auth.user?.displayName ?? "Tài khoản";
  const accountHref = auth.isEmployee ? "/admin" : auth.isAuthenticated ? "/account/profile" : "/login?returnUrl=%2Faccount%2Fprofile";
  const orders = auth.isAuthenticated && customerAccess ? "/account/orders" : MVP_FEATURE_GATES.guestOrder ? "/track-order" : "/login?returnUrl=%2Faccount%2Forders";
  const help = [...serviceLinks, auth.isEmployee ? { name: "Kênh quản trị", href: "/admin" } : { name: "Đơn hàng", href: orders }];
  const shopping = auth.isCustomer && hasCapability("B2B_QUOTE_VIEW") ? [...shopLinks, { name: "Báo giá sỉ", href: "/b2b" }] : shopLinks;
  const profile = auth.isEmployee ? [{ name: "Kênh quản trị", href: "/admin" }] : [{ name: "Hồ sơ cá nhân", href: accountHref }, { name: "Đơn hàng của tôi", href: "/account/orders" }];
  const close = () => sheet.current?.close();
  useEffect(() => {
    const query = window.matchMedia("(min-width: 768px)");
    const change = () => { if (query.matches) sheet.current?.close(); };
    query.addEventListener("change", change);
    return () => query.removeEventListener("change", change);
  }, []);
  const cart = <><Icon name="cart" />Giỏ hàng{itemCount !== null && itemCount > 0 && <b>{itemCount > 99 ? "99+" : itemCount}</b>}</>;
  return <>
    <header className="gh-header"><div className="gh-wrap gh-top">
      <BrandMark />
      <nav className="gh-desktop" aria-label="Điều hướng chính">
        <Dropdown label="Bộ sưu tập" links={shopping} />
        <Link href="/products?category=qua-bieu">Quà biếu</Link>
        <Dropdown label="Khám phá" links={help} />
      </nav>
      <SearchForm />
      <div className="gh-tools">
        {!auth.isEmployee && <Link href="/cart">{cart}</Link>}
        {auth.isAuthenticated ? <div className="gh-group"><button popoverTarget="gh-account" aria-label="Mở menu tài khoản"><Icon name="user" /></button>
          <div className="gh-popup gh-right" id="gh-account" ref={account} popover="auto"><AccountLinks name={displayName} links={profile} close={() => account.current?.hidePopover()} logout={auth.logout} /></div>
        </div> : <Link href={accountHref} aria-label="Đăng nhập"><Icon name="user" /></Link>}
      </div>
    </div></header>
    <nav className="gh-bottom" aria-label="Điều hướng di động">
      <Link href="/" aria-current={path === "/" ? "page" : undefined}><Icon name="home" />Trang chủ</Link>
      <button onClick={() => sheet.current?.showModal()} aria-haspopup="dialog"><Icon name="menu" />Danh mục</button>
      {!auth.isEmployee && <Link href="/cart" aria-current={path === "/cart" ? "page" : undefined}>{cart}</Link>}
      <Link href={accountHref} aria-current={path === accountHref ? "page" : undefined}><Icon name="user" />{auth.isEmployee ? "Quản trị" : "Tài khoản"}</Link>
    </nav>
    <dialog ref={sheet} className="gh-sheet" aria-labelledby="gh-sheet-title" onClick={event => { if (event.target === sheet.current) close(); }}>
      <div><header><h2 id="gh-sheet-title">Danh mục</h2><button aria-label="Đóng danh mục" onClick={close}>×</button></header>
        <nav aria-label="Danh mục và tiện ích"><strong>Bộ sưu tập</strong><Links links={shopping} close={close} />
          <strong>Khám phá</strong><Links links={help} close={close} />
          {auth.isAuthenticated && <AccountLinks name={displayName} links={profile} close={close} logout={auth.logout} />}
        </nav>
      </div>
    </dialog>
  </>;
}