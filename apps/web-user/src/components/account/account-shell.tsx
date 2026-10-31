"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { LogoutButton } from "./logout-button";

interface AccountShellProps {
  active: "profile" | "addresses" | "orders" | "loyalty";
  customerName?: string;
  customerEmail?: string;
  children: ReactNode;
}

export function AccountShell({ active, customerName = "Thành viên Tri Kỷ", customerEmail, children }: AccountShellProps) {
  const { hasCapability } = useCustomerSession();
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
            {hasCapability("PROFILE_MANAGE") ? <Link className={active === "profile" ? "active" : ""} href="/account/profile"><span>♙</span> Hồ sơ cá nhân <b>›</b></Link> : null}
            {hasCapability("ADDRESS_MANAGE") ? <Link className={active === "addresses" ? "active" : ""} href="/account/addresses"><span>⌖</span> Sổ địa chỉ nhận hàng <b>›</b></Link> : null}
            {hasCapability("ORDER_HISTORY_VIEW") ? <Link className={active === "orders" ? "active" : ""} href="/account/orders"><span>▤</span> Đơn hàng của tôi <b>›</b></Link> : null}
            {hasCapability("LOYALTY_VIEW") ? <Link className={active === "loyalty" ? "active" : ""} href="/account/loyalty"><span>♡</span> Điểm &amp; ưu đãi <b>›</b></Link> : null}
          </nav>
          <LogoutButton />
          <div className="account-help"><span>☏</span><p><strong>Cần hỗ trợ?</strong><br />Hotline Tri Kỷ: 1900 68 Hue</p></div>
        </aside>
        <div className="account-content">{children}</div>
      </div>
    </main>
  );
}
