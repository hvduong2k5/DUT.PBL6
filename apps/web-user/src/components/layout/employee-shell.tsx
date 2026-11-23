"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, type ReactNode } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import { EMPLOYEE_NAVIGATION } from "@/features/employee/shared/navigation";
import { canUseFulfillment } from "@/features/employee/packing/fulfillment-types";
import { GridIcon, SearchIcon } from "@/components/layout/icons";

export function EmployeeShell({ children, analyticsEnabled = false, fulfillmentEnabled = false }: { children: ReactNode; analyticsEnabled?: boolean; fulfillmentEnabled?: boolean }) {
  const pathname = usePathname() ?? "/admin";
  const { session, employee, hasPermission, logout } = useEmployeeSession();
  const [mobileOpen, setMobileOpen] = useState(false);
  const initials = session.employee.displayName.split(" ").slice(-2).map((part) => part[0]).join("").toLocaleUpperCase("vi") || "NV";

  return <div className="admin-root">
    <header className="admin-topbar gh-surface-imperial">
      <button className="admin-menu-toggle" type="button" onClick={() => setMobileOpen((value) => !value)} aria-label="Mở menu" aria-expanded={mobileOpen}><GridIcon /></button>
      <Link className="admin-brand" href="/admin" aria-label="Gói Huế - Hệ thống vận hành nội bộ">
        {/* Kit emblem, static SVG. */}
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src="/images/heritage/logo-goi-hue.svg" alt="" width={38} height={38} />
        <div><strong>Gói Huế</strong><small>Hệ thống vận hành nội bộ</small></div>
      </Link>
      <form className="admin-global-search" role="search" onSubmit={(event) => event.preventDefault()}><span aria-hidden="true"><SearchIcon /></span><input aria-label="Tìm nhanh" placeholder="Tìm mã yêu cầu, đơn hàng, khách hàng…" /></form>
      <div className="admin-top-actions">
        <button className="admin-top-link" type="button">Thông báo</button>
        <Link className="admin-top-link" href="/settings/password">Đổi mật khẩu</Link>
        <div className="admin-employee"><span aria-hidden="true">{initials}</span><div><strong>{session.employee.displayName}</strong><small>{session.employee.jobTitle}</small></div><button type="button" onClick={() => void logout()}>Đăng xuất</button></div>
      </div>
    </header>
    <aside className={`admin-sidebar ${mobileOpen ? "open" : ""}`}>
      <div className="admin-scope"><small>Không gian làm việc</small><strong>{session.employee.department}</strong><span>Đang hoạt động</span></div>
      <nav>{EMPLOYEE_NAVIGATION.map((section) => {
        const visible = section.items.filter((item) => {
          const analyticsVisible = item.href !== "/admin/analytics" || (analyticsEnabled && hasPermission("BATCH_VIEW"));
          const baseVisible = (!item.permission || hasPermission(item.permission)) && analyticsVisible;
          const fulfillmentVisible = fulfillmentEnabled && ["/admin/packing", "/admin/work/packing"].includes(item.href) && canUseFulfillment(employee);
          return baseVisible || fulfillmentVisible;
        });
        if (!visible.length) return null;
        return <section key={section.group}><h2>{section.group}</h2>{visible.map((item) => {
          const active = pathname === item.href || (item.href !== "/admin" && pathname.startsWith(`${item.href}/`));
          return <Link className={active ? "active" : ""} href={item.href} key={item.href} onClick={() => setMobileOpen(false)}>{item.label}</Link>;
        })}</section>;
      })}</nav>
      <footer><span>Quyền hiệu lực</span><strong>{session.permissions.length} permission</strong><small>{session.roles.join(" · ") || "EMPLOYEE"}</small></footer>
    </aside>
    <div className="admin-content">{children}</div>
  </div>;
}
