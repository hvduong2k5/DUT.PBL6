"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, type ReactNode } from "react";
import type { AdminPermission } from "@/lib/auth/types";
import { useAdminSession } from "@/components/auth/admin-session-provider";

const NAVIGATION: Array<{ group: string; items: Array<{ href: string; label: string; icon: string; permission: AdminPermission }> }> = [
  { group: "Tổng quan", items: [{ href: "/", label: "Bảng điều khiển", icon: "⌂", permission: "ADMIN_DASHBOARD_VIEW" }] },
  { group: "Công việc của tôi", items: [
    { href: "/work/warehouse", label: "Warehouse Workbench", icon: "▦", permission: "WAREHOUSE_WORKBENCH_VIEW" },
    { href: "/work/packing", label: "Workbench đóng gói", icon: "▣", permission: "PACKING_WORKBENCH_VIEW" },
    { href: "/work/delivery", label: "Công việc giao hàng", icon: "▰", permission: "DELIVERY_WORKBENCH_VIEW" }
  ] },
  { group: "Danh mục", items: [{ href: "/catalog/products", label: "Sản phẩm & SKU", icon: "◇", permission: "PRODUCT_VIEW" }] },
  { group: "Vận hành", items: [
    { href: "/inventory", label: "Tồn kho & Batch", icon: "▦", permission: "INVENTORY_MANAGEMENT_VIEW" },
    { href: "/packing", label: "Quản lý đóng gói", icon: "▧", permission: "PACKING_MANAGEMENT_VIEW" },
    { href: "/shipping", label: "Quản lý giao vận", icon: "▰", permission: "SHIPMENT_MANAGEMENT_VIEW" }
  ] },
  { group: "Kinh doanh", items: [
    { href: "/orders", label: "Đơn hàng", icon: "▣", permission: "ORDER_VIEW" },
    { href: "/payments", label: "Thanh toán", icon: "◫", permission: "PAYMENT_VIEW" },
    { href: "/b2b/quotes", label: "Báo giá B2B", icon: "▤", permission: "B2B_REQUEST_VIEW" }
  ] },
  { group: "Hậu mãi", items: [
    { href: "/returns", label: "Đổi trả & hoàn tiền", icon: "↺", permission: "RETURN_CASE_VIEW" },
    { href: "/support/tickets", label: "Ticket hỗ trợ", icon: "◇", permission: "TICKET_QUEUE_VIEW" }
  ] },
  { group: "Quản trị", items: [
    { href: "/administration/employees", label: "Tài khoản nhân viên", icon: "♙", permission: "EMPLOYEE_ACCOUNT_VIEW" },
    { href: "/administration/roles", label: "Role", icon: "◈", permission: "ROLE_VIEW" },
    { href: "/administration/permissions", label: "Permission", icon: "⌘", permission: "PERMISSION_VIEW" }
  ] }
];

export function AdminShell({ children }: { children: ReactNode }) {
  const pathname = usePathname() ?? "";
  const { status, session, hasPermission, reload } = useAdminSession();
  const [mobileOpen, setMobileOpen] = useState(false);
  if (status === "loading") return <main className="admin-state" aria-busy="true"><span className="admin-spinner" /><h1>Đang kiểm tra quyền nhân viên…</h1></main>;
  if (!session) return <main className="admin-state"><span>!</span><h1>Không thể mở Web Admin</h1><p>Phiên nhân viên hoặc projection phân quyền không khả dụng.</p><button className="admin-primary-button" onClick={() => void reload()}>Thử lại</button></main>;
  const initials = session.employee.displayName.split(" ").slice(-2).map((part) => part[0]).join("").toLocaleUpperCase("vi");
  return <div className="admin-root">
    <header className="admin-topbar"><button className="admin-menu-toggle" type="button" onClick={() => setMobileOpen((value) => !value)} aria-label="Mở menu">☰</button><Link className="admin-brand" href="/"><span>Ô</span><div><strong>Ô MẠ</strong><small>Hệ thống vận hành nội bộ</small></div></Link><form className="admin-global-search" role="search"><span>⌕</span><input aria-label="Tìm nhanh" placeholder="Tìm mã yêu cầu, đơn hàng, khách hàng…" /></form><div className="admin-top-actions"><button type="button" aria-label="Thông báo">♢<b>3</b></button><div className="admin-employee"><span>{initials}</span><div><strong>{session.employee.displayName}</strong><small>{session.employee.jobTitle}</small></div><button type="button" aria-label="Mở menu tài khoản">⌄</button></div></div></header>
    <aside className={`admin-sidebar ${mobileOpen ? "open" : ""}`}><div className="admin-scope"><small>Không gian làm việc</small><strong>{session.employee.department}</strong><span>● Đang hoạt động</span></div><nav>{NAVIGATION.map((section) => { const visible = section.items.filter((item) => hasPermission(item.permission)); if (!visible.length) return null; return <section key={section.group}><h2>{section.group}</h2>{visible.map((item) => <Link className={pathname === item.href || (item.href !== "/" && pathname.startsWith(item.href)) ? "active" : ""} href={item.href} key={item.href} onClick={() => setMobileOpen(false)}><span>{item.icon}</span>{item.label}<b>›</b></Link>)}</section>; })}</nav><footer><span>Quyền hiệu lực</span><strong>{session.permissions.length} permission</strong><small>{session.roles.join(" · ")}</small></footer></aside>
    <div className="admin-content">{children}</div>
  </div>;
}
