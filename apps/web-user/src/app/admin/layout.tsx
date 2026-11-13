import type { ReactNode } from "react";
import { connection } from "next/server";
import { EmployeeAccessBoundary } from "@/components/auth/employee-access-boundary";
import { EmployeeSessionProvider } from "@/components/auth/employee-session-provider";
import { EmployeeShell } from "@/components/layout/employee-shell";
import "@/styles/employee.css";

export default async function EmployeeLayout({ children }: { children: ReactNode }) {
  await connection();
  return <EmployeeAccessBoundary><EmployeeSessionProvider><EmployeeShell fulfillmentEnabled>{children}</EmployeeShell></EmployeeSessionProvider></EmployeeAccessBoundary>;
}
