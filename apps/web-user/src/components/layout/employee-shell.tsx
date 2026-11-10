"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, type ReactNode } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import { EMPLOYEE_NAVIGATION } from "@/features/employee/shared/navigation";
import { canUseFulfillment } from "@/features/employee/packing/fulfillment-types";

export function EmployeeShell({ children, analyticsEnabled = false, fulfillmentEnabled = false }: { children: ReactNode; analyticsEnabled?: boolean; fulfillmentEnabled?: boolean }) {
  const pathname = usePathname() ?? "/admin";
  const { session, employee, hasPermission, logout } = useEmployeeSession();
  const [mobileOpen, setMobileOpen] = useState(false);
  const initials = session.employee.displayName.split(" ").slice(-2).map((part) => part[0]).join("").toLocaleUpperCase("vi") || "NV";

  return <div className="admin-root">
    <header className="admin-topbar">
      <button className="admin-menu-toggle" type="button" onClick={() => setMobileOpen((value) => !value)} aria-label="Mở menu">☰</button>
      <Link className="admin-brand" href="/admin"><span>Ô</span><div><strong>Ô MẠ</strong><small>Hệ thống vận hành nội bộ</small></div></Link>
      <form className="admin-global-search" role="search" onSubmit={(event) => event.preventDefault()}><span>⌕</span><input aria-label="Tìm nhanh" placeholder="Tìm mã yêu cầu, đơn hàng, khách hàng…" /></form>
      <div className="admin-top-actions"><button type="button" aria-label="Thông báo">♢<b>3</b></button><Link href="/settings/password" aria-label="Đổi mật khẩu" title="Đổi mật khẩu">⌑</Link><div className="admin-employee"><span>{initials}</span><div><strong>{session.employee.displayName}</strong><small>{session.employee.jobTitle}</small></div><button type="button" onClick={() => void logout()} aria-label="Đăng xuất" title="Đăng xuất">⇥</button></div></div>
    </header>
    <aside className={`admin-sidebar ${mobileOpen ? "open" : ""}`}>
      <div className="admin-scope"><small>Không gian làm việc</small><strong>{session.employee.department}</strong><span>● Đang hoạt động</span></div>
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
          return <Link className={active ? "active" : ""} href={item.href} key={item.href} onClick={() => setMobileOpen(false)}><span>{item.icon}</span>{item.label}<b>›</b></Link>;
        })}</section>;
      })}</nav>
      <footer><span>Quyền hiệu lực</span><strong>{session.permissions.length} permission</strong><small>{session.roles.join(" · ") || "EMPLOYEE"}</small></footer>
    </aside>
    <div className="admin-content">{children}</div>
  </div>;
}
