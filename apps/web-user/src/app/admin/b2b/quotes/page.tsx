import type { Metadata } from "next";
import { Suspense } from "react";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { B2BQuoteWorkspace } from "@/features/employee/b2b/quote-workspace";

export const metadata: Metadata = { title: "Quản lý báo giá B2B" };

export default function B2BQuoteWorkspacePage() {
  return <EmployeePermissionGate permission="B2B_REQUEST_VIEW"><Suspense fallback={null}><B2BQuoteWorkspace /></Suspense></EmployeePermissionGate>;
}
