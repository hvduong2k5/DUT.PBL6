import type { Metadata } from "next";
import { EmployeeAccessWorkspace } from "./employee-access-workspace";

export const metadata: Metadata = { title: "Tài khoản và quyền nhân viên" };

export default function EmployeeAccessPage() {
  return <EmployeeAccessWorkspace />;
}
