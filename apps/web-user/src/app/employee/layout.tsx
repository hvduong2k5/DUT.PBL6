import type { ReactNode } from "react";
import { EmployeeAccessBoundary } from "@/components/auth/employee-access-boundary";
import { EmployeeSessionProvider } from "@/components/auth/employee-session-provider";
import { EmployeeShell } from "@/components/layout/employee-shell";

export default function EmployeeLayout({ children }: { children: ReactNode }) {
  return <EmployeeAccessBoundary><EmployeeSessionProvider><EmployeeShell>{children}</EmployeeShell></EmployeeSessionProvider></EmployeeAccessBoundary>;
}
