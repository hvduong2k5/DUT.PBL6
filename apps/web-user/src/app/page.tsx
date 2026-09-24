import Link from "next/link";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";

export default function HomePage() {
  return (
    <><SiteHeader /><main className="home-hero"><span className="eyebrow">Di sản ẩm thực Cố Đô</span><h1>Thức quà xứ Huế,<br />gói trọn một miền thương.</h1><p>Khám phá bánh mứt, trà sen và những hộp quà được tinh tuyển từ nếp xưa.</p><div className="home-actions"><Link className="primary-link" href="/products">Khám phá bộ sưu tập</Link><Link className="secondary-link" href="/products?category=qua-bieu">Chọn quà biếu</Link></div></main><SiteFooter /></>
  );
}
