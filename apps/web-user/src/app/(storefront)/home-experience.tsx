"use client";

import Link from "next/link";
import { HeritageBanner, type BannerSlide } from "@/components/heritage/heritage-banner";
import { Motif } from "@/components/heritage/motif";
import { TraceSearchForm } from "@/components/heritage/trace-search-form";
import { ProductImage } from "@/components/product/product-image";
import { RecommendationRail } from "@/components/recommendation/recommendation-rail";

const categories = [
  { href: "/products?category=me-xung-keo-hue", name: "Mè xửng Huế", note: "Dẻo thơm mạch nha, mè rang", image: "/images/heritage/photos/cat-me-xung.jpg" },
  { href: "/products?category=tra-cung-dinh", name: "Trà Huế", note: "Trà sen, trà cung đình", image: "/images/heritage/photos/cat-tra-hue.jpg" },
  { href: "/products?category=banh-cung-dinh", name: "Bánh & Mứt cung đình", note: "Bánh in, mứt, điểm tâm", image: "/images/heritage/photos/hero-tea-table.jpg" },
  { href: "/products?category=qua-bieu", name: "Quà biếu", note: "Hộp quà biếu tặng trang nhã", image: "/images/heritage/photos/cat-qua-bieu.jpg" }
];

const HERO_SLIDES: BannerSlide[] = [
  { kicker: "Đặc sản & quà biếu Cố đô", title: "Phong vị Huế", lead: "Mè xửng dẻo thơm, trà Huế thanh dịu, quà biếu đậm tình miền Trung — trao gửi bằng cả tấm lòng.",
    image: { src: "/images/heritage/photos/hero-studio.jpg", alt: "Hộp quà, lon trà sen và đĩa mè xửng thủ công của Gói Huế" } },
  { kicker: "Bên chén trà Huế", title: "Thong thả một chén trà", lead: "Mè xửng mạch nha dẻo thơm cùng trà sen thanh dịu — lời mời ngồi lại bên hiên nhà rường.",
    image: { src: "/images/heritage/photos/hero-tea-table.jpg", alt: "Đĩa mè xửng bên ấm trà sứ xanh trắng dưới nắng hiên nhà" } },
  { kicker: "Quà biếu Cố đô", title: "Hộp quà trao gửi tấm lòng", lead: "Hộp Đoàn Viên ba khoang, đóng gói chỉn chu cho người thân và đối tác.",
    image: { src: "/images/heritage/photos/sp-hop-qua-doan-vien.jpg", alt: "Hộp quà Đoàn Viên Gói Huế mở nắp" } }
];

const traceProofs = ["Chuẩn OCOP 4 sao", "Mã QR từng lô", "Chỉ dẫn địa lý", "Nguồn gốc rõ ràng"];

