import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { PackingWorkbench } from "@/features/employee/packing/packing-workbench";

export const metadata: Metadata = { title: "Workbench đóng gói" };

export default function PackingWorkbenchPage() { return <EmployeePermissionGate permission="PACKING_WORKBENCH_VIEW"><PackingWorkbench /></EmployeePermissionGate>; }
