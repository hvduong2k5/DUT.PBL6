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
        <nav aria-label="Điều hướng chính">
          <Link href="/">Trang Chủ</Link>
          <Link href="/#bo-suu-tap">Bánh &amp; Kẹo Cung Đình</Link>
          <Link href="/#tra-sen">Mè Xửng &amp; Trà Sen</Link>
          <Link href="/#qua-bieu">Quà Biếu Tặng</Link>
          <Link href="/#cau-chuyen">Câu Chuyện Huế</Link>
        </nav>
        <Link className="account-button" href="/account/profile" aria-label="Tài khoản Tri Kỷ">♙</Link>
      </div>
    </header>
  );
}
