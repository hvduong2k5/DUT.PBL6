"use client";

import Link from "next/link";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import { EMPLOYEE_NAVIGATION } from "./navigation";

export function EmployeeDashboard() {
  const { session, hasPermission } = useEmployeeSession();
  const groups = EMPLOYEE_NAVIGATION.slice(1).map((section) => ({
    ...section,
    items: section.items.filter((item) => !item.permission || hasPermission(item.permission))
  })).filter((section) => section.items.length);
  const modules = groups.flatMap((section) => section.items);
  const workbenches = modules.filter((item) => item.href.includes("/work/"));

  return <main className="admin-page">
    <header className="admin-page-header"><div><nav>Nhân viên <span>›</span> Tổng quan</nav><h1>Xin chào, {session.employee.displayName}</h1><p>Các khu vực bên dưới được tạo trực tiếp từ permission hiệu lực của tài khoản.</p></div></header>
    <section className="admin-summary-grid" aria-label="Tổng quan quyền truy cập">
      <article><span>Phân hệ khả dụng</span><strong>{modules.length}</strong><small>Đã lọc theo permission</small></article>
      <article><span>Workbench cá nhân</span><strong>{workbenches.length}</strong><small>Công việc theo phân công</small></article>
      <article><span>Permission hiệu lực</span><strong>{session.permissions.length}</strong><small>Do Identity Service cung cấp</small></article>
      <article><span>Role được gán</span><strong>{session.roles.length}</strong><small>{session.roles.join(" · ") || "Chưa có role"}</small></article>
    </section>
    <section className="employee-module-grid" aria-label="Các phân hệ được phép truy cập">
      {groups.map((section) => <article key={section.group}><h2>{section.group}</h2><div>{section.items.map((item) => <Link href={item.href} key={item.href}><span>{item.icon}</span><strong>{item.label}</strong><b>›</b></Link>)}</div></article>)}
      {!groups.length ? <div className="admin-empty-state"><strong>Chưa có phân hệ được cấp quyền</strong><p>Liên hệ quản trị viên Access Management để kiểm tra Role và Permission.</p></div> : null}
    </section>
  </main>;
}
