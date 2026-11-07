import type { ReactNode } from "react";
import { EmployeeAccessBoundary } from "@/components/auth/employee-access-boundary";
import { EmployeeSessionProvider } from "@/components/auth/employee-session-provider";
import { EmployeeShell } from "@/components/layout/employee-shell";
import { usesIdentityBff } from "@/lib/auth/oidc-config";
import "@/styles/employee.css";

export default function EmployeeLayout({ children }: { children: ReactNode }) {
  return <EmployeeAccessBoundary><EmployeeSessionProvider><EmployeeShell fulfillmentEnabled={usesIdentityBff()}>{children}</EmployeeShell></EmployeeSessionProvider></EmployeeAccessBoundary>;
}
