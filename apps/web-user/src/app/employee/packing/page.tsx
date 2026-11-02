import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { PackingWorkspace } from "@/features/employee/packing/packing-workspace";

export const metadata: Metadata = { title: "Quản lý đóng gói" };
export default function PackingPage() { return <EmployeePermissionGate permission="PACKING_MANAGEMENT_VIEW"><PackingWorkspace /></EmployeePermissionGate>; }
