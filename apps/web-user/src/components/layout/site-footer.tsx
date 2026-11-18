import Link from "next/link";
import { BrandMark } from "./brand-mark";
import { COLLECTIONS } from "./collections";

export function SiteFooter() {
  return (
    <footer className="gh-footer gh-surface-imperial">
      <div className="gh-wrap">
        <div className="gh-footer__grid">
          <div>
            <BrandMark />
            <p>Gìn giữ và tôn vinh phong vị bánh mứt, mè xửng và trà xứ Huế, đóng gói chỉn chu để trao gửi trọn tấm lòng.</p>
            <span className="gh-badge gh-badge--gold gh-footer__chip">OCOP 4 sao Thừa Thiên Huế</span>
          </div>
          <div>
            <h2>Bộ sưu tập</h2>
            <ul>{COLLECTIONS.map((item) => <li key={item.href}><Link href={item.href}>{item.name}</Link></li>)}</ul>
          </div>
          <div>
            <h2>Chăm sóc &amp; hỗ trợ</h2>
            <ul>
              <li><Link href="/support/request">Trung tâm hỗ trợ khách hàng</Link></li>
              <li><Link href="/promotions">Ưu đãi</Link></li>
              <li><Link href="/#ocop-traceability">Truy xuất nguồn gốc</Link></li>
            </ul>
          </div>
          <div>
            <h2>Phòng trưng bày &amp; xưởng bánh</h2>
            <p>128 Lê Lợi, Phường Phú Nhuận, Thành phố Huế</p>
            <p>(0234) 388 9988 · thuquan@omahue.vn</p>
          </div>
        </div>
        <div className="gh-footer__bottom">© 2026 Gói Huế · Tinh tuyển di sản · Chuẩn hóa OCOP</div>
      </div>
    </footer>
  );
}
