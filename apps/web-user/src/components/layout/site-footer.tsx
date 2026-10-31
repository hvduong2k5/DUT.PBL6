import Link from "next/link";
import { BrandMark } from "./brand-mark";

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
          <p><Link href="/support/request">Trung tâm hỗ trợ khách hàng</Link></p><p>Chính sách vận chuyển</p><p>Cam kết nguồn gốc</p>
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
