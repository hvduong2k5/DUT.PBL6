"use client";

import Link from "next/link";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { BrandMark } from "./brand-mark";

export function SiteHeader() {
  const { status, customer, hasCapability } = useCustomerSession();
  const { itemCount } = useCartSummary();
  const authenticated = status === "authenticated" && customer;
  const initial = customer?.displayName.trim().charAt(0).toLocaleUpperCase("vi") || "♙";
  return (
    <header className="site-header">
      <div className="utility-bar">
        <span>🚚 Giao hàng chuẩn quốc bảo toàn quốc</span>
        <span>✺ Di sản Ẩm thực Cung Đình Huế</span>
        <span className="utility-push">⌖ Cửa hàng Cố Đô: 128 Lê Lợi, TP. Huế</span>
        <strong>☎ 1900 68 Hue</strong>
      </div>
      <div className="main-nav">
        <BrandMark />
        <form className="header-search" action="/products" method="get" role="search">
          <label className="sr-only" htmlFor="header-search">Tìm sản phẩm</label>
          <input id="header-search" name="q" minLength={2} placeholder="Tìm mè xửng, hạt sen, trà…" />
          <button type="submit" aria-label="Tìm kiếm">⌕</button>
        </form>
        <nav aria-label="Điều hướng chính">
          <Link href="/">Trang Chủ</Link>
          <Link href="/products?category=banh-cung-dinh">Bánh &amp; Kẹo Cung Đình</Link>
          <Link href="/products?category=me-xung-keo-hue">Mè Xửng &amp; Trà Sen</Link>
          <Link href="/products?category=qua-bieu">Quà Biếu Tặng</Link>
          {hasCapability("B2B_QUOTE_VIEW") ? <Link href="/b2b">Báo Giá Sỉ B2B</Link> : null}
          <Link href="/#heritage-story">Câu Chuyện Huế</Link>
          <Link href="/#ocop-traceability">Truy Xuất OCOP</Link>
          <Link href="/promotions">Ưu Đãi Tri Kỷ</Link>
        </nav>
        <Link className="track-order-link" href="/track-order">Tra cứu đơn</Link>
        <Link className="cart-header-button" href="/cart" aria-label={itemCount === null ? "Mở giỏ hàng" : `Mở giỏ hàng, ${itemCount} sản phẩm`}>♧<span>Giỏ hàng</span>{itemCount !== null ? <b>{itemCount > 99 ? "99+" : itemCount}</b> : null}</Link>
        <Link className={`account-button ${authenticated ? "authenticated" : ""}`} href={authenticated ? "/account/profile" : "/login?returnUrl=%2Faccount%2Fprofile"} aria-label={authenticated ? `Tài khoản của ${customer.displayName}` : "Đăng nhập tài khoản Tri Kỷ"} title={authenticated ? customer.displayName : "Đăng nhập"}>{status === "loading" ? "…" : initial}</Link>
      </div>
    </header>
  );
}
