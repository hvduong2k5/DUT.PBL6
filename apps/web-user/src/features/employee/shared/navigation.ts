import type { EmployeePermission } from "@/features/employee/auth/types";

export interface EmployeeNavigationItem {
  href: string;
  label: string;
  icon: string;
  permission?: EmployeePermission;
}

export const EMPLOYEE_NAVIGATION: Array<{ group: string; items: EmployeeNavigationItem[] }> = [
  { group: "Tổng quan", items: [{ href: "/admin", label: "Bảng điều khiển", icon: "⌂" }] },
  { group: "Công việc của tôi", items: [
    { href: "/admin/work/packing", label: "Workbench đóng gói", icon: "▣", permission: "PACKING_WORKBENCH_VIEW" }
  ] },
  { group: "Danh mục", items: [{ href: "/admin/catalog/products", label: "Sản phẩm & SKU", icon: "◇", permission: "PRODUCT_VIEW" }] },
  { group: "Vận hành", items: [
    { href: "/admin/packing", label: "Quản lý đóng gói", icon: "▧", permission: "PACKING_MANAGEMENT_VIEW" }
  ] },
  { group: "Kinh doanh", items: [
    { href: "/admin/orders", label: "Đơn hàng", icon: "▣", permission: "ORDER_VIEW" }
  ] },
  { group: "Quản trị", items: [
    { href: "/admin/administration/employees", label: "Tài khoản nhân viên", icon: "♙", permission: "EMPLOYEE_ACCOUNT_VIEW" },
    { href: "/admin/administration/roles", label: "Role", icon: "◈", permission: "ROLE_VIEW" },
    { href: "/admin/administration/permissions", label: "Permission", icon: "⌘", permission: "PERMISSION_VIEW" }
  ] }
];
