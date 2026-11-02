import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { WarehouseWorkbench } from "@/features/employee/inventory/warehouse-workbench";

export const metadata: Metadata = { title: "Warehouse Operations Workbench" };

export default function WarehouseWorkbenchPage() { return <EmployeePermissionGate permission="WAREHOUSE_WORKBENCH_VIEW"><WarehouseWorkbench /></EmployeePermissionGate>; }
