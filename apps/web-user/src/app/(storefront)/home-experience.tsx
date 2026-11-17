"use client";

import Link from "next/link";
import { type FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { ProductImage } from "@/components/product/product-image";
import { RecommendationPreferences, RecommendationRail } from "@/components/recommendation/recommendation-rail";
import { collections } from "@/components/layout/storefront-links";

const art = "/images/goi-hue/";
const products = [
  {
    "skuId": "SKU-009-500",
    "slug": "bo-qua-hoang-trieu",
    "meta": "Trọng lượng: 500g",
    "name": "Hộp Quà Cố Đô Ngự Thiện",
    "price": "450.000đ",
    "image": "/images/goi-hue/photo-sp-hop-qua-mo.jpg"
  },
  {
    "skuId": "SKU-002-200",
    "slug": "me-xung-deo-ngu-thien",
    "meta": "Hộp cao cấp: 200g",
    "name": "Mè Xửng Dẻo Ngự Trà Thượng Hạng",
    "price": "85.000đ",
    "image": "/images/goi-hue/photo-sp-me-xung-dia.jpg"
  },
  {
    "skuId": "SKU-006-300",
    "slug": "banh-dau-xanh-trai-cay",
    "meta": "Hộp sơn mài: 12 cái",
    "name": "Bánh Đậu Xanh Trái Cây Nghệ Nhân",
    "price": "120.000đ",
    "image": "https://lh3.googleusercontent.com/aida-public/AB6AXuAGdlKPl8besUprgcIcaSNI-VXXEzt-aIkDTw6zE0If6NpcKmXbUXA58HO_oZbPDdSW8L_PwvVi7s3-n4fiI7yFLqHyWKiYM8zP30qlRMzUz2rDLvUDKnxRxHBMBN_TpgSNM9ciBOUI-kxDrnOfioqEgb9W16eSmlaDDnvGH_p6MK5CYsqFmT0ggFzCTnCpTmUHqbI7VNUkbm47hgj65LWqY9tqc9nNXP4jGIkreI45AUAse_CgYT8E"
  },
  {
    "skuId": "SKU-011-150",
    "slug": "tra-sen-tinh-tam",
    "meta": "Hộp thiếc cao cấp: 150g",
    "name": "Trà Sen Bách Diệp Hồ Tịnh Tâm",
    "price": "380.000đ",
    "image": "/images/goi-hue/photo-sp-tra-sen-lon.jpg"
  }
];

export function HomeExperience() {
  const router = useRouter();
  const [batchCode, setBatchCode] = useState("");
  const [newsletterSent, setNewsletterSent] = useState(false);
  function trace(event: FormEvent) {
    event.preventDefault();
    if (batchCode.trim()) router.push("/trace/" + encodeURIComponent(batchCode.trim()));
  }
  return <main className="gh-home">
    <section className="gh-hero" aria-labelledby="gh-title"><div className="gh-wrap gh-hero-inner">
      <div className="gh-hero-copy">
        <h1 id="gh-title">Phong vị Huế,<br /><em>gói ghém</em><br />trao nhau.</h1>
        <p>Mè xửng dẻo thơm, trà Huế thanh dịu và những thức quà dành cho người thương.</p>
        <Link className="gh-btn gh-gold" href="/products">Khám phá bộ sưu tập <span aria-hidden="true">→</span></Link>
      </div>
      <div className="gh-studio"><ProductImage fill priority sizes="(max-width: 767px) 100vw, 57vw" src={art + "photo-hero-studio.jpg"} alt="Mè xửng, chén trà và hộp quà Gói Huế trên bục trưng bày" /></div>
    </div></section>
    <section className="gh-wrap gh-section" aria-labelledby="gh-collections">
      <div className="gh-heading"><h2 id="gh-collections">Bộ sưu tập Gói Huế</h2><Link href="/products">Xem tất cả →</Link></div>
      <div className="gh-collections">{collections.map(item => <Link className="gh-tile" href={item.href} key={item.href}>
        <div className="gh-photo"><ProductImage fill sizes="(max-width: 767px) 92vw, 32vw" src={art + item.image} alt="" /></div>
        <div className="gh-caption"><h3>{item.name}</h3><span aria-hidden="true">→</span></div>
      </Link>)}</div>
    </section>
    <div className="gh-wrap gh-divider" aria-hidden="true" />
    <section id="heritage-story" className="gh-wrap gh-section gh-story">
      <div className="gh-story-photo"><ProductImage fill sizes="(max-width: 767px) 92vw, 50vw" src={art + "photo-story-nha-ruong.jpg"} alt="Hiên gỗ nhà rường nhìn ra khu vườn ở Huế" /></div>
      <div><h2>Thức quà từ Huế</h2><p>Một miếng mè xửng dẻo bùi, một ngụm trà ấm. Ở Huế, mời nhau đôi thức quà giản dị cũng là cách để câu chuyện bên bàn trà kéo dài thêm một chút.</p>
      <Link className="gh-text-link" href="/products?category=qua-bieu">Chọn quà biếu <span aria-hidden="true">→</span></Link></div>
    </section>
    <section className="gh-wrap gh-section"><div className="gh-heading"><h2>Gợi ý cho bàn trà</h2></div>
      <div className="gh-products">{products.map(item => <article key={item.skuId}><Link href={"/products/" + item.slug}>
        <div className="gh-photo"><ProductImage fill sizes="(max-width: 767px) 44vw, 23vw" src={item.image} alt="" /></div>
        <small>{item.meta}</small><h3>{item.name}</h3><strong>{item.price}</strong>
      </Link></article>)}</div>
      <RecommendationPreferences /><RecommendationRail placement="home" />
    </section>
    <section className="gh-wrap gh-services gh-section">
      <div id="ocop-traceability"><h2>Tra cứu nguồn gốc</h2><form onSubmit={trace}>
        <label htmlFor="gh-batch">Mã trên nhãn sản phẩm</label>
        <input id="gh-batch" value={batchCode} onChange={event => setBatchCode(event.target.value)} required />
        <div><button className="gh-btn" disabled={!batchCode.trim()}>Tra cứu</button><button type="button" className="gh-text-link" onClick={() => setBatchCode("QR-OMA-20261015-LOT08")}>Dùng mã mẫu</button></div>
      </form></div>
      <div><h2>Nhận tin từ Gói Huế</h2><form onSubmit={event => { event.preventDefault(); event.currentTarget.reset(); setNewsletterSent(true); }}>
        <label htmlFor="gh-email">Địa chỉ email</label><input id="gh-email" type="email" autoComplete="email" required />
        <button className="gh-btn">Đăng ký</button>
      </form>{newsletterSent && <p role="status">Đã ghi nhận trong bản xem thử.</p>}</div>
    </section>
  </main>;
}