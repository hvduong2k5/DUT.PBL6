"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { Motif } from "@/components/heritage/motif";
import { ProductImage } from "@/components/product/product-image";
import { RecommendationPreferences, RecommendationRail } from "@/components/recommendation/recommendation-rail";

const categories = [
  { href: "/products?category=me-xung-keo-hue", name: "Mè xửng Huế", action: "Xem ngay", image: "/images/heritage/photos/cat-me-xung.jpg" },
  { href: "/products?category=tra-cung-dinh", name: "Trà Huế", action: "Xem ngay", image: "/images/heritage/photos/cat-tra-hue.jpg" },
  { href: "/products?category=banh-cung-dinh", name: "Bánh & Mứt cung đình", action: "Xem ngay", image: "/images/heritage/photos/hero-tea-table.jpg" },
  { href: "/products?category=qua-bieu", name: "Quà biếu", action: "Xem ngay", image: "/images/heritage/photos/cat-qua-bieu.jpg" }
];

const proofs = [
  { icon: "✺", title: "Chuẩn OCOP 4 sao", note: "Chứng nhận cấp tỉnh" },
  { icon: "⌗", title: "Mã QR từng lô", note: "Quét để xem nguồn gốc" },
  { icon: "⌾", title: "Chỉ dẫn địa lý", note: "Đặc sản sông Hương" },
  { icon: "♧", title: "Nguồn gốc rõ ràng", note: "Vùng nguyên liệu minh bạch" }
];

