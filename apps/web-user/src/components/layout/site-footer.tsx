import Image from "next/image";
import Link from "next/link";

export function SiteFooter() {
  return (
    <div className="hue-surface">
      <footer className="hue-site-footer">
        <div className="wrap">
          <div className="hue-footer-grid">
            <div className="footer-col">
              <div
                className="row"
                style={{ gap: "12px", marginBottom: "14px" }}
              >
                <Image
                  src="/images/goi-hue/logo-goi-hue.svg"
                  alt=""
                  width="36"
                  height="36"
                  style={{
                    background: "var(--kem)",
                    padding: "4px",
                    borderRadius: "4px",
                  }}
                />
                <strong
                  style={{
                    font: "700 1.7rem/1 var(--chu-de)",
                    color: "var(--kem)",
                  }}
                >
                  Gói Huế
                </strong>
              </div>
              <p
                style={{
                  fontSize: "0.9rem",
                  color: "rgba(252,249,244,0.8)",
                  marginBottom: "14px",
                }}
              >
                Thức quà đặc sản &amp; trà biếu xứ Huế. Đóng gói chỉn chu trong
                chất liệu giấy Dó và họa tiết di sản Cố đô.
              </p>
              <small style={{ color: "var(--mat-sang)" }}>
                Phường Đông Ba, Quận Phú Xuân, Thành phố Huế
              </small>
            </div>
            <div className="footer-col">
              <h3>Bộ sưu tập</h3>
              <ul className="footer-links">
                <li>
                  <Link
                    href="/products?category=me-xung-keo-hue"
                    data-nav-group="0"
                  >
                    Mè xửng dẻo &amp; giòn
                  </Link>
                </li>
                <li>
                  <Link
                    href="/products?category=tra-cung-dinh"
                    data-nav-group="1"
                  >
                    Trà sen Tịnh Tâm &amp; Trà cung đình
                  </Link>
                </li>
                <li>
                  <Link href="/products?category=qua-bieu" data-nav-group="2">
                    Hộp quà biếu Đoàn Viên
                  </Link>
                </li>
                <li>
                  <Link href="/products?category=qua-bieu">Chọn quà biếu</Link>
                </li>
              </ul>
            </div>
            <div className="footer-col">
              <h3>Chăm sóc khách hàng</h3>
              <ul className="footer-links">
                <li>
                  <Link href="/#home-batch">Tra cứu nguồn gốc</Link>
                </li>
                <li>
                  <Link href="/support/request">Hỗ trợ khách hàng</Link>
                </li>
                <li>
                  <Link href="/promotions">Ưu đãi</Link>
                </li>
                <li>
                  <Link href="/#main">Về đầu trang ↑</Link>
                </li>
              </ul>
            </div>
            <div className="footer-col">
              <h3>Đồng hành &amp; Đặt quà</h3>
              <p
                style={{
                  fontSize: "0.88rem",
                  color: "rgba(252,249,244,0.8)",
                  marginBottom: "12px",
                }}
              >
                Hỗ trợ gói quà doanh nghiệp, in thiệp chúc riêng và giao quà tận
                tay trên toàn quốc.
              </p>
              <Link className="btn" href="/cart">
                Mở giỏ quà của bạn
              </Link>
            </div>
          </div>
          <div
            className="between"
            style={{
              borderTop: "1px solid rgba(252,249,244,0.16)",
              paddingTop: "20px",
              fontSize: "0.82rem",
              color: "rgba(252,249,244,0.7)",
            }}
          >
            © 2026 Gói Huế
          </div>
        </div>
      </footer>
    </div>
  );
}
