import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { EmployeeAccessWorkspace } from "@/features/employee/access/employee-access-workspace";

export const metadata: Metadata = { title: "Tài khoản và quyền nhân viên" };

export default function EmployeeAccessPage() {
  return <EmployeePermissionGate permission="EMPLOYEE_ACCOUNT_VIEW"><EmployeeAccessWorkspace /></EmployeePermissionGate>;
}
