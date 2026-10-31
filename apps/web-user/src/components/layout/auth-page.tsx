import Link from "next/link";
import { ReactNode } from "react";
import { SiteFooter } from "./site-footer";
import { SiteHeader } from "./site-header";

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