export function HomeExperience() {
  const router = useRouter();
  const [batchCode, setBatchCode] = useState("");

  function trace(event: FormEvent) {
    event.preventDefault();
    const code = batchCode.trim();
    if (!code) return;
    router.push(`/trace/${encodeURIComponent(code)}`);
  }

  return <main className="gh-home" id="main-content">
    {/* Hero: dark lacquer frame (kit hero A) */}
    <section className="gh-hero" aria-labelledby="gh-hero-title">
      <Motif name="02-canh-hoa" className="gh-hero__blossom" />
      <Motif name="01-hien-nha-phai" className="gh-hero__veranda" />
      <div className="gh-wrap">
        <div className="gh-hero__stage gh-surface-imperial">
          <div className="gh-hero__copy">
            <span className="gh-hero__kicker">Đặc sản &amp; quà biếu Cố đô</span>
            <h1 id="gh-hero-title">Phong vị Huế</h1>
            <div className="gh-hero__rule" aria-hidden="true" />
            <p>Mè xửng dẻo thơm, trà Huế thanh dịu, quà biếu đậm tình miền Trung — trao gửi bằng cả tấm lòng.</p>
            <div className="gh-hero__actions">
              <Link className="gh-btn gh-btn--fold" href="/products">Khám phá bộ sưu tập <span className="gh-btn__arrow" aria-hidden="true">→</span></Link>
              <a className="gh-btn gh-btn--paper gh-hero__ghost" href="#ocop-traceability">Truy xuất nguồn gốc</a>
            </div>
          </div>
          <div className="gh-hero__visual">
            <ProductImage fill priority sizes="(max-width: 960px) 100vw, 58vw" src="/images/heritage/photos/hero-studio.jpg" alt="Hộp quà, lon trà sen và đĩa mè xửng thủ công của Gói Huế" />
          </div>
        </div>
      </div>
    </section>

    {/* Featured products: fed by the recommendation API (placement=home) */}
    <section className="gh-home-section" id="featured" aria-label="Sản phẩm nổi bật">
      <div className="gh-wrap">
        <RecommendationRail placement="home" />
        <RecommendationPreferences />
      </div>
    </section>

    <div className="gh-wrap"><div className="gh-divider" aria-hidden="true"><span className="gh-divider__knot" /></div></div>

    {/* Collections: filter links into the catalog */}
    <section className="gh-home-section gh-collections" id="collection" aria-labelledby="gh-collection-title">
      <Motif name="05-may" className="gh-collections__cloud" />
      <div className="gh-wrap">
        <div className="gh-section-header gh-section-header--center">
          <div className="gh-section-header__line" />
          <h2 id="gh-collection-title">Chọn thức quà</h2>
          <div className="gh-section-header__line" />
        </div>
        <div className="gh-categories">
          {categories.map((category) => <Link className="gh-frame-card" href={category.href} key={category.href}>
            <div className="gh-card-media"><ProductImage fill sizes="(max-width: 600px) 100vw, (max-width: 1100px) 50vw, 25vw" src={category.image} alt="" /></div>
            <div className="gh-categories__foot"><strong>{category.name}</strong><span aria-hidden="true">→</span></div>
          </Link>)}
        </div>
      </div>
    </section>

    {/* Story */}
    <section className="gh-story" id="heritage-story" aria-labelledby="gh-story-title">
      <Motif name="02-canh-hoa" className="gh-story__blossom" />
      <Motif name="10-tuong-thanh" className="gh-story__wall" />
      <Motif name="12-hoa-sen-cung-dinh-phai" className="gh-story__lotus" />
      <div className="gh-wrap gh-story__grid">
        <div className="gh-story__copy">
          <span className="gh-kicker">Chuyện Cố đô</span>
          <h2 id="gh-story-title">Bên chén trà Huế</h2>
          <div className="gh-story__rule" aria-hidden="true" />
          <p>Thưởng chén trà Huế, cảm nhận hương vị thanh dịu và nét tinh tế trong những thức quà mộc mạc của cố đô.</p>
          <p className="gh-story__muted">Một miếng mè xửng dẻo thơm mùi mạch nha, nhấp cùng ngụm trà sen ấm nóng — lời mời thong thả ngồi lại, kể nhau nghe đôi chuyện xứ kinh kỳ.</p>
          <Link className="gh-btn gh-btn--paper" href="/products?category=tra-cung-dinh">Khám phá Trà Huế <span className="gh-btn__arrow" aria-hidden="true">→</span></Link>
        </div>
        <div className="gh-frame-plaque gh-story__photo">
          <ProductImage width={900} height={620} sizes="(max-width: 960px) 100vw, 50vw" src="/images/heritage/photos/story-tra-hue.jpg" alt="Bàn tay rót trà Huế từ ấm sứ hoa lam bên đĩa mè xửng" />
        </div>
      </div>
    </section>

    {/* Traceability: real /trace/[qrCode] flow */}
    <section className="gh-home-section" id="ocop-traceability" aria-labelledby="gh-trace-title">
      <div className="gh-wrap">
        <div className="gh-trace gh-surface-imperial">
          <div className="gh-trace__copy">
            <span className="gh-hero__kicker">Hệ thống truy xuất nguồn gốc</span>
            <h2 id="gh-trace-title">Minh bạch nguồn gốc từng hộp quà</h2>
            <p>Mỗi hộp xuất xưởng đều gắn mã định danh. Nhập mã QR trên nhãn để xem vùng nguyên liệu, lô sản xuất và chứng nhận.</p>
            <form onSubmit={trace}>
              <div className="gh-field-group">
                <label className="gh-field-label gh-trace__label" htmlFor="trace-code">Mã QR trên nhãn hộp</label>
                <div className="gh-trace__row">
                  <input id="trace-code" className="gh-field" value={batchCode} onChange={(event) => setBatchCode(event.target.value)} placeholder="Ví dụ: QR-OMA-20261015-LOT08" required />
                  <button type="button" className="gh-btn gh-btn--paper" onClick={() => setBatchCode("QR-OMA-20261015-LOT08")}>Mã mẫu</button>
                  <button type="submit" className="gh-btn">Kiểm tra</button>
                </div>
              </div>
            </form>
          </div>
          <ul className="gh-trace__proofs">
            {proofs.map((proof) => <li key={proof.title}><span aria-hidden="true">{proof.icon}</span><strong>{proof.title}</strong><small>{proof.note}</small></li>)}
          </ul>
        </div>
      </div>
    </section>

    {/* Account call-to-action */}
    <section className="gh-home-section" aria-labelledby="gh-account-title">
      <div className="gh-wrap">
        <div className="gh-frame-functional gh-frame-functional--fold gh-account-cta">
          <div>
            <span className="gh-kicker">Đồng hành cùng khách hàng</span>
            <h2 id="gh-account-title">Một tài khoản cho trọn hành trình</h2>
            <p>Quản lý đơn hàng, thông tin cá nhân và yêu cầu đổi trả ngay tại Gói Huế.</p>
          </div>
          <Link className="gh-btn gh-btn--brown gh-btn--fold" href="/account/profile">Mở tài khoản của tôi <span className="gh-btn__arrow" aria-hidden="true">→</span></Link>
        </div>
      </div>
    </section>
  </main>;
}
