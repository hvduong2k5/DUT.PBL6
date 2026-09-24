import Link from "next/link";
import type { ReactNode } from "react";

interface AccountShellProps {
  active: "profile" | "addresses";
  customerName?: string;
  customerEmail?: string;
  children: ReactNode;
}

export function AccountShell({ active, customerName = "Thành viên Tri Kỷ", customerEmail, children }: AccountShellProps) {
  const initial = customerName.trim().charAt(0).toLocaleUpperCase("vi") || "Ô";
  return (
    <main className="account-page">
      <nav className="breadcrumbs" aria-label="Đường dẫn trang">
        <Link href="/">Trang chủ</Link><span aria-hidden="true">›</span><span>Tài khoản Tri Kỷ</span>
      </nav>
      <div className="account-layout">
        <aside className="account-sidebar">
          <section className="account-identity">
            <div className="account-avatar" aria-hidden="true">{initial}</div>
            <div><strong>{customerName}</strong>{customerEmail && <small>{customerEmail}</small>}</div>
          </section>
          <nav className="account-menu" aria-label="Điều hướng tài khoản">
            <Link className={active === "profile" ? "active" : ""} href="/account/profile"><span>♙</span> Hồ sơ cá nhân <b>›</b></Link>
            <Link className={active === "addresses" ? "active" : ""} href="/account/addresses"><span>⌖</span> Sổ địa chỉ nhận hàng <b>›</b></Link>
            <span className="account-menu-disabled"><span>▤</span> Đơn hàng của tôi <small>Sắp có</small></span>
            <span className="account-menu-disabled"><span>♡</span> Điểm &amp; ưu đãi <small>Sắp có</small></span>
          </nav>
          <div className="account-help"><span>☏</span><p><strong>Cần hỗ trợ?</strong><br />Hotline Tri Kỷ: 1900 68 Hue</p></div>
        </aside>
        <div className="account-content">{children}</div>
      </div>
    </main>
  );
}
