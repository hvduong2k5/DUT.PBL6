import type { EmployeePermission } from "@/features/employee/auth/types";

export interface EmployeeNavigationItem {
  href: string;
  label: string;
  icon: string;
  permission?: EmployeePermission;
}

export const EMPLOYEE_NAVIGATION: Array<{ group: string; items: EmployeeNavigationItem[] }> = [
  { group: "Tổng quan", items: [{ href: "/employee", label: "Bảng điều khiển", icon: "⌂" }] },
  { group: "Công việc của tôi", items: [
    { href: "/employee/work/warehouse", label: "Warehouse Workbench", icon: "▦", permission: "WAREHOUSE_WORKBENCH_VIEW" },
    { href: "/employee/work/packing", label: "Workbench đóng gói", icon: "▣", permission: "PACKING_WORKBENCH_VIEW" },
    { href: "/employee/work/delivery", label: "Công việc giao hàng", icon: "▰", permission: "DELIVERY_WORKBENCH_VIEW" }
  ] },
  { group: "Danh mục", items: [{ href: "/employee/catalog/products", label: "Sản phẩm & SKU", icon: "◇", permission: "PRODUCT_VIEW" }] },
  { group: "Vận hành", items: [
    { href: "/employee/inventory", label: "Tồn kho & Batch", icon: "▦", permission: "INVENTORY_MANAGEMENT_VIEW" },
    { href: "/employee/packing", label: "Quản lý đóng gói", icon: "▧", permission: "PACKING_MANAGEMENT_VIEW" },
    { href: "/employee/shipping", label: "Quản lý giao vận", icon: "▰", permission: "SHIPMENT_MANAGEMENT_VIEW" }
  ] },
  { group: "Kinh doanh", items: [
    { href: "/employee/orders", label: "Đơn hàng", icon: "▣", permission: "ORDER_VIEW" },
    { href: "/employee/payments", label: "Thanh toán", icon: "◫", permission: "PAYMENT_VIEW" },
    { href: "/employee/b2b/quotes", label: "Báo giá B2B", icon: "▤", permission: "B2B_REQUEST_VIEW" }
  ] },
  { group: "Hậu mãi", items: [
    { href: "/employee/returns", label: "Đổi trả & hoàn tiền", icon: "↺", permission: "RETURN_CASE_VIEW" },
    { href: "/employee/support/tickets", label: "Ticket hỗ trợ", icon: "◇", permission: "TICKET_QUEUE_VIEW" }
  ] },
  { group: "Quản trị", items: [
    { href: "/employee/administration/employees", label: "Tài khoản nhân viên", icon: "♙", permission: "EMPLOYEE_ACCOUNT_VIEW" },
    { href: "/employee/administration/roles", label: "Role", icon: "◈", permission: "ROLE_VIEW" },
    { href: "/employee/administration/permissions", label: "Permission", icon: "⌘", permission: "PERMISSION_VIEW" }
  ] }
];
