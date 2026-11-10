import type { ReactNode } from "react";
import { connection } from "next/server";
import { EmployeeAccessBoundary } from "@/components/auth/employee-access-boundary";
import { EmployeeSessionProvider } from "@/components/auth/employee-session-provider";
import { EmployeeShell } from "@/components/layout/employee-shell";
import { usesIdentityBff } from "@/lib/auth/oidc-config";
import "@/styles/employee.css";

export default async function EmployeeLayout({ children }: { children: ReactNode }) {
  await connection();
  return <EmployeeAccessBoundary><EmployeeSessionProvider><EmployeeShell analyticsEnabled={!!process.env.ANALYTICS_ADMIN_UPSTREAM_URL} fulfillmentEnabled={usesIdentityBff()}>{children}</EmployeeShell></EmployeeSessionProvider></EmployeeAccessBoundary>;
}
