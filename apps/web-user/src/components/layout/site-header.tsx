import Link from "next/link";
import { BrandMark } from "./brand-mark";

export function SiteHeader() {
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
          <Link href="/products?category=banh-cung-dinh">Bánh Cung Đình</Link>
          <Link href="/products?category=me-xung-keo-hue">Mè Xửng</Link>
          <Link href="/products?category=qua-bieu">Quà Biếu</Link>
        </nav>
        <Link className="cart-header-button" href="/cart" aria-label="Mở giỏ hàng">♧<span>Giỏ hàng</span></Link>
        <Link className="account-button" href="/account/profile" aria-label="Tài khoản Tri Kỷ">♙</Link>
      </div>
    </header>
  );
}