export function HomeExperience() {
  return <main className="gh-home" id="main-content">
    <section className="gh-home-band gh-home-hero" aria-labelledby="gh-hero-title">
      <Motif name="02-canh-hoa" className="gh-band__blossom" />
      <Motif name="01-hien-nha-phai" className="gh-band__veranda" />
      <div className="gh-wrap">
        <HeritageBanner level={1} titleId="gh-hero-title" priority slides={HERO_SLIDES}>
          <div className="gh-banner__actions">
            <Link className="gh-btn gh-btn--fold" href="/products">Khám phá bộ sưu tập <span className="gh-btn__arrow" aria-hidden="true">→</span></Link>
            <a className="gh-btn gh-btn--ghost" href="#ocop-traceability">Truy xuất nguồn gốc</a>
          </div>
        </HeritageBanner>
      </div>
    </section>

    <section className="gh-home-section" id="featured" aria-label="Sản phẩm nổi bật">
      <div className="gh-wrap">
        <RecommendationRail placement="home" />
      </div>
    </section>

    <div className="gh-wrap"><div className="gh-divider" aria-hidden="true"><span className="gh-divider__knot" /></div></div>

    <section className="gh-home-section gh-collections" id="collection" aria-labelledby="gh-collection-title">
      <Motif name="05-may" className="gh-collections__cloud" />
      <Motif name="07-hat-me" className="gh-collections__seeds" />
      <div className="gh-wrap">
        <div className="gh-section-header gh-section-header--center">
          <div className="gh-section-header__line" />
          <h2 id="gh-collection-title">Chọn thức quà</h2>
          <div className="gh-section-header__line" />
        </div>
        <div className="gh-categories">
          {categories.map((category) => <Link className="gh-frame-card" href={category.href} key={category.href}>
            <div className="gh-card-media"><ProductImage fill sizes="(max-width: 600px) 100vw, (max-width: 1100px) 50vw, 25vw" src={category.image} alt="" /></div>
            <div className="gh-categories__foot"><span><strong>{category.name}</strong><small>{category.note}</small></span><span className="gh-categories__arrow" aria-hidden="true">→</span></div>
          </Link>)}
        </div>
      </div>
    </section>

    <section className="gh-story" id="heritage-story" aria-labelledby="gh-story-title">
      <Motif name="02-canh-hoa" className="gh-story__blossom" />
      <Motif name="01-hien-nha" className="gh-story__veranda" />
      <Motif name="10-tuong-thanh" className="gh-story__wall" />
      <Motif name="12-hoa-sen-cung-dinh-phai" className="gh-story__lotus" />
      <div className="gh-wrap">
        <div className="gh-story__grid">
          <div className="gh-story__copy">
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
        <div className="gh-story__grid gh-story__grid--reverse">
          <div className="gh-frame-plaque gh-story__photo">
            <ProductImage width={900} height={620} sizes="(max-width: 960px) 100vw, 50vw" src="/images/heritage/photos/story-nha-ruong.jpg" alt="Hiên nhà rường gỗ cổ truyền xứ Huế ngập nắng" />
          </div>
          <div className="gh-story__copy">
            <span className="gh-kicker">Di sản kiến trúc &amp; ẩm thực</span>
            <h2>Thức quà từ Huế</h2>
            <div className="gh-story__rule" aria-hidden="true" />
            <p>Gói Huế mang đến những thức quà mộc mạc mà tinh tế của xứ Huế: mè xửng dẻo thơm, trà Huế thanh dịu và những món quà biếu trang nhã.</p>
            <p className="gh-story__muted">Mỗi hộp quà lấy cảm hứng từ nếp giấy Dó thủ công, nhịp mái hiên nhà rường và cổng thành cổ kính.</p>
            <Link className="gh-btn gh-btn--brown gh-btn--fold" href="/products?category=qua-bieu">Chọn hộp quà biếu <span className="gh-btn__arrow" aria-hidden="true">→</span></Link>
          </div>
        </div>
      </div>
    </section>

    <section className="gh-home-band" id="ocop-traceability" aria-labelledby="gh-trace-title">
      <Motif name="09-cong-thanh" className="gh-band__gate" />
      <Motif name="02-canh-hoa-phai" className="gh-band__blossom-right" />
      <div className="gh-wrap">
        <HeritageBanner titleId="gh-trace-title" kicker="Hệ thống truy xuất nguồn gốc" title="Minh bạch nguồn gốc từng hộp quà"
          lead="Mỗi hộp xuất xưởng đều gắn mã định danh. Nhập mã QR trên nhãn để xem vùng nguyên liệu, lô sản xuất và chứng nhận."
          image={{ src: "/images/heritage/photos/sp-goi-le-tra.jpg", alt: "Túi mè xửng và trà cung đình Gói Huế có nhãn mã QR truy xuất" }}>
          <TraceSearchForm inputId="home-trace-code" />
          <ul className="gh-banner__badges" aria-label="Cam kết nguồn gốc">
            {traceProofs.map((proof) => <li key={proof} className="gh-badge gh-badge--gold">{proof}</li>)}
          </ul>
        </HeritageBanner>
      </div>
    </section>

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
