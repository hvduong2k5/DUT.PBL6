import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { InventoryWorkspace } from "@/features/employee/inventory/inventory-workspace";

export const metadata: Metadata = { title: "Quản lý Tồn kho & Batch" };

export default function InventoryPage() { return <EmployeePermissionGate permission="INVENTORY_MANAGEMENT_VIEW"><InventoryWorkspace /></EmployeePermissionGate>; }
