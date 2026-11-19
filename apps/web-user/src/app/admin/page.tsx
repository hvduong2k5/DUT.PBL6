import type { Metadata } from "next";
import { EmployeeDashboard } from "@/features/employee/shared/employee-dashboard";

export const metadata: Metadata = { title: "Không gian nhân viên" };

export default function EmployeeHomePage() {
  return <EmployeeDashboard />;
}
