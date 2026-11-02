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
    { href: "/admin/work/warehouse", label: "Warehouse Workbench", icon: "▦", permission: "WAREHOUSE_WORKBENCH_VIEW" },
    { href: "/admin/work/packing", label: "Workbench đóng gói", icon: "▣", permission: "PACKING_WORKBENCH_VIEW" },
    { href: "/admin/work/delivery", label: "Công việc giao hàng", icon: "▰", permission: "DELIVERY_WORKBENCH_VIEW" }
  ] },
  { group: "Danh mục", items: [{ href: "/admin/catalog/products", label: "Sản phẩm & SKU", icon: "◇", permission: "PRODUCT_VIEW" }] },
  { group: "Vận hành", items: [
    { href: "/admin/inventory", label: "Tồn kho & Batch", icon: "▦", permission: "INVENTORY_MANAGEMENT_VIEW" },
    { href: "/admin/packing", label: "Quản lý đóng gói", icon: "▧", permission: "PACKING_MANAGEMENT_VIEW" },
    { href: "/admin/shipping", label: "Quản lý giao vận", icon: "▰", permission: "SHIPMENT_MANAGEMENT_VIEW" }
  ] },
  { group: "Kinh doanh", items: [
    { href: "/admin/orders", label: "Đơn hàng", icon: "▣", permission: "ORDER_VIEW" },
    { href: "/admin/payments", label: "Thanh toán", icon: "◫", permission: "PAYMENT_VIEW" },
    { href: "/admin/b2b/quotes", label: "Báo giá B2B", icon: "▤", permission: "B2B_REQUEST_VIEW" }
  ] },
  { group: "Hậu mãi", items: [
    { href: "/admin/returns", label: "Đổi trả & hoàn tiền", icon: "↺", permission: "RETURN_CASE_VIEW" },
    { href: "/admin/support/tickets", label: "Ticket hỗ trợ", icon: "◇", permission: "TICKET_QUEUE_VIEW" }
  ] },
  { group: "Quản trị", items: [
    { href: "/admin/administration/employees", label: "Tài khoản nhân viên", icon: "♙", permission: "EMPLOYEE_ACCOUNT_VIEW" },
    { href: "/admin/administration/roles", label: "Role", icon: "◈", permission: "ROLE_VIEW" },
    { href: "/admin/administration/permissions", label: "Permission", icon: "⌘", permission: "PERMISSION_VIEW" }
  ] }
];
