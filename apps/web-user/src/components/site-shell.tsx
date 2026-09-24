import Link from "next/link";
import { ReactNode } from "react";

export function BrandMark() {
  return (
    <Link className="brand" href="/" aria-label="Ô Mạ - Trang chủ">
      <span className="brand-seal" aria-hidden="true">Ô</span>
      <span>
        <strong>Ô MẠ</strong>
        <small>Tinh hoa bánh mứt Cố Đô</small>
      </span>
    </Link>
  );
}

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
        <Link className="account-button" href="/dang-nhap" aria-label="Tài khoản Tri Kỷ">♙</Link>
      </div>
    </header>
  );
}

export function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="footer-grid">
        <div>
          <BrandMark />
          <p>Gìn giữ và tôn vinh phong vị bánh mứt yến tiệc hoàng triều Cố Đô, chuẩn hóa theo tinh thần OCOP đương đại.</p>
          <span className="trust-chip">✺ OCOP 4 Sao Thừa Thiên Huế</span>
        </div>
        <div>
          <h2>Bộ Sưu Tập</h2>
          <p>Mè Xửng Dẻo Ngự Thiện</p><p>Bánh Hạt Sen Cung Đình</p><p>Trà Ướp Hoa Sen Tịnh Tâm</p>
        </div>
        <div>
          <h2>Chăm Sóc &amp; Hỗ Trợ</h2>
          <p>Hướng dẫn đặt hàng</p><p>Chính sách vận chuyển</p><p>Cam kết nguồn gốc</p>
        </div>
        <div>
          <h2>Phòng Trưng Bày &amp; Xưởng Bánh</h2>
          <p>128 Lê Lợi, Phường Phú Nhuận, Thành phố Huế</p>
          <p>(0234) 388 9988 · thuquan@omahue.vn</p>
        </div>
      </div>
      <div className="footer-bottom">© 2026 Ô Mạ · Tinh tuyển Di sản · Chuẩn hóa OCOP</div>
    </footer>
  );
}

interface AuthPageProps {
  eyebrow: string;
  title: string;
  description: string;
  asideTitle: string;
  asideBody: string;
  benefits: Array<{ icon: string; title: string; body: string }>;
  children: ReactNode;
}

export function AuthPage({ eyebrow, title, description, asideTitle, asideBody, benefits, children }: AuthPageProps) {
  return (
    <>
      <SiteHeader />
      <main className="auth-page">
        <div className="breadcrumbs"><Link href="/">Trang chủ</Link><span>/</span><span>Tài khoản Tri Kỷ</span></div>
        <section className="auth-frame">
          <aside className="auth-story">
            <span className="story-badge">✺ Đặc quyền Cố Đô</span>
            <p className="story-kicker">{asideTitle}</p>
            <h2>Nơi lưu giữ phong vị ngự trà &amp; bánh mứt di sản</h2>
            <p>{asideBody}</p>
            <div className="benefit-list">
              {benefits.map((benefit) => (
                <article key={benefit.title}>
                  <span aria-hidden="true">{benefit.icon}</span>
                  <div><h3>{benefit.title}</h3><p>{benefit.body}</p></div>
                </article>
              ))}
            </div>
            <div className="story-signature"><span>✺ Chứng nhận OCOP Cố Đô</span><em>Ô Mạ · 1885</em></div>
          </aside>
          <div className="auth-content">
            <span className="eyebrow">{eyebrow}</span>
            <h1>{title}</h1>
            <p className="lead">{description}</p>
            {children}
          </div>
        </section>
        <section className="trust-row" aria-label="Cam kết dịch vụ">
          <article><span>☎</span><div><h2>Cố Vấn Ẩm Thực Cung Đình</h2><p>Tư vấn chọn thức quà chuẩn vị.</p></div></article>
          <article><span>✺</span><div><h2>Chứng Thư Hoàng Tộc OCOP</h2><p>Tự hào đặc sản đạt chuẩn xứ Huế.</p></div></article>
          <article><span>🚚</span><div><h2>Thưởng Thức Nhanh Toàn Quốc</h2><p>Bọc lụa chống va đập, bảo toàn hương hoa.</p></div></article>
        </section>
      </main>
      <SiteFooter />
    </>
  );
}
