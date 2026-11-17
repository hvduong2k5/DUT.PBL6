"use client";

import Link from "next/link";
import { BrandMark } from "./brand-mark";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { useAuth } from "@/components/auth/auth-provider";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import { GroupOptions } from "@/components/storefront/group-options";
import { StorefrontDialog } from "@/components/storefront/dialog";
import { MobileTabs } from "@/components/storefront/mobile-tabs";

import { CartPreview } from "@/components/storefront/cart-preview";
import { catalogService } from "@/services/catalog-service";
import type { ProductSummary } from "@/lib/catalog/types";
import { collections, shopLinks, serviceLinks } from "./storefront-links";

export function SiteHeader() {
  const path = usePathname();
  const router = useRouter();
  const { itemCount } = useCartSummary();
  const auth = useAuth();
  const { customerAccess, hasCapability } = useCustomerSession();
  const [panel, setPanel] = useState<
    "categories" | "search" | "cart" | "account" | null
  >(null);
  const [popup, setPopup] = useState<"menu" | "suggest" | null>(null);
  const [query, setQuery] = useState("");
  const header = useRef<HTMLElement>(null);
  const menuButton = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (!popup) return;
    const cancel = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setPopup(null);
        if (popup === "menu") menuButton.current?.focus();
      }
    };
    const outside = (e: PointerEvent) => {
      if (!header.current?.contains(e.target as Node)) setPopup(null);
    };
    document.addEventListener("keydown", cancel);
    document.addEventListener("pointerdown", outside);
    const mq = window.matchMedia("(max-width: 767px)");
    const resize = () => setPopup(null);
    mq.addEventListener("change", resize);
    return () => {
      document.removeEventListener("keydown", cancel);
      document.removeEventListener("pointerdown", outside);
      mq.removeEventListener("change", resize);
    };
  }, [popup]);
  function pick(group: number) {
    setPopup(null);
    setPanel(null);
    router.push(group < 0 ? "/products" : collections[group].href);
  }
  function open(value: typeof panel) {
    setPopup(null);
    setPanel(value);
  }
  const categories = (
    <>
      <div className="between">
        <strong>Chọn Thức Quà Xứ Huế</strong>
        <button className="chip-btn" onClick={() => pick(-1)}>
          Xem tất cả →
        </button>
      </div>
      <div className="mega-dropdown__grid">
        <GroupOptions choose={pick} menu />
      </div>
    </>
  );
  const [matches, setMatches] = useState<ProductSummary[]>([]);
  const [searchState, setSearchState] = useState<
    "idle" | "loading" | "ready" | "error"
  >("idle");
  useEffect(() => {
    if (query.trim().length < 2) return;
    let active = true;
    const timer = setTimeout(() => {
      setSearchState("loading");
      catalogService
        .getProducts(new URLSearchParams({ q: query.trim(), pageSize: "5" }))
        .then((data) => {
          if (active) {
            setMatches(data.items);
            setSearchState("ready");
          }
        })
        .catch(() => {
          if (active) {
            setMatches([]);
            setSearchState("error");
          }
        });
    }, 250);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [query]);
  const suggestions = (
    <div className="suggestions">
      {(query.trim().length >= 2 && searchState === "ready" ? matches : []).map(
        (item) => (
          <button
            className="suggestion"
            key={item.slug}
            onClick={() => {
              setPanel(null);
              setPopup(null);
              router.push("/products/" + item.slug);
            }}
            type="button"
          >
            {item.name}
          </button>
        ),
      )}
      {query.trim().length < 2 ? (
        <p>Nhập ít nhất 2 ký tự để tìm thức quà.</p>
      ) : searchState === "loading" ? (
        <p role="status">Đang tìm sản phẩm…</p>
      ) : searchState === "error" ? (
        <p>Chưa tải được gợi ý. Bấm tìm kiếm để thử lại.</p>
      ) : searchState === "ready" && !matches.length ? (
        <p>Chưa tìm thấy thức quà phù hợp.</p>
      ) : null}
      <Link
        href={"/products?q=" + encodeURIComponent(query)}
        onClick={() => {
          setPanel(null);
          setPopup(null);
        }}
      >
        Xem trong danh mục →
      </Link>
    </div>
  );
  const search = (mobile: boolean) => (
    <form
      action="/products"
      role="search"
      className={mobile ? "sheet-search" : "hue-header-search"}
    >
      <svg
        className="header-search__icon"
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        aria-hidden="true"
      >
        <circle cx="11" cy="11" r="7" />
        <path d="m16 16 4.5 4.5" />
      </svg>
      <input
        className="header-search__input"
        type="search"
        name="q"
        autoComplete="off"
        aria-label="Tìm sản phẩm đặc sản Huế"
        placeholder="Tìm sản phẩm..."
        value={query}
        onChange={(e) => {
          setQuery(e.target.value);
          if (!mobile) setPopup("suggest");
        }}
        onFocus={() => {
          if (!mobile) setPopup("suggest");
        }}
      />
      {!mobile && popup === "suggest" && (
        <div className="search-suggest-panel">{suggestions}</div>
      )}
    </form>
  );
  const orders =
    auth.isAuthenticated && customerAccess
      ? "/account/orders"
      : MVP_FEATURE_GATES.guestOrder
        ? "/track-order"
        : "/login?returnUrl=%2Faccount%2Forders";
  return (
    <div className="hue-surface hue-navigation">
      <div className="top-bar">
        <div className="wrap top-bar__inner">
          <span>
            ✦ <strong>GÓI HUẾ</strong> · Đặc sản Mè xửng, Trà cung đình &amp;
            Quà biếu Cố đô
          </span>
          <div className="top-bar__controls">
            <button className="pill-switch" onClick={() => open("account")}>
              Tài khoản
            </button>
          </div>
        </div>
      </div>
      <header ref={header} className="hue-site-header">
        <div className="wrap site-header__inner">
          <BrandMark />
          <nav className="hue-main-nav" aria-label="Điều hướng chính">
            <button
              className="nav-trigger"
              ref={menuButton}
              aria-expanded={popup === "menu"}
              aria-controls="megaDropdown"
              onClick={() => setPopup(popup === "menu" ? null : "menu")}
            >
              Sản phẩm ﹀
            </button>
            {collections.map((c, i) => (
              <button key={c.name} className="nav-link" onClick={() => pick(i)}>
                {c.name}
              </button>
            ))}
            <Link className="nav-link" href="/#story">
              Chuyện Huế
            </Link>
          </nav>
          <div className="header-tools">
            {search(false)}
            <button
              className="btn-icon"
              onClick={() => open("search")}
              aria-label="Tìm kiếm thức quà"
            >
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
              >
                <circle cx="11" cy="11" r="7" />
                <path d="m16 16 4.5 4.5" />
              </svg>
            </button>
            {!auth.isEmployee && (
              <button className="cart-trigger" onClick={() => open("cart")}>
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.8"
                >
                  <path d="M4 8h16l-1.2 12H5.2L4 8zM8 8V6a4 4 0 0 1 8 0v2" />
                </svg>
                Giỏ quà
                {itemCount !== null && (
                  <b className="cart-badge">{itemCount}</b>
                )}
              </button>
            )}
          </div>
          {popup === "menu" && (
            <div id="megaDropdown" className="mega-dropdown">
              {categories}
            </div>
          )}
        </div>
      </header>
      <MobileTabs
        onMenu={() => open("categories")}
        onSearch={() => open("search")}
        onCart={() => open("cart")}
        count={itemCount}
        employee={auth.isEmployee}
        onHome={path === "/"}
      />
      {panel && (
        <StorefrontDialog
          close={() => setPanel(null)}
          title={
            {
              categories: "Chọn thức quà",
              search: "Tìm kiếm thức quà",
              cart: "Giỏ quà của bạn",
              account: "Tài khoản và tiện ích",
            }[panel]
          }
        >
          {panel === "categories" && categories}
          {panel === "search" && (
            <>
              {search(true)}
              {suggestions}
            </>
          )}
          {panel === "cart" && <CartPreview close={() => setPanel(null)} />}
          {panel === "account" && (
            <nav className="account-links" onClick={() => setPanel(null)}>
              <Link
                href={
                  auth.isEmployee
                    ? "/admin"
                    : auth.isAuthenticated
                      ? "/account/profile"
                      : "/login?returnUrl=%2Faccount%2Fprofile"
                }
              >
                {auth.isEmployee
                  ? "Kênh quản trị"
                  : auth.isAuthenticated
                    ? "Hồ sơ cá nhân"
                    : "Đăng nhập"}
              </Link>
              {!auth.isEmployee && <Link href={orders}>Đơn hàng</Link>}
              {auth.isCustomer && hasCapability("B2B_QUOTE_VIEW") && (
                <Link href="/b2b">Báo giá sỉ</Link>
              )}
              {[...shopLinks.slice(4), ...serviceLinks].map((l) => (
                <Link key={l.href} href={l.href}>
                  {l.name}
                </Link>
              ))}
              {auth.isAuthenticated && (
                <>
                  <Link href="/settings/password">Đổi mật khẩu</Link>
                  <button
                    className="chip-btn"
                    onClick={() => void auth.logout()}
                  >
                    Đăng xuất
                  </button>
                </>
              )}
            </nav>
          )}
        </StorefrontDialog>
      )}
    </div>
  );
}
