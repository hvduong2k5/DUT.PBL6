import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { RoleCatalogWorkspace } from "@/features/employee/access/role-catalog-workspace";

export const metadata: Metadata = { title: "Quản lý Role" };

export default function RoleCatalogPage() { return <EmployeePermissionGate permission="ROLE_VIEW"><RoleCatalogWorkspace /></EmployeePermissionGate>; }
