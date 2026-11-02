import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { PermissionCatalogWorkspace } from "@/features/employee/access/permission-catalog-workspace";

export const metadata: Metadata = { title: "Danh mục Permission" };

export default function PermissionCatalogPage() { return <EmployeePermissionGate permission="PERMISSION_VIEW"><PermissionCatalogWorkspace /></EmployeePermissionGate>; }
