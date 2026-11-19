import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { OrderWorkspace } from "@/features/employee/orders/order-workspace";

export const metadata: Metadata = { title: "Quản lý đơn hàng" };

export default function OrderPage() { return <EmployeePermissionGate permission="ORDER_VIEW"><OrderWorkspace /></EmployeePermissionGate>; }
