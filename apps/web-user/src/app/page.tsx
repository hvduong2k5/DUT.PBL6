import Link from "next/link";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";

export default function HomePage() {
  return (
    <><SiteHeader /><main className="home-hero"><span className="eyebrow">Di sản ẩm thực Cố Đô</span><h1>Thức quà xứ Huế,<br />gói trọn một miền thương.</h1><p>Authentication MVP đang được phát triển trên hành trình thành viên Tri Kỷ.</p><div className="home-actions"><Link className="primary-link" href="/login">Đăng nhập Tri Kỷ</Link><Link className="secondary-link" href="/register">Khởi tạo tài khoản</Link></div></main><SiteFooter /></>
  );
}
