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
import { useStorefrontView } from "@/components/storefront/view-mode";
import { CartPreview } from "@/components/storefront/cart-preview";
import { ItemDialog } from "@/components/storefront/item-dialog";
import { items, type HomeItem } from "@/components/storefront/items";
import { collections, shopLinks, serviceLinks } from "./storefront-links";

function normalize(text: string) {
  return text
    .normalize("NFD")
    .replace(/\p{M}/gu, "")
    .toLowerCase()
    .replaceAll("đ", "d")
    .trim();
}

export function SiteHeader() {
  const { paper, compact, setPaper, setCompact, setGroup } =
    useStorefrontView();
  const path = usePathname();
  const router = useRouter();
  const { itemCount } = useCartSummary();
  const auth = useAuth();
  const { customerAccess, hasCapability } = useCustomerSession();
  const [panel, setPanel] = useState<
    "categories" | "search" | "cart" | "account" | null
  >(null);
  const [popup, setPopup] = useState<"menu" | "suggest" | null>(null);
  const [detail, setDetail] = useState<HomeItem | null>(null);
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
    setGroup(group);
    setPopup(null);
    setPanel(null);
    if (path === "/") document.getElementById("shop")?.scrollIntoView();
    else router.push("/#shop");
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
  const matches = items.filter((item) =>
    normalize(item.name).includes(normalize(query)),
  );
  const suggestions = (
    <div className="suggestions">
      {matches.map((item) => (
        <button
          className="suggestion"
          key={item.slug}
          onClick={() => {
            setPanel(null);
            setPopup(null);
            setDetail(item);
          }}
          type="button"
        >
          {item.name}
        </button>
      ))}
      {!matches.length && <p>Chưa tìm thấy thức quà phù hợp.</p>}
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
            {path === "/" && (
              <>
                <button
                  className="pill-switch"
                  aria-pressed={paper}
                  onClick={() => setPaper(!paper)}
                >
                  ❖ Kiểu Hero:{" "}
                  {paper
                    ? "Khung Giấy Dó Kiến Trúc"
                    : "Khung Nâu Trầm & Bàn Trà"}
                </button>
                <button
                  className="pill-switch"
                  aria-pressed={compact}
                  onClick={() => {
                    setCompact(!compact);
                    setPopup(null);
                  }}
                >
                  Chế độ xem: {compact ? "Mobile (390px)" : "Web Desktop"}
                </button>
              </>
            )}
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
              Bộ sưu tập ﹀
            </button>
            {collections.map((c, i) => (
              <button key={c.name} className="nav-link" onClick={() => pick(i)}>
                {c.name}
              </button>
            ))}
            <Link className="nav-link" href="/#gift">
              Gói quà
            </Link>
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
      {detail && <ItemDialog item={detail} close={() => setDetail(null)} />}
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
