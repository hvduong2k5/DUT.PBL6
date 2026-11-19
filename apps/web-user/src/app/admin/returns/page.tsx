import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { ReturnWorkspace } from "@/features/employee/returns/return-workspace";

export const metadata: Metadata = { title: "Quản lý đổi trả và hoàn tiền" };
export default function ReturnPage() { return <EmployeePermissionGate permission="RETURN_CASE_VIEW"><ReturnWorkspace /></EmployeePermissionGate>; }
